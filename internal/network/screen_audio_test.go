package network

import (
	"context"
	"io"
	"projetop2p/internal/voice"
	"testing"
	"time"
)

func TestScreenAudioUsesIndependentAuthenticatedLibp2pProtocol(t *testing.T) {
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer b.Close()
	received := make(chan string, 1)
	b.SetScreenAudioHandler(func(remote string, r voice.Request, c voice.MediaConn) error {
		if remote != a.SelfID() || r.Method != "screen-audio" {
			t.Error("identity/protocol mismatch")
		}
		go func() {
			buf := make([]byte, 3)
			_, e := io.ReadFull(c, buf)
			if e == nil {
				_, _ = c.Write(buf)
				received <- string(buf)
			}
			c.Close()
		}()
		return nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, e := a.OpenScreenAudioMedia(ctx, b.SelfID(), b.Addresses(), voice.Request{Method: "screen-audio", Nonce: "01234567890123456789012345678901"})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if _, e = c.Write([]byte{1, 2, 3}); e != nil {
		t.Fatal(e)
	}
	buf := make([]byte, 3)
	if _, e = io.ReadFull(c, buf); e != nil || buf[0] != 1 {
		t.Fatal("independent duplex stream failed", e)
	}
	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if ScreenAudioProtocol == ScreenMediaProtocol || ScreenAudioProtocol == VoiceMediaProtocol {
		t.Fatal("media protocols were conflated")
	}
}
