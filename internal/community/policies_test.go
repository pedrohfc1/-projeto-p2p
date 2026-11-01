package community

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInvitePoliciesAndLegacy(t *testing.T) {
	owner := key(t)
	_, s := root(t, owner)
	addr := []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + s.Owner}
	old := ticket(t, owner, s)
	s = appendOp(t, owner, key(t), s, Join, Body{Name: "one", Invite: old})
	o, _ := SignOperation(key(t), s.ID, s.Head, Join, Body{Name: "two", Invite: old})
	if _, _, e := MakeCommit(owner, s, o); e == nil || !strings.Contains(e.Error(), "invite_used") {
		t.Fatal("legacy ticket reuse", e)
	}
	raw, e := NewPolicyInvite(owner, s, "interface", addr, InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 720})
	if e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 5; n++ {
		s = appendOp(t, owner, key(t), s, Join, Body{Name: "multi", Invite: raw})
	}
	o, _ = SignOperation(key(t), s.ID, s.Head, Join, Body{Name: "overflow", Invite: raw})
	if _, _, e = MakeCommit(owner, s, o); e == nil || !strings.Contains(e.Error(), "invite_exhausted") {
		t.Fatal("quota exceeded", e)
	}
	forever, e := NewPolicyInvite(owner, s, "interface", addr, InviteOptions{Kind: "permanent"})
	if e != nil {
		t.Fatal(e)
	}
	i, e := ParseInvite(forever, time.Now().Add(365*24*time.Hour))
	if e != nil || i.Expires != 0 {
		t.Fatal("permanent", e)
	}
	changed := i
	changed.MaxUses = 100
	b, _ := json.Marshal(changed)
	if _, e = ParseInvite(InvitePrefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("policy tampering accepted")
	}
	s = appendOp(t, owner, owner, s, InviteRevoke, Body{Invite: i.Nonce})
	o, _ = SignOperation(key(t), s.ID, s.Head, Join, Body{Name: "revoked", Invite: forever})
	if _, _, e = MakeCommit(owner, s, o); e == nil || !strings.Contains(e.Error(), "invite_revoked") {
		t.Fatal("revoked ticket", e)
	}
	exp, e := NewPolicyInvite(owner, s, "interface", addr, InviteOptions{Kind: "single", MaxUses: 1, LifetimeHours: 1})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = ParseInvite(exp, time.Now().Add(2*time.Hour)); e == nil || !strings.Contains(e.Error(), "invite_expired") {
		t.Fatal("expiration", e)
	}
}
func TestHundredRegisteredMembers(t *testing.T) {
	owner := key(t)
	op, _ := SignOperation(owner, "", "", Create, Body{Name: "Capacity", Limit: 100})
	_, s, e := MakeCommit(owner, State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := NewPolicyInvite(owner, s, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + s.Owner}, InviteOptions{Kind: "permanent"})
	if e != nil {
		t.Fatal(e)
	}
	for n := 1; n < 100; n++ {
		s = appendOp(t, owner, key(t), s, Join, Body{Name: "registered", Invite: raw})
		if n == 2 || n == 4 || n == 9 || n == 19 {
			t.Log("authenticated members", len(s.Members))
		}
	}
	if len(s.Members) != 100 {
		t.Fatal(len(s.Members))
	}
	op, _ = SignOperation(key(t), s.ID, s.Head, Join, Body{Name: "full", Invite: raw})
	if _, _, e = MakeCommit(owner, s, op); e == nil || !strings.Contains(e.Error(), "community_full") {
		t.Fatal("community capacity", e)
	}
}

func TestChannelCapacityAndSignedVoiceSettings(t *testing.T) {
	owner := key(t)
	op, _ := SignOperation(owner, "", "", Create, Body{Name: "Rooms", Limit: 100})
	_, s, e := MakeCommit(owner, State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"text", "voice"} {
		count := 50
		if kind == "voice" {
			count = 20
		}
		for n := 1; n < count; n++ {
			s = appendOp(t, owner, owner, s, ChannelCreate, Body{ChannelID: RandomID(), Name: "room", ChannelKind: kind})
		}
		o, _ := SignOperation(owner, s.ID, s.Head, ChannelCreate, Body{ChannelID: RandomID(), Name: "overflow", ChannelKind: kind})
		if _, _, e := MakeCommit(owner, s, o); e == nil {
			t.Fatal("channel quota bypassed", kind)
		}
	}
	s = appendOp(t, owner, owner, s, ChannelUpdate, Body{ChannelID: "voz", Name: "Restricted", Limit: 1, Role: "moderator"})
	if ch := s.Channels["voz"]; ch.Name != "Restricted" || ch.Limit != 1 || ch.Role != "moderator" {
		t.Fatal(ch)
	}
}
