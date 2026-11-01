package community

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"testing"
	"time"
)

func key(t *testing.T) crypto.PrivKey {
	t.Helper()
	k, _, e := crypto.GenerateEd25519Key(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	return k
}
func root(t *testing.T, k crypto.PrivKey) (Commit, State) {
	t.Helper()
	o, e := SignOperation(k, "", "", Create, Body{Name: "Comunidade", Nickname: "Dona"})
	if e != nil {
		t.Fatal(e)
	}
	c, s, e := MakeCommit(k, State{}, o)
	if e != nil {
		t.Fatal(e)
	}
	return c, s
}
func ticket(t *testing.T, k crypto.PrivKey, s State) string {
	t.Helper()
	raw, e := NewInvite(k, s, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + s.Owner})
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func appendOp(t *testing.T, k, actor crypto.PrivKey, s State, kind string, b Body) State {
	t.Helper()
	o, e := SignOperation(actor, s.ID, s.Head, kind, b)
	if e != nil {
		t.Fatal(e)
	}
	_, next, e := MakeCommit(k, s, o)
	if e != nil {
		t.Fatal(e)
	}
	return next
}
func TestRolesInvitesAndRevocation(t *testing.T) {
	owner := key(t)
	member := key(t)
	other := key(t)
	_, s := root(t, owner)
	id, _ := peer.IDFromPrivateKey(member)
	invite := ticket(t, owner, s)
	s = appendOp(t, owner, member, s, Join, Body{Name: "Pessoa", Invite: invite})
	if s.Role(id.String()) != "member" {
		t.Fatal("membership not derived from signed join")
	}
	op, _ := SignOperation(member, s.ID, s.Head, SetRole, Body{Target: id.String(), Role: "moderator"})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("self-declared moderator accepted")
	}
	s = appendOp(t, owner, owner, s, SetRole, Body{Target: id.String(), Role: "moderator"})
	s = appendOp(t, owner, member, s, ChannelCreate, Body{ChannelID: RandomID(), Name: "bairro", ChannelKind: "text"})
	op, _ = SignOperation(member, s.ID, s.Head, Remove, Body{Target: s.Owner})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("moderator removed owner")
	}
	op, _ = SignOperation(member, s.ID, s.Head, SetRole, Body{Target: id.String(), Role: "owner"})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("moderator changed ownership")
	}
	stale := ticket(t, owner, s)
	s = appendOp(t, owner, owner, s, Remove, Body{Target: id.String()})
	op, _ = SignOperation(member, s.ID, s.Head, Text, Body{ChannelID: "geral", Text: "revoked"})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("revoked member posted")
	}
	op, _ = SignOperation(other, s.ID, s.Head, Join, Body{Name: "Other", Invite: stale})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("old invitation survived membership revocation")
	}
	if _, e := NewInvite(member, s, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + s.Owner}); e == nil {
		t.Fatal("non-owner minted an owner invite")
	}
}
func TestConflictOrderingSignaturesAndOneUseInvite(t *testing.T) {
	owner := key(t)
	member := key(t)
	attacker := key(t)
	c, s := root(t, owner)
	forged := c
	forged.Operation.Body.Name = "forged"
	if _, e := Apply(State{}, forged); e == nil {
		t.Fatal("tampered genesis accepted")
	}
	raw := ticket(t, owner, s)
	parsed, e := ParseInvite(raw, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	parsed.Owner = "invalid"
	b, _ := json.Marshal(parsed)
	if _, e = ParseInvite(InvitePrefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("tampered invite accepted")
	}
	s = appendOp(t, owner, member, s, Join, Body{Name: "Member", Invite: raw})
	op, _ := SignOperation(attacker, s.ID, s.Head, Join, Body{Name: "Other", Invite: raw})
	if _, _, e = MakeCommit(owner, s, op); e == nil {
		t.Fatal("one-use invite replayed")
	}
	a, _ := SignOperation(owner, s.ID, s.Head, ChannelCreate, Body{ChannelID: RandomID(), Name: "a", ChannelKind: "text"})
	z, _ := SignOperation(owner, s.ID, s.Head, ChannelCreate, Body{ChannelID: RandomID(), Name: "z", ChannelKind: "text"})
	_, s2, e := MakeCommit(owner, s, a)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = MakeCommit(owner, s2, z); e == nil {
		t.Fatal("stale control update accepted")
	}
	text, _ := SignOperation(member, s.ID, s.Head, Text, Body{ChannelID: "geral", Text: "offline but still authorized"})
	commit, next, e := MakeCommit(owner, s2, text)
	if e != nil || next.Sequence != s2.Sequence+1 {
		t.Fatal(e)
	}
	if _, e = Apply(next, commit); e == nil {
		t.Fatal("duplicate commit applied twice")
	}
}
func TestInputBoundsAndVoiceDefinitions(t *testing.T) {
	owner := key(t)
	_, s := root(t, owner)
	if s.Channels["voz"].Kind != "voice" {
		t.Fatal("voice architecture missing")
	}
	op, _ := SignOperation(owner, s.ID, s.Head, Text, Body{ChannelID: "voz", Text: "not audio"})
	if _, _, e := MakeCommit(owner, s, op); e == nil {
		t.Fatal("text posted into voice-only definition")
	}
	if _, e := SignOperation(owner, s.ID, s.Head, Text, Body{Text: string(make([]byte, MaxText+1))}); e == nil {
		t.Fatal("oversized/invalid text accepted")
	}
	op, e := SignOperation(owner, s.ID, s.Head, ChannelCreate, Body{ChannelID: "../../private", Name: "x", ChannelKind: "text"})
	if e == nil {
		_, _, e = MakeCommit(owner, s, op)
	}
	if e == nil {
		t.Fatal("path-like channel ID accepted")
	}
}
