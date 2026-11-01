package invitediscovery

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/community"
	"testing"
	"time"
)

func TestLocalReplicasAuthorizationTamperingExpiryRevocation(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	op, _ := community.SignOperation(k, "", "", community.Create, community.Body{Name: "Grupo"})
	_, s, e := community.MakeCommit(k, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := community.NewInvite(k, s, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + id.String()})
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	r, e := New(k, raw, time.Hour, now)
	if e != nil {
		t.Fatal(e)
	}
	other, _ := New(k, raw, time.Hour, now)
	if r.Code() == other.Code() {
		t.Fatal("nonce collision")
	}
	a, b := NewMirror(), NewMirror()
	if e = a.Put(r, s, "outsider", now); e == nil {
		t.Fatal("unauthorized replica")
	}
	for _, m := range []*Mirror{a, b} {
		if e = m.Put(r, s, id.String(), now); e != nil {
			t.Fatal(e)
		}
		if _, e = m.Resolve(r.Code(), now); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = b.Resolve(r.Code(), now.Add(2*time.Hour)); e == nil {
		t.Fatal("expired")
	}
	changed := r
	changed.Expires++
	if _, e = changed.Verify(r.Code(), now); e == nil {
		t.Fatal("substitution")
	}
	rev, _ := Revoke(k, r)
	for _, m := range []*Mirror{a, b} {
		if e = m.ApplyRevocation(rev); e != nil {
			t.Fatal(e)
		}
		if _, e = m.Resolve(r.Code(), now); e == nil {
			t.Fatal("revoked")
		}
		if e = m.Put(r, s, id.String(), now); e == nil {
			t.Fatal("replay resurrected")
		}
	}
}
