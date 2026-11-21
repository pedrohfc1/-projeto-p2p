package localdb

import (
	"testing"
	"time"
)

func TestInboxPrivacyDedupMuteAndUnread(t *testing.T) {
	db, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	_ = db.Contact(Contact{ID: "friend", Name: "friend"})
	_ = db.PollInbox("self", nil)
	prefs, _ := db.NotificationPreferences()
	if prefs.Enabled || prefs.Preview {
		t.Fatal("private content opt-in bypassed")
	}
	prefs.Enabled = true
	_ = db.SaveNotificationPreferences(prefs)
	m := Message{ID: "one", Peer: "friend", From: "friend", Text: "private body", Time: time.Now().Unix()}
	_ = db.AddMessage(m)
	_ = db.PollInbox("self", nil)
	_ = db.PollInbox("self", nil)
	notices, e := db.DrainNotices()
	if e != nil || len(notices) != 1 || notices[0].Body != "Nova mensagem privada" {
		t.Fatal(notices, e)
	}
	again, _ := db.DrainNotices()
	if len(again) != 0 {
		t.Fatal("duplicate native alert")
	}
	unread, _ := db.Unread()
	if unread["d:friend"] != 1 {
		t.Fatal(unread)
	}
	_ = db.MarkRead("d:friend")
	unread, _ = db.Unread()
	if unread["d:friend"] != 0 {
		t.Fatal(unread)
	}
	prefs.Muted["d:friend"] = true
	_ = db.SaveNotificationPreferences(prefs)
	m.ID = "two"
	_ = db.AddMessage(m)
	_ = db.PollInbox("self", nil)
	notices, _ = db.DrainNotices()
	if len(notices) != 0 {
		t.Fatal("muted context notified")
	}
	m.ID = "three"
	_ = db.AddMessage(m)
	_ = db.PollInbox("self", map[string]bool{"d:friend": true})
	unread, _ = db.Unread()
	if unread["d:friend"] != 1 {
		t.Fatal("visible conversation counted as unread", unread)
	}
}
