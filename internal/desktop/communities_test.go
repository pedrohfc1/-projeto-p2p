package desktop

import (
	"context"
	"fmt"
	"projetop2p/internal/adapters"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"testing"
	"time"
)

func communityCore(t *testing.T, dir, name string, port int) *Core {
	t.Helper()
	c, e := OpenWithProvider(context.Background(), dir, func() ([]adapters.Info, error) {
		return []adapters.Info{{ID: "lo", Index: 1, Name: "Loopback", Up: true, IPv4: []string{"127.0.0.1"}}}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if e = c.SaveSettings(localdb.Settings{Name: name, Mode: "interface", InterfaceID: "lo", Port: port}); e != nil {
		c.Close()
		t.Fatal(e)
	}
	return c
}
func TestCommunitiesAuthenticatedRolesSyncAndRevocation(t *testing.T) {
	owner := communityCore(t, t.TempDir(), "Dona", freePort(t))
	defer owner.Close()
	mod := communityCore(t, t.TempDir(), "Moderador", freePort(t))
	defer mod.Close()
	member := communityCore(t, t.TempDir(), "Membro", freePort(t))
	defer member.Close()
	outsider := communityCore(t, t.TempDir(), "Intruso", freePort(t))
	defer outsider.Close()
	group, e := owner.CreateCommunity("Bairro conectado", "Sem servidor global", "🇧🇷")
	if e != nil {
		t.Fatal(e)
	}
	inv, e := owner.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	joined, e := mod.JoinCommunity(inv)
	if e != nil || joined.Status != "participante" {
		t.Fatal(joined, e)
	}
	// A consumed ticket cannot join another identity.
	reused, e := outsider.JoinCommunity(inv)
	if e != nil {
		t.Fatal(e)
	}
	if reused.Status == "participante" {
		t.Fatal("one-use invite accepted twice")
	}
	ms, _ := mod.State("")
	mid := ms.Network.ID
	if _, e = mod.CommunityAction(group, community.SetRole, community.Body{Target: mid, Role: "moderator"}); e == nil {
		t.Fatal("self-promotion accepted")
	}
	if _, e = owner.CommunityAction(group, community.SetRole, community.Body{Target: mid, Role: "moderator"}); e != nil {
		t.Fatal(e)
	}
	if e = mod.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	inv, e = mod.CommunityInvite(group)
	if e != nil {
		t.Fatal("moderator could not obtain owner-signed invite", e)
	}
	joined, e = member.JoinCommunity(inv)
	if e != nil || joined.Status != "participante" {
		t.Fatal(joined, e)
	}
	if _, e = member.CommunityAction(group, community.ChannelCreate, community.Body{Name: "unauthorized", ChannelKind: "text"}); e == nil {
		t.Fatal("member created channel")
	}
	if e = mod.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	if _, e = mod.CommunityAction(group, community.ChannelCreate, community.Body{Name: "conversa", ChannelKind: "text"}); e != nil {
		t.Fatal(e)
	}
	if e = member.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	view, e := member.CommunityState(group, "geral", 0)
	if e != nil {
		t.Fatal(e)
	}
	channel := ""
	for _, ch := range view.Community.Channels {
		if ch.Name == "conversa" {
			channel = ch.ID
		}
	}
	if channel == "" {
		t.Fatal("channel not replicated")
	}
	receipt, e := member.CommunityAction(group, community.Text, community.Body{ChannelID: channel, Text: "<script>not executable</script> 🇧🇷"})
	if e != nil || receipt.Status != "confirmada" {
		t.Fatal(receipt, e)
	}
	if e = mod.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	view, e = mod.CommunityState(group, channel, 0)
	if e != nil || len(view.Messages) != 1 || view.Messages[0].ID != receipt.ID {
		t.Fatal(view, e)
	}
	// Authenticating a transport does not authorize outsiders or forwarded owner operations.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, _ := owner.db.Community(group)
	response, e := outsider.node.CommunityRequest(ctx, r.Owner, owner.node.Addresses(), community.Request{Method: "sync", CommunityID: group})
	if e != nil || response.Code != "forbidden" || len(response.Commits) != 0 {
		t.Fatal(response, e)
	}
	genesis, _ := owner.db.CommunityEvent(group, 1)
	response, e = outsider.node.CommunityRequest(ctx, r.Owner, owner.node.Addresses(), community.Request{Method: "submit", CommunityID: group, Operation: &genesis.Operation})
	if e != nil || response.Code != "forbidden" {
		t.Fatal("forwarded identity accepted", response, e)
	}
	memberState, _ := member.State("")
	if _, e = owner.CommunityAction(group, community.Remove, community.Body{Target: memberState.Network.ID}); e != nil {
		t.Fatal(e)
	}
	if e = member.SyncCommunity(group); e == nil {
		t.Fatal("revoked member synchronized")
	}
	after, _ := member.CommunityState(group, channel, 0)
	if !after.Community.Denied {
		t.Fatal("revocation not surfaced")
	}
	if _, e = member.CommunityAction(group, community.Text, community.Body{ChannelID: channel, Text: "revoked"}); e == nil {
		t.Fatal("revoked member posted")
	}
	if _, e = mod.CommunityAction(group, community.Remove, community.Body{Target: r.Owner}); e == nil {
		t.Fatal("moderator removed owner")
	}
	// Rejoining must work both after a known revocation and before the replica
	// has learned about a new revocation. It must not resurrect delegated roles.
	fresh, _ := owner.CommunityInvite(group)
	rejoined, e := member.JoinCommunity(fresh)
	if e != nil || rejoined.Status != "participante" {
		t.Fatal("rejoin failed", rejoined, e)
	}
	if _, e = owner.CommunityAction(group, community.Remove, community.Body{Target: memberState.Network.ID}); e != nil {
		t.Fatal(e)
	}
	fresh, _ = owner.CommunityInvite(group)
	rejoined, e = member.JoinCommunity(fresh)
	if e != nil || rejoined.Status != "participante" {
		t.Fatal("stale cached membership prevented fresh rejoin", rejoined, e)
	}
	restored, _ := member.CommunityState(group, channel, 0)
	if restored.Community.Role != "member" || restored.Community.Denied {
		t.Fatal("rejoin changed role or kept stale denial")
	}
}
func TestCommunityOutboxSurvivesOwnerOutageAndRestart(t *testing.T) {
	ownerDir := t.TempDir()
	ownerPort := freePort(t)
	owner := communityCore(t, ownerDir, "Dona", ownerPort)
	defer func() {
		if owner != nil {
			owner.Close()
		}
	}()
	memberDir := t.TempDir()
	memberPort := freePort(t)
	member := communityCore(t, memberDir, "Membro", memberPort)
	defer func() {
		if member != nil {
			member.Close()
		}
	}()
	group, e := owner.CreateCommunity("Persistência", "", "🌱")
	if e != nil {
		t.Fatal(e)
	}
	inv, _ := owner.CommunityInvite(group)
	join, e := member.JoinCommunity(inv)
	if e != nil || join.Status != "participante" {
		t.Fatal(join, e)
	}
	original, _ := owner.State("")
	memberIdentity, _ := member.State("")
	owner.Close()
	owner = nil
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	queued, e := member.groups.Action(ctx, group, community.Text, community.Body{ChannelID: "geral", Text: "offline durable"})
	cancel()
	if e != nil || queued.Status != "pendente" {
		t.Fatal(queued, e)
	}
	member.Close()
	member = nil
	member = communityCore(t, memberDir, "Membro", memberPort)
	view, e := member.CommunityState(group, "geral", 0)
	if e != nil || len(view.Messages) != 1 || view.Messages[0].ID != queued.ID || view.Messages[0].Sequence != 0 {
		t.Fatal("outbox lost", view, e)
	}
	owner = communityCore(t, ownerDir, "Dona", ownerPort)
	deadline := time.Now().Add(25 * time.Second)
	for {
		e = member.SyncCommunity(group)
		view, _ = member.CommunityState(group, "geral", 0)
		if e == nil && len(view.Messages) == 1 && view.Messages[0].Status == "confirmada" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("recovery failed", e, view)
		}
		time.Sleep(300 * time.Millisecond)
	}
	if view.Messages[0].ID != queued.ID {
		t.Fatal("retry changed stable ID")
	}
	if e = member.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	rows, _ := owner.CommunityState(group, "geral", 0)
	if len(rows.Messages) != 1 {
		t.Fatal("replay created duplicate", rows.Messages)
	}
	a, _ := owner.State("")
	b, _ := member.State("")
	if a.Network.ID != original.Network.ID || b.Network.ID != memberIdentity.Network.ID {
		t.Fatal("identity changed")
	}
	// Paginated messages are bounded, stable and survive reconstruction.
	for i := 0; i < 103; i++ {
		if _, e = owner.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: fmt.Sprint(i)}); e != nil {
			t.Fatal(e)
		}
	}
	last, _ := owner.CommunityState(group, "geral", 0)
	if len(last.Messages) != 100 || !last.HasMore {
		t.Fatal("history not paginated")
	}
	earlier, e := owner.CommunityState(group, "geral", last.Messages[0].Sequence)
	if e != nil || len(earlier.Messages) != 4 {
		t.Fatal("pagination cursor wrong", len(earlier.Messages), e)
	}
}
