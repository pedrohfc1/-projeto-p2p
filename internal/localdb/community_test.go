package localdb

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/community"
	"testing"
)

func TestCommunityForkDetectionQuotaAndAtomicity(t *testing.T) {
	d, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	op, _ := community.SignOperation(key, "", "", community.Create, community.Body{Name: "Group"})
	root, s, e := community.MakeCommit(key, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	r := CommunityRecord{ID: s.ID, Owner: s.Owner, Root: s.Root, Name: s.Name}
	if e = d.CreateOwnedCommunity(r, root); e != nil {
		t.Fatal(e)
	}
	a, _ := community.SignOperation(key, s.ID, s.Head, community.Text, community.Body{ChannelID: "geral", Text: "first"})
	b, _ := community.SignOperation(key, s.ID, s.Head, community.Text, community.Body{ChannelID: "geral", Text: "fork"})
	ca, _, _ := community.MakeCommit(key, s, a)
	cb, _, _ := community.MakeCommit(key, s, b)
	if ok, e := d.CommunityCommit(ca); e != nil || !ok {
		t.Fatal(e)
	}
	if _, e = d.CommunityCommit(cb); e == nil {
		t.Fatal("owner-signed fork replaced existing sequence")
	}
	if ok, e := d.CommunityCommit(ca); e != nil || ok {
		t.Fatal("duplicate did not remain idempotent", e)
	}
	before, _ := d.Community(s.ID)
	_ = d.db.Update(func(tx *bolt.Tx) error {
		x, e := getRecord(tx, s.ID)
		if e != nil {
			return e
		}
		x.LogBytes = maxCommunityLog
		return putRecord(tx, x)
	})
	more, _ := community.SignOperation(key, s.ID, before.State.Head, community.Text, community.Body{ChannelID: "geral", Text: "quota"})
	commit, _, _ := community.MakeCommit(key, before.State, more)
	if _, e = d.CommunityCommit(commit); e == nil {
		t.Fatal("log quota ignored")
	}
	after, _ := d.Community(s.ID)
	if after.State.Head != before.State.Head {
		t.Fatal("failed quota append advanced state")
	}
	if seq, _ := d.FindCommunityOperation(s.ID, more.ID); seq != 0 {
		t.Fatal("failed append leaked into operation index")
	}
}
