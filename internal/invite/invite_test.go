package invite

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"testing"
	"time"
)

func TestHamachiInviteScopeSigned(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	s, e := NewScoped(k, "Peer", []string{"/ip4/25.10.20.30/tcp/4001/p2p/" + id.String()}, "hamachi")
	if e != nil {
		t.Fatal(e)
	}
	i, e := Parse(s, time.Now())
	if e != nil || i.Scope != "hamachi" || len(i.Relays) > 0 {
		t.Fatal(i, e)
	}
	i.Scope = ""
	b, _ := json.Marshal(i)
	if _, e = Parse(Prefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("scope tampering accepted")
	}
}

func TestRadminInviteScope(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	s, e := NewScoped(k, "Radmin peer", []string{"/ip4/26.10.20.30/tcp/4001/p2p/" + id.String()}, "radmin")
	if e != nil {
		t.Fatal(e)
	}
	i, e := Parse(s, time.Now())
	if e != nil || i.Scope != "radmin" || len(i.Relays) != 0 || i.ID != id.String() {
		t.Fatal(i, e)
	}
	i.Scope = "hamachi"
	b, _ := json.Marshal(i)
	if _, e = Parse(Prefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("provider scope tampering accepted")
	}
}

func TestSignedInvite(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	s, e := New(k, "Alice", nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	i, e := Parse(s, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	i.Name = "Mallory"
	b, _ := json.Marshal(i)
	if _, e = Parse(Prefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("tampering accepted")
	}
	if _, e = Parse(s, time.Now().Add(8*24*time.Hour)); e == nil {
		t.Fatal("expired accepted")
	}
}

func TestRadminInviteRejectsOtherInterfacesAndDynamicListener(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	for _, prefix := range []string{"/ip4/169.254.4.5/tcp/4001", "/ip4/192.168.1.2/tcp/4001", "/ip4/10.5.0.2/tcp/4001", "/ip4/25.1.2.3/tcp/4001", "/ip4/26.1.2.3/tcp/0"} {
		if _, e := NewScoped(k, "Peer", []string{prefix + "/p2p/" + id.String()}, "radmin"); e == nil {
			t.Fatal("Radmin invitation accepted wrong endpoint", prefix)
		}
	}
}
