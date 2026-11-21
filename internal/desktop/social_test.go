package desktop

import (
	"context"
	"projetop2p/internal/chat"
	"projetop2p/internal/community"
	"testing"
	"time"
)

func activityPull(t *testing.T, receiver, sender *Core, room chat.Room) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, e := receiver.node.CommunityRequest(ctx, sender.selfID(), sender.node.Addresses(), community.Request{Method: "activity.sync", ActivityRoom: &room})
	if e != nil || res.Error != "" {
		t.Fatal(res.Error, e)
	}
	for _, entry := range res.Activities {
		if e = receiver.db.SaveActivity(entry.Event, receiver.selfID(), sender.selfID()); e != nil {
			t.Fatal(e)
		}
	}
}
func TestAuthenticatedDirectChatActivitiesAndRecovery(t *testing.T) {
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	ai, _ := a.Invite()
	bi, _ := b.Invite()
	if e := a.AddContact(bi); e != nil {
		t.Fatal(e)
	}
	if e := b.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	if e := a.Send(b.selfID(), "original signed text"); e != nil {
		t.Fatal(e)
	}
	messages, _ := a.db.Messages(b.selfID())
	id := messages[len(messages)-1].ID
	ar, br := chat.Room{Peer: b.selfID()}, chat.Room{Peer: a.selfID()}
	if _, e := b.ChatAction(br, "edit", id, "forged", "", false); e == nil {
		t.Fatal("recipient edited another author")
	}
	if _, e := a.ChatAction(ar, "edit", id, "edited text", "", false); e != nil {
		t.Fatal(e)
	}
	activityPull(t, b, a, br)
	v, e := b.ActivityView(br, []string{a.selfID() + ":" + id}, true)
	if e != nil || v.Overlays[a.selfID()+":"+id].Text != "edited text" {
		t.Fatal(v, e)
	}
	if _, e = b.ChatAction(br, "reaction", id, "", "👍", true); e != nil {
		t.Fatal(e)
	}
	activityPull(t, a, b, ar)
	v, _ = a.ActivityView(ar, []string{a.selfID() + ":" + id}, true)
	if len(v.Reactions[a.selfID()+":"+id]) != 1 || v.Reactions[a.selfID()+":"+id][0].Count != 1 {
		t.Fatal(v)
	}
	reply, e := b.ChatAction(br, "reply", id, "a reply", "", false)
	if e != nil {
		t.Fatal(e)
	}
	activityPull(t, a, b, ar)
	v, _ = a.ActivityView(ar, []string{a.selfID() + ":" + id}, true)
	if v.Overlays[b.selfID()+":"+reply].ReplyTo != id {
		t.Fatal("reply lost", v)
	}
	if _, e = a.ChatAction(ar, "delete", id, "", "", false); e != nil {
		t.Fatal(e)
	}
	activityPull(t, b, a, br)
	v, _ = b.ActivityView(br, []string{a.selfID() + ":" + id}, true)
	if !v.Overlays[a.selfID()+":"+id].Deleted || v.Overlays[a.selfID()+":"+id].Text != "" {
		t.Fatal(v)
	}
	original, _ := b.db.Messages(a.selfID())
	if original[0].Text != "original signed text" {
		t.Fatal("immutable signature history overwritten")
	}
	if _, e = a.ChatAction(ar, "edit", id, "resurrect", "", false); e == nil {
		t.Fatal("deleted message resurrected")
	}
}
func TestCommunityActivityRequiresCurrentMembershipAndOriginalAuthor(t *testing.T) {
	owner := communityCore(t, t.TempDir(), "owner", freePort(t))
	defer owner.Close()
	b := communityCore(t, t.TempDir(), "member", freePort(t))
	defer b.Close()
	outsider := communityCore(t, t.TempDir(), "outsider", freePort(t))
	defer outsider.Close()
	group, e := owner.CreateCommunity("social fixture", "", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := owner.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.JoinCommunity(ticket); e != nil {
		t.Fatal(e)
	}
	receipt, e := owner.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "base message"})
	if e != nil {
		t.Fatal(e)
	}
	// Transfer the original signed journal record just as peer.sync does, before its dependent events.
	originals, _ := owner.db.PeerTexts(group, 0, 100)
	for _, entry := range originals {
		if e = b.db.SavePeerText(entry.Message, b.selfID()); e != nil {
			t.Fatal(e)
		}
	}
	_ = b.SyncCommunity(group)
	room := chat.Room{Group: group, Channel: "geral"}
	if _, e = b.ChatAction(room, "delete", receipt.ID, "", "", false); e == nil {
		t.Fatal("member deleted owner message")
	}
	if _, e = owner.ChatAction(room, "edit", receipt.ID, "updated", "", false); e != nil {
		t.Fatal(e)
	}
	activityPull(t, b, owner, room)
	v, _ := b.ActivityView(room, []string{receipt.ID}, true)
	if v.Overlays[receipt.ID].Text != "updated" {
		t.Fatal(v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, e := outsider.node.CommunityRequest(ctx, owner.selfID(), owner.node.Addresses(), community.Request{Method: "activity.sync", ActivityRoom: &room})
	if e == nil && res.Error == "" {
		t.Fatal("outsider read activity log")
	}
}
