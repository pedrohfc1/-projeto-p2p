package desktop

import (
	"context"
	"net"
	"projetop2p/internal/adapters"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"sync"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}
func TestScopedContactsReconnectAndIPChange(t *testing.T) {
	var mu sync.Mutex
	up := true
	ip := "127.0.0.1"
	provider := func() ([]adapters.Info, error) {
		mu.Lock()
		defer mu.Unlock()
		return []adapters.Info{{ID: "test-interface", Name: "Test loopback", Up: up, IPv4: []string{ip}}}, nil
	}
	a, e := OpenWithProvider(context.Background(), t.TempDir(), provider)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	dir := t.TempDir()
	b, e := OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if b != nil {
			b.Close()
		}
	}()
	sa := localdb.Settings{Name: "A", Mode: "interface", InterfaceID: "test-interface", Port: freePort(t)}
	sb := sa
	sb.Name = "B"
	sb.Port = freePort(t)
	if e = a.SaveSettings(sa); e != nil {
		t.Fatal(e)
	}
	if e = b.SaveSettings(sb); e != nil {
		t.Fatal(e)
	}
	as, _ := a.State("")
	bs, _ := b.State("")
	if !as.Network.LocalOnly || a.node.DHT != nil || len(as.Network.Addresses) != 1 {
		t.Fatal("scoped transport leaked discovery")
	}
	ai, e := a.Invite()
	if e != nil {
		t.Fatal(e)
	}
	bi, e := b.Invite()
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := invite.Parse(bi, time.Now())
	if e != nil || parsed.Scope != "interface" || len(parsed.Relays) != 0 {
		t.Fatal(parsed, e)
	}
	result, e := a.ConnectInvite(bi)
	if e != nil || !result.Connected {
		t.Fatal(result, e)
	}
	if r, e := a.ConnectInvite(bs.Network.Addresses[0]); e != nil || !r.Connected {
		t.Fatal("raw multiaddr failed", r, e)
	}
	if _, e = a.ConnectInvite("/ip4/invalid/tcp/4001/p2p/bad"); e == nil {
		t.Fatal("invalid raw invite accepted")
	}
	if e = b.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	if e = a.Send(bs.Network.ID, "interface message"); e != nil {
		t.Fatal(e)
	}
	oldID := bs.Network.ID
	if e = b.Close(); e != nil {
		t.Fatal(e)
	}
	b = nil
	b, e = OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	bs, e = b.State(as.Network.ID)
	if e != nil || bs.Network.ID != oldID || len(bs.Messages) != 1 || len(bs.Contacts) != 1 {
		t.Fatal("restart lost profile", e)
	}
	mu.Lock()
	up = false
	mu.Unlock()
	state, e := a.State("")
	if e != nil || state.NetworkError == "" || a.node != nil {
		t.Fatal("disconnected adapter retained host", e)
	}
	mu.Lock()
	up = true
	mu.Unlock()
	state, e = a.State("")
	if e != nil || a.node == nil || state.Network.ID != as.Network.ID {
		t.Fatal("reconnect lost identity", e)
	}
	// A changed IP that is not assigned locally must stop the old listener;
	// it must never silently fall back to the public/native mode.
	mu.Lock()
	ip = "192.0.2.10"
	mu.Unlock()
	state, e = a.State("")
	if e != nil || state.Connectivity.IP != ip || state.NetworkError == "" || a.node != nil {
		t.Fatal("IP change did not rebind/fail closed", e)
	}
	mu.Lock()
	ip = "127.0.0.1"
	mu.Unlock()
	state, e = a.State("")
	if e != nil || a.node == nil || !state.Network.LocalOnly {
		t.Fatal("recovery failed", e)
	}
	if _, e = b.State(""); e != nil {
		t.Fatal(e)
	}
	if r, e := a.TestConnectivity(oldID); e != nil || !r.Connected {
		t.Fatal("peer reconnection failed", r, e)
	}
	if e = a.Send(oldID, "after adapter recovery"); e != nil {
		t.Fatal(e)
	}
}
func TestHamachiAbsentRemainsConfigurable(t *testing.T) {
	c, e := OpenWithProvider(context.Background(), t.TempDir(), func() ([]adapters.Info, error) { return nil, nil })
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if e = c.SaveSettings(localdb.Settings{Name: "User", Mode: "hamachi"}); e != nil {
		t.Fatal(e)
	}
	s, e := c.State("")
	if e != nil || s.Network.ID == "" || s.Connectivity.Detected || s.NetworkError == "" || c.node != nil {
		t.Fatal(s, e)
	}
	if e = c.SaveSettings(localdb.Settings{Name: "User", Mode: "auto"}); e != nil {
		t.Fatal(e)
	}
	if c.node == nil {
		t.Fatal("cannot restore native mode")
	}
}

func TestRadminModePreservesProfileWithoutHamachi(t *testing.T) {
	provider := func() ([]adapters.Info, error) {
		return []adapters.Info{adapters.Classify(adapters.Info{ID: "radmin", Index: 9, DriverID: "Famatech Radmin VPN Ethernet Adapter", Up: true, IPv4: []string{"26.10.20.30"}})}, nil
	}
	dir := t.TempDir()
	c, e := OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	original, _ := c.State("")
	settings := localdb.Settings{Name: "Radmin user", Mode: "radmin", Bootstrap: []string{}, Relays: []string{}}
	if e = c.SaveSettings(settings); e != nil {
		t.Fatal(e)
	}
	state, e := c.State("")
	if e != nil || !state.Connectivity.Detected || state.Connectivity.DetectedProviders["hamachi"] || state.Connectivity.IP != "26.10.20.30" || state.Network.ID != original.Network.ID {
		t.Fatal(state, e)
	}
	// Fixture address is not installed on this host: fail closed instead of
	// requiring Hamachi, reverting to automatic, or inventing peer connectivity.
	if c.node != nil || state.NetworkError == "" || len(state.Network.Peers) != 0 {
		t.Fatal("unassigned virtual fixture was treated as connected")
	}
	if !c.allowedAddress("/ip4/26.1.2.3/tcp/4001", settings) || c.allowedAddress("/ip4/25.1.2.3/tcp/4001", settings) {
		t.Fatal("VPN endpoint policy mixed providers")
	}
	if e = c.db.Contact(localdb.Contact{ID: "fixture-contact", Name: "Saved contact", Scope: "radmin"}); e != nil {
		t.Fatal(e)
	}
	if e = c.db.AddMessage(localdb.Message{ID: "fixture-message", From: "fixture-contact", Peer: "fixture-contact", Text: "retained"}); e != nil {
		t.Fatal(e)
	}
	if e = c.Close(); e != nil {
		t.Fatal(e)
	}
	c, e = OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	state, e = c.State("fixture-contact")
	if e != nil || state.Settings.Mode != "radmin" || state.Network.ID != original.Network.ID || len(state.Contacts) != 1 || len(state.Messages) != 1 {
		t.Fatal("profile lost after Radmin restart", e)
	}
}
