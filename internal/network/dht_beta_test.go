//go:build !legacydht

package network

import (
	"context"
	"testing"
	"time"
)

func TestBetaQuarantinesProviderDHTButKeepsInvites(t *testing.T) {
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer b.Close()
	if a.DHT != nil || a.Status().DHTEnabled {
		t.Fatal("affected provider DHT linked/enabled in beta")
	}
	p, e := ParseAddress(b.Addresses()[0])
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = a.Connect(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e = a.Send(ctx, b.Host.ID(), "beta direct delivery"); e != nil {
		t.Fatal(e)
	}
}
