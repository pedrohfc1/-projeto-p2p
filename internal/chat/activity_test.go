package chat

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"testing"
	"time"
)

func TestSignedActivityAndTerminalDeletion(t *testing.T) {
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	other, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(other)
	e, err := Sign(key, Event{Room: Room{Peer: id.String()}, Kind: "edit", Target: "0123456789abcdef0123456789abcdef", Text: "edited", Nonce: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if err = Verify(e, time.Now()); err != nil {
		t.Fatal(err)
	}
	tampered := e
	tampered.Text = "forged"
	if Verify(tampered, time.Now()) == nil {
		t.Fatal("tamper accepted")
	}
	p := Overlay{ID: e.Target, Text: "original"}
	p = Apply(p, e)
	if p.Text != "edited" {
		t.Fatal(p)
	}
	del := e
	del.Kind = "delete"
	p = Apply(p, del)
	p = Apply(p, e)
	if !p.Deleted || p.Text != "" {
		t.Fatal("edit resurrected tombstone")
	}
	e.Kind = "typing"
	e.Target = ""
	e.Text = ""
	e, err = Sign(key, e)
	if err != nil {
		t.Fatal(err)
	}
	if Verify(e, time.Now().Add(8*time.Second)) == nil {
		t.Fatal("typing never expires")
	}
}
func TestReactionConvergence(t *testing.T) {
	a := Event{Actor: "peer", Kind: "reaction", Emoji: "👍", Enabled: true, At: 1, ID: "a"}
	b := a
	b.Enabled = false
	b.At = 2
	b.ID = "b"
	p := Apply(Apply(Overlay{}, a), b)
	q := Apply(Apply(Overlay{}, b), a)
	if len(Counts(p, "peer")) != 0 || len(Counts(q, "peer")) != 0 {
		t.Fatal("arrival order changed reaction state")
	}
}
