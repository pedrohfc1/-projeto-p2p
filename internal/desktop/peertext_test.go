package desktop

import (
	"context"
	"projetop2p/internal/community"
	"testing"
	"time"
)

func TestPeerTextContinuesWithoutOwnerAndRejectsForgery(t *testing.T) {
	owner := communityCore(t, t.TempDir(), "owner", freePort(t))
	defer owner.Close()
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	group, e := owner.CreateCommunity("Peer journal", "", "")
	if e != nil {
		t.Fatal(e)
	}
	raw, e := owner.CommunityInvitePolicy(group, community.InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 24})
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range []*Core{a, b} {
		r, e := c.JoinCommunity(raw)
		if e != nil || r.Status != "participante" {
			t.Fatal(r, e)
		}
	}
	if e = a.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	for _, c := range []*Core{a, b, a} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.groups.DiscoverPeers(ctx, group)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}
	// Wait for authenticated introductions before deliberately taking the authority offline.
	deadline := time.Now().Add(15 * time.Second)
	for {
		r, e := a.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "warmup"})
		if e != nil {
			t.Fatal(e)
		}
		_ = r
		v, _ := b.CommunityState(group, "geral", 0)
		if len(v.Messages) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("peer introduction failed")
		}
		time.Sleep(time.Second)
	}
	owner.mu.Lock()
	owner.stopNode()
	owner.mu.Unlock()
	first, e := a.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "owner offline"})
	if e != nil || first.Status != "replicada" {
		t.Fatal(first, e)
	}
	second, e := b.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "reply without authority"})
	if e != nil || second.Status != "replicada" {
		t.Fatal(second, e)
	}
	state, _ := a.db.Community(group)
	entries, _ := a.db.PeerTextHistory(group, "geral", 0)
	var m community.PeerText
	for _, entry := range entries {
		if entry.Message.Operation.ID == first.ID {
			m = entry.Message
		}
	}
	if m.Operation.ID == "" {
		t.Fatal("local journal missing")
	}
	// Acknowledgements cannot authenticate content; signatures and transport identity do.
	forged := m
	forged.Operation.Body.Text = "forged"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, e := a.node.CommunityRequest(ctx, b.node.SelfID(), b.node.Addresses(), community.Request{Method: "peer.text", CommunityID: group, PeerText: &forged})
	if e != nil || res.Error == "" {
		t.Fatal("forgery accepted", res, e)
	}
	res, e = a.node.CommunityRequest(ctx, b.node.SelfID(), b.node.Addresses(), community.Request{Method: "peer.text", CommunityID: group, PeerText: &m})
	if e != nil || res.Ack != m.Operation.ID {
		t.Fatal("idempotent retry", res, e)
	}
	v, _ := b.CommunityState(group, "geral", 0)
	n := 0
	for _, message := range v.Messages {
		if message.ID == m.Operation.ID {
			n++
		}
	}
	if n != 1 {
		t.Fatal("duplicate", n)
	}
	m.Epoch = state.State.InviteEpoch + 1
	if e = b.db.SavePeerText(m, b.node.SelfID()); e == nil {
		t.Fatal("tampered authorization epoch accepted")
	}
}
