package desktop

import (
	"projetop2p/internal/community"
	"testing"
)

func TestConsumedInviteIsNotReportedAsOwnerDenial(t *testing.T) {
	owner := communityCore(t, t.TempDir(), "owner", freePort(t))
	defer owner.Close()
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	group, e := owner.CreateCommunity("Diagnostic", "", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := owner.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := a.JoinCommunity(ticket); e != nil || r.Status != "participante" {
		t.Fatal(r, e)
	}
	r, e := b.JoinCommunity(ticket)
	if e != nil || r.Code != "invite_used" {
		t.Fatal("lost authenticated invite rejection", r, e)
	}
	record, e := b.db.Community(group)
	if e != nil || record.Denied {
		t.Fatal("unregistered identity was falsely marked denied", record, e)
	}
	if _, e = owner.CommunityAction(group, community.Remove, community.Body{Target: a.node.SelfID()}); e != nil {
		t.Fatal(e)
	}
	if e = a.SyncCommunity(group); e == nil {
		t.Fatal("removed identity still synced")
	}
	record, _ = a.db.Community(group)
	if !record.Denied {
		t.Fatal("known revocation not retained")
	}
	record.Addresses = owner.node.Addresses()
	if e = a.db.TrackCommunity(record); e != nil {
		t.Fatal(e)
	}
	record, _ = a.db.Community(group)
	if !record.Denied {
		t.Fatal("tracking a stale invitation cleared a known revocation")
	}
}
