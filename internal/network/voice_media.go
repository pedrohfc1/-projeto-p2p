package network

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"io"
	"log/slog"
	"projetop2p/internal/security"
	"projetop2p/internal/voice"
	"sync"
	"time"
)

const VoiceMediaProtocol = protocol.ID("/projeto-p2p/voice-media/1.0.0")
const ScreenAudioProtocol = protocol.ID("/peetwopee/screen-audio/1.0.0")
const ScreenMediaProtocol = protocol.ID("/peetwopee/screen-media/1.0.0")

type mediaConn struct {
	diagnosticEnd     func(error, bool)
	diagnosticFailure sync.Once
	libnet.Stream
	reader      *bufio.Reader
	ready, done chan struct{}
	once        sync.Once
	flow        string
}

func (c *mediaConn) Read(b []byte) (int, error) {
	select {
	case <-c.ready:
	case <-c.done:
		return 0, errors.New("closed")
	}
	n, e := c.reader.Read(b)
	if e != nil && c.diagnosticEnd != nil {
		select {
		case <-c.done:
		default:
			c.diagnosticFailure.Do(func() { c.diagnosticEnd(e, false) })
		}
	}
	return n, e
}
func (c *mediaConn) Write(b []byte) (int, error) {
	select {
	case <-c.ready:
	case <-c.done:
		return 0, errors.New("closed")
	}
	n, e := c.Stream.Write(b)
	if e != nil && c.diagnosticEnd != nil {
		select {
		case <-c.done:
		default:
			c.diagnosticFailure.Do(func() { c.diagnosticEnd(e, false) })
		}
	}
	return n, e
}
func (c *mediaConn) Close() error {
	c.once.Do(func() {
		slog.Info("voice_transport", "flow", c.flow, "component", "libp2p-media", "origin", "local", "reason", "consumer_close_reset", "stream", c.Stream.ID())
		close(c.done)
		if c.diagnosticEnd != nil {
			c.diagnosticEnd(nil, true)
		}
		c.Stream.Reset()
	})
	return nil
}
func mediaFlow(nonce string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(nonce)))[:12] }
func (n *Node) SetVoiceMediaHandler(handler func(string, voice.Request, voice.MediaConn) error) {
	n.setMediaHandler(VoiceMediaProtocol, handler)
}
func (n *Node) SetScreenMediaHandler(handler func(string, voice.Request, voice.MediaConn) error) {
	n.setMediaHandler(ScreenMediaProtocol, handler)
}
func (n *Node) setMediaHandler(mediaProtocol protocol.ID, handler func(string, voice.Request, voice.MediaConn) error) {
	if handler == nil {
		n.Host.RemoveStreamHandler(mediaProtocol)
		return
	}
	gate := &security.StreamGate{}
	n.Host.SetStreamHandler(mediaProtocol, func(stream libnet.Stream) {
		c := &mediaConn{diagnosticEnd: func(e error, local bool) { n.observeMediaEnd(stream.Conn().RemotePeer(), e, local) }, Stream: stream, reader: bufio.NewReader(stream), ready: make(chan struct{}), done: make(chan struct{})}
		defer c.Close()
		release, ok := gate.Acquire(stream.Conn().RemotePeer().String(), time.Now())
		if !ok {
			return
		}
		defer release()
		stream.SetDeadline(time.Now().Add(10 * time.Second))
		var req voice.Request
		if readVoiceMediaFrame(c.reader, maxVoiceFrame, &req) != nil {
			return
		}
		c.flow = mediaFlow(req.Nonce)
		slog.Info("voice_transport", "flow", c.flow, "source_session", mediaFlow(req.Participant.Session), "target_session", mediaFlow(req.Session), "origin", "remote", "state", "handshake_received", "stream", stream.ID())
		if e := handler(stream.Conn().RemotePeer().String(), req, c); e != nil {
			slog.Warn("voice_transport", "flow", c.flow, "origin", "local", "reason", e.Error(), "state", "authorization_rejected")
			_ = writeCommunityFrame(stream, 1024, voice.Response{Error: e.Error()})
			return
		}
		if writeCommunityFrame(stream, 1024, voice.Response{}) != nil {
			return
		}
		stream.SetDeadline(time.Time{})
		close(c.ready)
		select {
		case <-c.done:
		case <-n.ctx.Done():
		}
	})
}
func (n *Node) OpenVoiceMedia(ctx context.Context, id string, addresses []string, req voice.Request) (voice.MediaConn, error) {
	return n.openMedia(ctx, id, addresses, req, VoiceMediaProtocol)
}
func (n *Node) OpenScreenMedia(ctx context.Context, id string, addresses []string, req voice.Request) (voice.MediaConn, error) {
	return n.openMedia(ctx, id, addresses, req, ScreenMediaProtocol)
}
func (n *Node) openMedia(ctx context.Context, id string, addresses []string, req voice.Request, mediaProtocol protocol.ID) (voice.MediaConn, error) {
	pid, e := peer.Decode(id)
	if e != nil {
		return nil, e
	}
	if len(addresses) > 16 {
		return nil, errors.New("too many media endpoints")
	}
	info := peer.AddrInfo{ID: pid}
	for _, raw := range addresses {
		a, e := ParseAddress(raw)
		if e != nil || a.ID != pid {
			return nil, errors.New("media identity mismatch")
		}
		info.Addrs = append(info.Addrs, a.Addrs...)
	}
	if e = n.Connect(ctx, info); e != nil {
		return nil, e
	}
	openAt := time.Now()
	stream, e := n.Host.NewStream(libnet.WithAllowLimitedConn(ctx, "authenticated Opus media"), pid, mediaProtocol)
	n.observeMediaOpen(pid, time.Since(openAt), e)
	if e != nil {
		n.failedVoiceStream(pid, e)
		return nil, e
	}
	ok := false
	defer func() {
		if !ok {
			slog.Warn("voice_transport", "flow", mediaFlow(req.Nonce), "origin", "local", "reason", "handshake_not_committed", "stream", stream.ID())
			stream.Reset()
		}
	}()
	stop := context.AfterFunc(ctx, func() {
		slog.Warn("voice_transport", "flow", mediaFlow(req.Nonce), "origin", "local", "reason", "handshake_context_cancelled", "stream", stream.ID())
		stream.Reset()
	})
	defer stop()
	stream.SetDeadline(time.Now().Add(10 * time.Second))
	if e = writeCommunityFrame(stream, maxVoiceFrame, req); e != nil {
		return nil, e
	}
	var res voice.Response
	reader := bufio.NewReader(stream)
	if e = readVoiceMediaFrame(reader, 1024, &res); e != nil {
		return nil, e
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	// Disarm cancellation before ownership transfers to the long-lived media session.
	if !stop() || ctx.Err() != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, context.Canceled
	}
	stream.SetDeadline(time.Time{})
	slog.Info("voice_transport", "flow", mediaFlow(req.Nonce), "source_session", mediaFlow(req.Participant.Session), "target_session", mediaFlow(req.Session), "state", "handshake_accepted", "stream", stream.ID())
	c := &mediaConn{diagnosticEnd: func(e error, local bool) { n.observeMediaEnd(pid, e, local) }, flow: mediaFlow(req.Nonce), Stream: stream, reader: reader, ready: make(chan struct{}), done: make(chan struct{})}
	close(c.ready)
	ok = true
	return c, nil
}

// Preserve prefetched media bytes across the JSON-to-binary handshake boundary.
func readVoiceMediaFrame(r *bufio.Reader, max int, value any) error {
	b := make([]byte, 0, 1024)
	for len(b) < max {
		x, e := r.ReadByte()
		if e != nil {
			return e
		}
		b = append(b, x)
		if x == '\n' {
			d := json.NewDecoder(bytes.NewReader(b))
			d.DisallowUnknownFields()
			if e = d.Decode(value); e != nil {
				return e
			}
			var extra any
			if e = d.Decode(&extra); e != io.EOF {
				return errors.New("trailing media handshake data")
			}
			return nil
		}
	}
	return errors.New("media handshake exceeds limit")
}

func (c *mediaConn) SetReadDeadline(t time.Time) error {
	select {
	case <-c.ready:
		return c.Stream.SetReadDeadline(t)
	case <-c.done:
		return errors.New("media closed")
	}
}
func (c *mediaConn) SetWriteDeadline(t time.Time) error {
	select {
	case <-c.ready:
		return c.Stream.SetWriteDeadline(t)
	case <-c.done:
		return errors.New("media closed")
	}
}

func (n *Node) SetScreenAudioHandler(h func(string, voice.Request, voice.MediaConn) error) {
	n.setMediaHandler(ScreenAudioProtocol, h)
}
func (n *Node) OpenScreenAudioMedia(ctx context.Context, id string, addresses []string, r voice.Request) (voice.MediaConn, error) {
	return n.openMedia(ctx, id, addresses, r, ScreenAudioProtocol)
}
