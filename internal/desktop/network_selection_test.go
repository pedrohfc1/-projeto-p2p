package desktop

import (
	"context"
	"projetop2p/internal/adapters"
	"sync/atomic"
	"testing"
)

func TestIntegratedPollingCachesAdapterDiscoveryAndFailedChoicePersists(t *testing.T) {
	var calls atomic.Int32
	dir := t.TempDir()
	c, e := OpenWithProvider(context.Background(), dir, func() ([]adapters.Info, error) { calls.Add(1); return []adapters.Info{}, nil })
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 5; i++ {
		s, e := c.State("")
		if e != nil || s.Settings.Mode != "integrated" {
			t.Fatal(s.Settings, e)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("enumerated adapters %d times for unchanged integrated polling", calls.Load())
	}
	before, _ := c.State("")
	if e = c.SelectNetwork("interface", "missing-fixture"); e == nil {
		t.Fatal("missing interface must be unavailable")
	}
	after, _ := c.State("")
	if after.Settings.Mode != "interface" || after.Settings.InterfaceID != "missing-fixture" || after.Network.ID != before.Network.ID {
		t.Fatal("choice or identity lost")
	}
	c.Close()
	c, e = OpenWithProvider(context.Background(), dir, func() ([]adapters.Info, error) { return nil, nil })
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	s, _ := c.State("")
	if s.Settings.Mode != "interface" || s.Network.ID != before.Network.ID {
		t.Fatal("failed choice overwritten on restart")
	}
}
