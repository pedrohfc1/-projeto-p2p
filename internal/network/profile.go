package network

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/sync/singleflight"
	"projetop2p/internal/profile"
	"projetop2p/internal/security"
)

const ProfileProtocol = protocol.ID("/projeto-p2p/profile/1.0.0")
const profileTimeout = 6 * time.Second
const profileFrameLimit = 2048
const profileWorkers = 4

type profileRequest struct {
	Kind     string `json:"kind"`
	Known    uint64 `json:"known,omitempty"`
	Revision uint64 `json:"revision,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

type profileResponse struct {
	Profile *profile.SignedMetadata `json:"profile,omitempty"`
	Size    int                     `json:"size,omitempty"`
	MIME    string                  `json:"mime,omitempty"`
}

type profileSync struct {
	node          *Node
	store         *profile.Store
	authorizeIcon func(peerID, hash string) bool
	changed       chan struct{}
	updates       chan peer.ID
	fetchSlots    chan struct{}
	inFlight      singleflight.Group
}

func (n *Node) SetProfileStore(store *profile.Store, authorizeArtwork func(peerID, hash string) bool) {
	if store == nil {
		return
	}
	n.mu.Lock()
	if n.profiles != nil || n.ctx.Err() != nil {
		n.mu.Unlock()
		return
	}
	syncer := &profileSync{
		node: n, store: store, authorizeIcon: authorizeArtwork,
		changed: make(chan struct{}, 1), updates: make(chan peer.ID, 128),
		fetchSlots: make(chan struct{}, profileWorkers),
	}
	n.profiles = syncer
	gate := &security.StreamGate{}
	n.Host.SetStreamHandler(ProfileProtocol, func(s libnet.Stream) {
		defer s.Close()
		release, ok := gate.Acquire(s.Conn().RemotePeer().String(), time.Now())
		if !ok {
			_ = s.Reset()
			return
		}
		defer release()
		ctx, cancel := context.WithTimeout(n.ctx, profileTimeout)
		defer cancel()
		stop := context.AfterFunc(ctx, func() { _ = s.Reset() })
		defer stop()
		_ = s.SetDeadline(time.Now().Add(profileTimeout))
		if !syncer.serve(s) {
			_ = s.Reset()
		}
	})
	n.wg.Add(1)
	go syncer.run()
	n.mu.Unlock()
}

func (n *Node) ProfileChanged() {
	n.mu.Lock()
	syncer := n.profiles
	n.mu.Unlock()
	if syncer != nil {
		select {
		case syncer.changed <- struct{}{}:
		default:
		}
	}
}

func (n *Node) FetchProfileAsset(ctx context.Context, sourceID, hash string) (profile.Asset, error) {
	n.mu.Lock()
	syncer := n.profiles
	n.mu.Unlock()
	if syncer == nil {
		return profile.Asset{}, errors.New("perfil indisponível na rede")
	}
	if !profile.ValidHash(hash) {
		return profile.Asset{}, errors.New("invalid profile asset hash")
	}
	id, err := peer.Decode(sourceID)
	if err != nil {
		return profile.Asset{}, err
	}
	return syncer.fetch(ctx, id, hash, "")
}

func readProfileFrame(r io.Reader, value any) error {
	var prefix [4]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(prefix[:])
	if size == 0 || size > profileFrameLimit {
		return errors.New("invalid profile frame size")
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func writeProfileFrame(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > profileFrameLimit {
		return errors.New("profile frame too large")
	}
	frame := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	frame = append(frame, data...)
	return writeProfileBytes(w, frame)
}

func writeProfileBytes(w io.Writer, data []byte) error {
	written, err := w.Write(data)
	if err == nil && written != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func (p *profileSync) serve(stream libnet.Stream) bool {
	remote := stream.Conn().RemotePeer()
	var req profileRequest
	if err := readProfileFrame(stream, &req); err != nil {
		return false
	}
	switch req.Kind {
	case "profile":
		known, _ := p.store.Get(remote.String())
		if req.Revision > known.Revision {
			select {
			case p.updates <- remote:
			default:
			}
		}
		var response profileResponse
		local := p.store.Signed()
		if local.Revision > req.Known {
			response.Profile = &local
		}
		return writeProfileFrame(stream, response) == nil
	case "asset":
		if !profile.ValidHash(req.Hash) {
			return false
		}
		owns := p.store.Local().AvatarHash == req.Hash
		if !owns && (p.authorizeIcon == nil || !p.authorizeIcon(remote.String(), req.Hash)) {
			return writeProfileFrame(stream, profileResponse{}) == nil
		}
		asset, data, err := p.store.ReadImage(req.Hash)
		if err != nil {
			return writeProfileFrame(stream, profileResponse{}) == nil
		}
		return writeProfileFrame(stream, profileResponse{Size: len(data), MIME: asset.MIME}) == nil && writeProfileBytes(stream, data) == nil
	default:
		return false
	}
}

type profileCursor struct {
	last     time.Time
	next     time.Time
	revision uint64
	busy     bool
}

// Only established authenticated connections are used. Old clients continue to
// use the other libp2p protocols and have no profile exchange handler.
func (p *profileSync) run() {
	defer p.node.wg.Done()
	ctx := p.node.ctx
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	cursors := make(map[peer.ID]*profileCursor)
	type outcome struct {
		id    peer.ID
		retry bool
	}
	results := make(chan outcome, profileWorkers)
	active := 0
	for {
		peers := p.node.Host.Network().Peers()
		connected := make(map[peer.ID]bool, len(peers))
		for _, id := range peers {
			connected[id] = true
			if cursors[id] == nil {
				cursors[id] = &profileCursor{}
			}
		}
		for id, cursor := range cursors {
			if !connected[id] && !cursor.busy {
				delete(cursors, id)
			}
		}
		now, local := time.Now(), p.store.Local()
		for _, id := range peers {
			cursor := cursors[id]
			if active >= profileWorkers {
				break
			}
			if cursor.busy || now.Sub(cursor.last) < time.Second || (now.Before(cursor.next) && cursor.revision == local.Revision) {
				continue
			}
			cursor.busy, cursor.last, cursor.next, cursor.revision = true, now, now.Add(30*time.Second), local.Revision
			active++
			go func() {
				results <- outcome{id: id, retry: p.exchange(ctx, id, local.Revision)}
			}()
		}
		select {
		case <-ctx.Done():
			for active > 0 {
				<-results
				active--
			}
			return
		case <-ticker.C:
		case <-p.changed:
		case id := <-p.updates:
			if cursor := cursors[id]; cursor != nil {
				// Keep the one-second cooldown even under repeat notifications.
				cursor.next = time.Time{}
			}
		case result := <-results:
			cursor := cursors[result.id]
			cursor.busy = false
			if retryAt := time.Now().Add(3 * time.Second); result.retry && cursor.next.After(retryAt) {
				cursor.next = retryAt
			}
			active--
		}
	}
}

func (p *profileSync) exchange(ctx context.Context, id peer.ID, revision uint64) (retry bool) {
	remote := id.String()
	if !p.store.CanReceive(remote) {
		return false
	}
	known, _ := p.store.Get(remote)
	var response profileResponse
	err := p.request(ctx, id, profileRequest{Kind: "profile", Known: known.Revision, Revision: revision}, &response, nil)
	if err != nil {
		return false
	}
	if response.Profile == nil {
		// A local cache can be lost independently of the authenticated revision.
		if known.AvatarHash != "" && !p.store.HasImage(known.AvatarHash) {
			_, err := p.fetch(ctx, id, known.AvatarHash, known.AvatarMIME)
			return err != nil
		}
		return false
	}
	next := *response.Profile
	if next.Revision <= known.Revision || profile.Verify(remote, next) != nil {
		return false
	}
	if next.AvatarHash != "" {
		if _, err = p.fetch(ctx, id, next.AvatarHash, next.AvatarMIME); err != nil {
			return true
		}
	}
	return p.store.Accept(remote, next) != nil
}

func (p *profileSync) fetch(ctx context.Context, id peer.ID, hash, expectedMIME string) (profile.Asset, error) {
	if asset, _, err := p.store.ReadImage(hash); err == nil && (expectedMIME == "" || expectedMIME == asset.MIME) {
		return asset, nil
	}
	result := p.inFlight.DoChan(hash, func() (any, error) {
		select {
		case p.fetchSlots <- struct{}{}:
		case <-ctx.Done():
			return profile.Asset{}, ctx.Err()
		case <-p.node.ctx.Done():
			return profile.Asset{}, p.node.ctx.Err()
		}
		defer func() { <-p.fetchSlots }()
		slog.Info("avatar_asset_requested")
		var response profileResponse
		var data []byte
		if err := p.request(ctx, id, profileRequest{Kind: "asset", Hash: hash}, &response, &data); err != nil {
			return profile.Asset{}, err
		}
		asset, err := p.store.PutExpectedImage(hash, response.MIME, data)
		if errors.Is(err, profile.ErrHashMismatch) {
			slog.Warn("avatar_hash_mismatch")
		}
		if err == nil {
			slog.Info("avatar_asset_received", "size", len(data))
		}
		return asset, err
	})
	select {
	case <-ctx.Done():
		return profile.Asset{}, ctx.Err()
	case <-p.node.ctx.Done():
		return profile.Asset{}, p.node.ctx.Err()
	case entry := <-result:
		if entry.Err != nil {
			return profile.Asset{}, entry.Err
		}
		asset := entry.Val.(profile.Asset)
		if expectedMIME != "" && asset.MIME != expectedMIME {
			return profile.Asset{}, errors.New("profile image MIME mismatch")
		}
		return asset, nil
	}
}

func (p *profileSync) request(parent context.Context, id peer.ID, req profileRequest, response *profileResponse, image *[]byte) error {
	ctx, cancel := context.WithTimeout(parent, profileTimeout)
	defer cancel()
	stopNode := context.AfterFunc(p.node.ctx, cancel)
	defer stopNode()
	ctx = libnet.WithNoDial(libnet.WithAllowLimitedConn(ctx, "authorized profile relay"), "no independent profile dial")
	stream, err := p.node.Host.NewStream(ctx, id, ProfileProtocol)
	if err != nil {
		return err
	}
	defer stream.Close()
	stop := context.AfterFunc(ctx, func() { _ = stream.Reset() })
	defer stop()
	deadline, _ := ctx.Deadline()
	_ = stream.SetDeadline(deadline)
	if err = writeProfileFrame(stream, req); err != nil {
		return err
	}
	if err = readProfileFrame(stream, response); err != nil {
		return err
	}
	if image != nil {
		if response.Size <= 0 || response.Size > profile.MaxImageBytes {
			return errors.New("profile image is unavailable or exceeds its limit")
		}
		*image = make([]byte, response.Size)
		_, err = io.ReadFull(stream, *image)
	}
	return err
}
