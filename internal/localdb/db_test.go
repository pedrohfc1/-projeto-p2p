package localdb

import "testing"

func TestTransactionsAndDeduplication(t *testing.T) {
	d := t.TempDir()
	db, e := Open(d)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Contact(Contact{ID: "peer", Name: "Alice"}); e != nil {
		t.Fatal(e)
	}
	m := Message{ID: "nonce", From: "sender", Peer: "peer", State: "enviando", Text: "olá"}
	if e = db.AddMessage(m); e != nil {
		t.Fatal(e)
	}
	m.State = "confirmada"
	if e = db.AddMessage(m); e != nil {
		t.Fatal(e)
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	db, e = Open(d)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	history, e := db.Messages("peer")
	if e != nil || len(history) != 1 || history[0].State != "confirmada" {
		t.Fatal(history, e)
	}
	cs, e := db.Contacts()
	if e != nil || len(cs) != 1 || cs[0].Name != "Alice" {
		t.Fatal(cs, e)
	}
}

func TestMessageContentCannotBeRewrittenAfterRestart(t *testing.T) {
	dir := t.TempDir()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	original := Message{ID: "same-id", From: "peer", Peer: "peer", Text: "original", Time: 123, State: "recebida"}
	if e = d.AddMessage(original); e != nil {
		t.Fatal(e)
	}
	d.Close()
	d, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	altered := original
	altered.Text = "replacement"
	if e = d.AddMessage(altered); e == nil {
		t.Fatal("immutable text rewritten")
	}
	if e = d.AddMessage(original); e != nil {
		t.Fatal("idempotent duplicate rejected", e)
	}
	rows, _ := d.Messages("peer")
	if len(rows) != 1 || rows[0].Text != "original" {
		t.Fatal(rows)
	}
}
