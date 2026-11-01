package desktop

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/community"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentInviteQuotaOwnerSerializesAdmissions(t *testing.T) {
	owner := communityCore(t, t.TempDir(), "Authority", freePort(t))
	defer owner.Close()
	group, e := owner.CreateCommunity("Quota race", "", "")
	if e != nil {
		t.Fatal(e)
	}
	raw, e := owner.CommunityInvitePolicy(group, community.InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 24})
	if e != nil {
		t.Fatal(e)
	}
	legacy, e := community.LegacyInvite(raw)
	if e != nil {
		t.Fatal(e)
	}
	record, e := owner.db.Community(group)
	if e != nil {
		t.Fatal(e)
	}
	var admitted atomic.Int32
	var wg sync.WaitGroup
	for n := 0; n < 20; n++ {
		key, _, e := crypto.GenerateEd25519Key(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := peer.IDFromPrivateKey(key)
		op, e := community.SignOperation(key, group, record.State.Head, community.Join, community.Body{Name: "Concurrent", Invite: legacy})
		if e != nil {
			t.Fatal(e)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := owner.groups.Handle(owner.ctx, id.String(), community.Request{Method: "submit", CommunityID: group, Operation: &op})
			if res.Error == "" {
				admitted.Add(1)
			}
		}()
	}
	wg.Wait()
	if admitted.Load() != 5 {
		t.Fatal("global authoritative quota", admitted.Load())
	}
	info, e := owner.CommunityInviteStatus(group, raw)
	if e != nil || info.Used != 5 || info.Remaining != 0 {
		t.Fatal(info, e)
	}
	receipt, e := owner.CommunityAction(group, community.InviteRevoke, community.Body{Invite: info.Nonce})
	if e != nil || receipt.Status != "confirmada" {
		t.Fatal(receipt, e)
	}
	info, e = owner.CommunityInviteStatus(group, raw)
	if e != nil || !info.Revoked {
		t.Fatal("revocation status", info, e)
	}
}
