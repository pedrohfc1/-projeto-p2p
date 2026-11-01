package desktop

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/adapters"
	"projetop2p/internal/identity"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"strings"
	"testing"
	"time"
)

func TestRadminInviteRequiresExplicitSelectionAndPinsGUID(t *testing.T) {
	provider := func() ([]adapters.Info, error) {
		return []adapters.Info{
			adapters.Classify(adapters.Info{ID: "h", Name: "Hamachi", DriverID: "hamachi", Up: true, IPv4: []string{"25.1.2.3"}}),
			adapters.Classify(adapters.Info{ID: "r-guid", Name: "Renamed Radmin", Index: 42, DriverID: "radminvpn", Up: true, IPv4: []string{"169.254.4.5", "10.5.0.2", "26.10.20.30"}})}, nil
	}
	dir := t.TempDir()
	c, e := OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if c != nil {
			c.Close()
		}
	}()
	before, _ := c.State("")
	key, e := identity.Load(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	id, _ := peer.IDFromPrivateKey(key)
	raw, e := invite.NewScoped(key, "Remote", []string{"/ip4/26.1.2.3/tcp/4001/p2p/" + id.String()}, "radmin")
	if e != nil {
		t.Fatal(e)
	}
	r, e := c.ConnectInvite(raw)
	if e != nil || r.RequiredMode != "radmin" || r.Code != "mode_mismatch" || r.ContactSaved {
		t.Fatal(r, e)
	}
	s, _ := c.State("")
	if mode(s.Settings) != "auto" || len(s.Contacts) != 0 || s.Network.ID != before.Network.ID {
		t.Fatal("invite silently modified profile")
	}
	if e = c.SaveSettings(localdb.Settings{Name: "Saved", Mode: "radmin"}); e == nil {
		t.Fatal("unassigned fixture should fail to bind")
	}
	s, _ = c.State("")
	if s.Settings.InterfaceID != "r-guid" || s.Settings.Port != 4001 || s.Connectivity.IP != "26.10.20.30" || s.Settings.Mode != "radmin" {
		t.Fatal(s.Settings, s.Connectivity)
	}
	cfg := c.config(s.Settings)
	if !cfg.LocalOnly || cfg.ScopedMode != "radmin" || len(cfg.Bootstrap) != 0 || len(cfg.Relays) != 0 || cfg.Listen[0] != "/ip4/26.10.20.30/tcp/4001" {
		t.Fatal(cfg)
	}
	c.Close()
	c = nil
	c, e = OpenWithProvider(context.Background(), dir, provider)
	if e != nil {
		t.Fatal(e)
	}
	s, _ = c.State("")
	if s.Settings.InterfaceID != "r-guid" || s.Settings.Mode != "radmin" || s.Network.ID != before.Network.ID {
		t.Fatal("selection or identity lost")
	}
}
func TestConnectionFailureReportsExactEndpointWithoutSavingRawContact(t *testing.T) {
	provider := func() ([]adapters.Info, error) {
		return []adapters.Info{{ID: "lo", Name: "Loopback test", Index: 7, Up: true, IPv4: []string{"127.0.0.1"}}}, nil
	}
	c, e := OpenWithProvider(context.Background(), t.TempDir(), provider)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	localPort := freePort(t)
	if e = c.SaveSettings(localdb.Settings{Name: "Tester", Mode: "interface", InterfaceID: "lo", Port: localPort}); e != nil {
		t.Fatal(e)
	}
	key, e := identity.Load(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	id, _ := peer.IDFromPrivateKey(key)
	closed := freePort(t)
	raw := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", closed, id)
	r, e := c.ConnectInvite(raw)
	if e != nil || r.Connected || r.Authenticated || r.ContactSaved || r.Code != "dial_failed" || r.InterfaceID != "lo" || r.InterfaceIndex != 7 || r.LocalPort != localPort || len(r.Targets) != 1 || r.Targets[0].Port != closed || !strings.Contains(r.Message, "127.0.0.1") {
		t.Fatal(r, e)
	}
	s, _ := c.State("")
	if len(s.Contacts) != 0 || s.Diagnostic == nil || s.Diagnostic.Code != "dial_failed" {
		t.Fatal("failed raw peer saved or diagnostic absent")
	}
	// The host's real invite contains its listener port and persistent Peer ID.
	own, e := c.Invite()
	if e != nil {
		t.Fatal(e)
	}
	i, e := invite.Parse(own, time.Now())
	if e != nil || i.ID != s.Network.ID || len(i.Addresses) != 1 || i.Addresses[0] != fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", localPort, s.Network.ID) {
		t.Fatal(i, e)
	}
}
