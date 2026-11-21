package network

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"sync/atomic"
	"testing"
	"time"

	libnet "github.com/libp2p/go-libp2p/core/network"
	"projetop2p/internal/profile"
)

func attachProfile(t *testing.T, dir string, authorize func(string, string) bool) (*Node, *profile.Store) {
	t.Helper()
	node := create(t, Config{DataDir: dir, Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	store, err := profile.Open(dir, node.Host.Peerstore().PrivKey(node.Host.ID()), "Perfil legado")
	if err != nil {
		_ = node.Close()
		t.Fatal(err)
	}
	node.SetProfileStore(store, authorize)
	return node, store
}

func connectProfilePeers(t *testing.T, a, b *Node) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := ParseAddress(b.Addresses()[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Connect(ctx, info); err != nil {
		t.Fatal(err)
	}
}

func makeProfilePNG(t *testing.T, tint uint8) []byte {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, 32, 32))
	pixels.SetRGBA(2, 4, color.RGBA{R: tint, G: 4, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, pixels); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func publishProfile(t *testing.T, node *Node, store *profile.Store, name, font, hash string) profile.Metadata {
	t.Helper()
	m, err := store.Update(name, font, hash)
	if err != nil {
		t.Fatal(err)
	}
	node.ProfileChanged()
	return m
}

func awaitProfile(t *testing.T, store *profile.Store, expected profile.Metadata) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if got, ok := store.Get(expected.PeerID); ok && got == expected {
			return
		}
		time.Sleep(35 * time.Millisecond)
	}
	got, ok := store.Get(expected.PeerID)
	t.Fatalf("expected profile before deadline: got revision=%d exists=%v, want revision=%d", got.Revision, ok, expected.Revision)
}

func TestProfileSyncAvatarsNameFontRevisionsAndCacheAcrossRestarts(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	a, sa := attachProfile(t, dirA, nil)
	b, sb := attachProfile(t, dirB, nil)
	defer func() { _ = a.Close(); _ = b.Close() }()
	connectProfilePeers(t, a, b)
	awaitProfile(t, sb, sa.Local())
	awaitProfile(t, sa, sb.Local())

	firstBytes := makeProfilePNG(t, 75)
	first, err := sa.PutImage(firstBytes)
	if err != nil {
		t.Fatal(err)
	}
	v1 := publishProfile(t, a, sa, "Pedro ✨", "poppins", first.Hash)
	awaitProfile(t, sb, v1)
	if _, bytesB, err := sb.ReadImage(first.Hash); err != nil || !bytes.Equal(bytesB, firstBytes) {
		t.Fatal("the receiving peer did not cache the matching avatar", err)
	}

	var assetRequests atomic.Int32
	// Observe the real request stream for the next name-only update.
	observedStore := sa
	a.Host.SetStreamHandler(ProfileProtocol, func(stream libnet.Stream) {
		defer stream.Close()
		_ = stream.SetDeadline(time.Now().Add(profileTimeout))
		var request profileRequest
		if readProfileFrame(stream, &request) != nil {
			return
		}
		if request.Kind == "asset" {
			assetRequests.Add(1)
			return
		}
		var response profileResponse
		if signed := observedStore.Signed(); signed.Revision > request.Known {
			response.Profile = &signed
		}
		_ = writeProfileFrame(stream, response)
	})
	v2 := publishProfile(t, a, sa, "Pedro Café", "future-font", first.Hash)
	awaitProfile(t, sb, v2)
	if assetRequests.Load() != 0 {
		t.Fatal("unchanged avatar was downloaded again")
	}

	_ = b.Close()
	_ = a.Close()
	a, sa = attachProfile(t, dirA, nil)
	b, sb = attachProfile(t, dirB, nil)
	if sa.Local() != v2 {
		t.Fatal("publisher lost its profile or identity on restart")
	}
	awaitProfile(t, sb, v2) // Cache is usable before re-establishing the connection.
	if _, bytesB, err := sb.ReadImage(first.Hash); err != nil || !bytes.Equal(bytesB, firstBytes) {
		t.Fatal("receiver lost the cached avatar on restart", err)
	}
	connectProfilePeers(t, a, b)
	secondBytes := makeProfilePNG(t, 170)
	second, err := sa.PutImage(secondBytes)
	if err != nil {
		t.Fatal(err)
	}
	v3 := publishProfile(t, a, sa, "Pedro Café", "inter", second.Hash)
	awaitProfile(t, sb, v3)
	if _, bytesB, err := sb.ReadImage(second.Hash); err != nil || !bytes.Equal(bytesB, secondBytes) {
		t.Fatal("new avatar was not delivered", err)
	}

	c, sc := attachProfile(t, t.TempDir(), nil)
	defer c.Close()
	connectProfilePeers(t, c, a)
	awaitProfile(t, sc, v3)
	v4 := publishProfile(t, a, sa, "Pedro Café", "inter", "")
	awaitProfile(t, sb, v4)
	awaitProfile(t, sc, v4)
}

func TestProfileAssetAllowsAuthorizedMemberAndPreservesLegacyChat(t *testing.T) {
	b, sb := attachProfile(t, t.TempDir(), nil)
	defer b.Close()
	c, _ := attachProfile(t, t.TempDir(), nil)
	defer c.Close()
	var allowedHash atomic.Value
	allowedHash.Store("")
	a, sa := attachProfile(t, t.TempDir(), func(id, hash string) bool {
		return id == b.SelfID() && hash == allowedHash.Load().(string)
	})
	defer a.Close()
	asset, err := sa.PutImage(makeProfilePNG(t, 237))
	if err != nil {
		t.Fatal(err)
	}
	allowedHash.Store(asset.Hash)
	connectProfilePeers(t, b, a)
	connectProfilePeers(t, c, a)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if _, err := c.FetchProfileAsset(ctx, a.SelfID(), asset.Hash); err == nil {
		t.Fatal("unauthorized participant fetched the artwork")
	}
	if got, err := b.FetchProfileAsset(ctx, a.SelfID(), asset.Hash); err != nil || got != asset {
		t.Fatal(got, err)
	}
	if _, _, err := sb.ReadImage(asset.Hash); err != nil {
		t.Fatal(err)
	}
	legacy := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer legacy.Close()
	connectProfilePeers(t, a, legacy)
	if err := a.Send(ctx, legacy.Host.ID(), "legacy chat remains compatible"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-legacy.Messages:
		if message.Text != "legacy chat remains compatible" {
			t.Fatal(message)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
