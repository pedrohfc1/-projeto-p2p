package desktop

import (
	"context"
	"projetop2p/internal/localdb"
	"testing"
	"time"
)

func TestTwoProfilesAndRestart(t *testing.T) {
	a, e := Open(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	dir := t.TempDir()
	b, e := Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if b != nil {
			b.Close()
		}
	}()
	ai, e := a.Invite()
	if e != nil {
		t.Fatal(e)
	}
	bi, e := b.Invite()
	if e != nil {
		t.Fatal(e)
	}
	if e = a.AddContact(bi); e != nil {
		t.Fatal(e)
	}
	if e = b.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	as, _ := a.State("")
	bs, _ := b.State("")
	if e = a.SaveSettings(localdb.Settings{Name: "Alice", Assisted: true}); e != nil {
		t.Fatal(e)
	}
	// Allow known contact reconnection; direct loopback connections work even in assisted mode.
	deadline := time.Now().Add(30 * time.Second)
	for {
		e = a.Send(bs.Network.ID, "mensagem durável")
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(e)
		}
		time.Sleep(200 * time.Millisecond)
	}
	state, e := b.State(as.Network.ID)
	if e != nil || len(state.Messages) == 0 {
		t.Fatal("no persisted message", e)
	}
	if e = b.Close(); e != nil {
		t.Fatal(e)
	}
	b = nil
	b, e = Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	state, e = b.State(as.Network.ID)
	if e != nil {
		t.Fatal(e)
	}
	if state.Network.ID != bs.Network.ID || len(state.Contacts) != 1 || len(state.Messages) == 0 {
		t.Fatal("restart lost state")
	}
}
