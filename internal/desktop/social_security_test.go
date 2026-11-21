package desktop

import (
	"projetop2p/internal/chat"
	"projetop2p/internal/localdb"
	"testing"
	"time"
)

func TestDirectReferencesCannotCollideAcrossAuthors(t *testing.T) {
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	ai, _ := a.Invite()
	bi, _ := b.Invite()
	_ = a.AddContact(bi)
	_ = b.AddContact(ai)
	id := "1234567890abcdef1234567890abcdef"
	for _, c := range []*Core{a, b} {
		other := a.selfID()
		if c == a {
			other = b.selfID()
		}
		_ = c.db.AddMessage(localdb.Message{ID: id, Peer: other, From: a.selfID(), Text: "author A", Time: time.Now().Unix()})
		_ = c.db.AddMessage(localdb.Message{ID: id, Peer: other, From: b.selfID(), Text: "author B", Time: time.Now().Unix()})
	}
	room := chat.Room{Peer: a.selfID()}
	if _, e := b.ChatAction(room, "edit", b.selfID()+":"+id, "B edited", "", false); e != nil {
		t.Fatal(e)
	}
	activityPull(t, a, b, chat.Room{Peer: b.selfID()})
	view, e := a.ActivityView(chat.Room{Peer: b.selfID()}, []string{a.selfID() + ":" + id, b.selfID() + ":" + id}, true)
	if e != nil || view.Overlays[b.selfID()+":"+id].Text != "B edited" {
		t.Fatal(view, e)
	}
	if _, ok := view.Overlays[a.selfID()+":"+id]; ok {
		t.Fatal("other author's message was overwritten")
	}
	if _, e = b.ChatAction(room, "delete", a.selfID()+":"+id, "", "", false); e == nil {
		t.Fatal("foreign deletion accepted")
	}
}
func TestTypingPrivacyAndBackgroundReadState(t *testing.T) {
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	ai, _ := a.Invite()
	bi, _ := b.Invite()
	_ = a.AddContact(bi)
	_ = b.AddContact(ai)
	room := chat.Room{Peer: b.selfID()}
	other := chat.Room{Peer: a.selfID()}
	p, _ := a.NotificationPreferences()
	p.Typing = false
	_ = a.SaveNotificationPreferences(p)
	_, _ = a.ChatAction(room, "typing", "", "", "", true)
	view, _ := b.ActivityView(other, nil, true)
	if len(view.Typing) > 0 {
		t.Fatal("typing preference ignored")
	}
	p.Typing = true
	_ = a.SaveNotificationPreferences(p)
	_, e := a.ChatAction(room, "typing", "", "", "", true)
	if e != nil {
		t.Fatal(e)
	}
	view, _ = b.ActivityView(other, nil, true)
	if len(view.Typing) != 1 {
		t.Fatal("signed typing not delivered", view)
	}
	time.Sleep(6100 * time.Millisecond)
	view, _ = b.ActivityView(other, nil, true)
	if len(view.Typing) != 0 {
		t.Fatal("typing did not expire")
	}
	b.SetChatForeground(false)
	if e = a.Send(b.selfID(), "background fixture"); e != nil {
		t.Fatal(e)
	}
	_, _ = b.ActivityView(other, nil, true)
	b.pollInbox()
	unread, _ := b.db.Unread()
	if unread["d:"+a.selfID()] != 1 {
		t.Fatal("background query marked unread message read", unread)
	}
	b.SetChatForeground(true)
	_, _ = b.ActivityView(other, nil, true)
	unread, _ = b.db.Unread()
	if unread["d:"+a.selfID()] != 0 {
		t.Fatal("foreground did not mark read", unread)
	}
}
