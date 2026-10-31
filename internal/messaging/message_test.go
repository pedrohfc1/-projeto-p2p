package messaging

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"strings"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	other, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	to, _ := peer.IDFromPrivateKey(other)
	m, e := New(k, to, "olá")
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Validate(id, to, k.GetPublic(), time.Now()); e != nil {
		t.Fatal(e)
	}
	m.Text = "alterado"
	if m.Validate(id, to, k.GetPublic(), time.Now()) == nil {
		t.Fatal("tampering accepted")
	}
	m, _ = New(k, to, "ok")
	if m.Validate(id, to, k.GetPublic(), time.Now().Add(3*time.Minute)) == nil {
		t.Fatal("expired accepted")
	}
	if m.Validate(to, id, other.GetPublic(), time.Now()) == nil {
		t.Fatal("wrong sender accepted")
	}
	if _, e = New(k, to, strings.Repeat("x", MaxText+1)); e == nil {
		t.Fatal("oversize accepted")
	}
}
