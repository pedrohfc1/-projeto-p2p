package desktop

import (
	"context"
	"path/filepath"
	"projetop2p/internal/community"
	"testing"
	"time"
)

func TestLegacyCommunityPeerTextOwnerOfflineAndRecovery(t *testing.T) {
	dir := t.TempDir()
	owner := communityCore(t, filepath.Join(dir, "owner"), "owner", freePort(t))
	defer func() {
		if owner != nil {
			owner.Close()
		}
	}()
	b := communityCore(t, filepath.Join(dir, "b"), "b", freePort(t))
	defer b.Close()
	c := communityCore(t, filepath.Join(dir, "c"), "c", freePort(t))
	defer c.Close()
	group, e := createLegacyCommunity(t, owner, "legacy offline", "", "")
	if e != nil {
		t.Fatal(e)
	}
	for _, member := range []*Core{b, c} {
		ticket, e := owner.CommunityInvite(group)
		if e != nil {
			t.Fatal(e)
		}
		r, e := member.JoinCommunity(ticket)
		if e != nil || r.Status != "participante" {
			t.Fatal(r, e)
		}
	}
	for _, member := range []*Core{b, c} {
		if e = member.SyncCommunity(group); e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e = member.groups.DiscoverPeers(ctx, group)
		cancel()
		if e != nil {
			t.Fatal(e)
		}
	}

	ownerSettings, _ := owner.db.Settings()
	ownerID := owner.node.SelfID()
	owner.Close()
	owner = nil
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	receipt, e := b.groups.Action(ctx, group, community.Text, community.Body{ChannelID: "geral", Text: "legacy distributed offline"})
	cancel()
	if e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(22 * time.Second)
	for {
		v, _ := c.CommunityState(group, "geral", 0)
		if len(v.Messages) == 1 && v.Messages[0].ID == receipt.ID {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("legacy offline text not delivered", v)
		}
		time.Sleep(200 * time.Millisecond)
	}
	owner = communityCore(t, filepath.Join(dir, "owner"), "owner", ownerSettings.Port)
	if owner.node.SelfID() != ownerID {
		t.Fatal("identity changed")
	}
	deadline = time.Now().Add(20 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		e = b.groups.Sync(ctx, group)
		cancel()
		v, _ := b.CommunityState(group, "geral", 0)
		if e == nil && len(v.Messages) == 1 && v.Messages[0].ID == receipt.ID && v.Messages[0].Status == "confirmada" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("legacy replay duplicated or not confirmed", e, v)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
