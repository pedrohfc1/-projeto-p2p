package voice

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"sync"
	"testing"
	"time"
)

func testKey(t *testing.T) crypto.PrivKey {
	k, _, e := crypto.GenerateEd25519Key(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	return k
}
func TestRoomAuthorityAuthenticationQuotaAndRevocation(t *testing.T) {
	owner := testKey(t)
	op, _ := community.SignOperation(owner, "", "", community.Create, community.Body{Name: "Voice security"})
	_, state, e := community.MakeCommit(owner, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	keys := []crypto.PrivKey{owner, testKey(t), testKey(t)}
	people := []Participant{}
	for n, key := range keys {
		id, _ := peer.IDFromPrivateKey(key)
		if n > 0 {
			raw, e := community.NewInvite(owner, state, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + state.Owner})
			if e != nil {
				t.Fatal(e)
			}
			op, _ := community.SignOperation(key, state.ID, state.Head, community.Join, community.Body{Name: "member", Invite: raw})
			_, state, e = community.MakeCommit(owner, state, op)
			if e != nil {
				t.Fatal(e)
			}
		}
		raw, e := invite.NewScoped(key, "member", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + id.String()}, "interface")
		if e != nil {
			t.Fatal(e)
		}
		people = append(people, Participant{ID: id.String(), Session: community.RandomID(), Invite: raw, UDPPort: 4002})
	}
	var mu sync.Mutex
	authority := NewAuthority(owner, func(id string) (community.State, error) { mu.Lock(); defer mu.Unlock(); return state, nil })
	request := func(p Participant) Request {
		return Request{Method: "join", Community: state.ID, Channel: "voz", Participant: p}
	}
	if res := authority.Handle(people[1].ID, request(people[0])); res.Error == "" {
		t.Fatal("transport identity spoof accepted")
	}
	if res := authority.Handle(people[0].ID, request(people[0])); res.Error != "" {
		t.Fatal(res.Error)
	}
	lease := authority.Handle(people[1].ID, request(people[1]))
	if lease.Error != "" || lease.Lease == nil {
		t.Fatal(lease)
	}
	if e = VerifyLease(*lease.Lease, state, "voz", time.Now()); e != nil {
		t.Fatal(e)
	}
	if res := authority.Handle(people[2].ID, request(people[2])); res.Error == "" {
		t.Fatal("third person bypassed v0.6.0 media quota")
	}
	forged := *lease.Lease
	forged.Signature = append([]byte(nil), forged.Signature...)
	forged.Signature[0] ^= 1
	if VerifyLease(forged, state, "voz", time.Now()) == nil {
		t.Fatal("forged authority")
	}
	if VerifyLease(*lease.Lease, state, "voz", time.Now().Add(time.Minute)) == nil {
		t.Fatal("expired lease accepted")
	}
	op, _ = community.SignOperation(owner, state.ID, state.Head, community.Remove, community.Body{Target: people[1].ID})
	_, state, e = community.MakeCommit(owner, state, op)
	if e != nil {
		t.Fatal(e)
	}
	if VerifyLease(*lease.Lease, state, "voz", time.Now()) == nil {
		t.Fatal("known removed participant authorized by old lease")
	}
	if res := authority.Handle(people[1].ID, request(people[1])); res.Error == "" {
		t.Fatal("removed participant rejoined")
	}
}
