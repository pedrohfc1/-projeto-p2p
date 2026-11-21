package main

import (
	"context"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"tailscale.com/ipn/ipnstate"
	"testing"
	"time"
)

func TestOnlyOfficialAuthURLsAndConfirmedPaths(t *testing.T) {
	for _, raw := range []string{"http://login.tailscale.com/a/x", "https://login.tailscale.com.evil/a/x", "https://user@login.tailscale.com/a/x", "https://login.tailscale.com:444/a/x"} {
		if authURL(raw) {
			t.Fatal(raw)
		}
	}
	if !authURL("https://login.tailscale.com/a/local-fixture") {
		t.Fatal("official URL")
	}
	if pathOf(nil) != "desconhecido" || pathOf(&ipnstate.PingResult{PeerRelay: "x"}) != "peer-relay" || pathOf(&ipnstate.PingResult{Endpoint: "x"}) != "direto confirmado por ping disco" || pathOf(&ipnstate.PingResult{DERPRegionID: 1}) != "DERP" {
		t.Fatal("classification")
	}
}
func TestStateRefusesUnrelatedData(t *testing.T) {
	base := t.TempDir()
	os.WriteFile(filepath.Join(base, "personal"), []byte("preserve"), 0600)
	if stateDir(base) == nil {
		t.Fatal("unrelated directory accepted")
	}
	fresh := filepath.Join(t.TempDir(), "own")
	if e := stateDir(fresh); e != nil {
		t.Fatal(e)
	}
	if e := stateDir(fresh); e != nil {
		t.Fatal(e)
	}
}
func TestTailAddressPolicy(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "192.168.1.2", "169.254.1.1", "8.8.8.8"} {
		if tailIP(netip.MustParseAddr(ip)) {
			t.Fatal(ip)
		}
	}
	if !tailIP(netip.MustParseAddr("100.80.1.1")) {
		t.Fatal("tailnet")
	}
	if _, e := (tailDialer{}).DialContext(context.Background(), "tcp", "127.0.0.1:1"); e == nil {
		t.Fatal("native fallback permitted")
	}
}
func TestBridgeBoundedShutdown(t *testing.T) {
	target, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer target.Close()
	go func() {
		c, e := target.Accept()
		if e == nil {
			defer c.Close()
			b := make([]byte, 4)
			c.Read(b)
			c.Write(b)
		}
	}()
	incoming, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closeBridge := bridge(ctx, incoming, target.Addr().String())
	c, e := net.Dial("tcp", incoming.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second))
	c.Write([]byte("test"))
	b := make([]byte, 4)
	if _, e = c.Read(b); e != nil || string(b) != "test" {
		t.Fatal(e, string(b))
	}
	done := make(chan struct{})
	go func() { closeBridge(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("bridge leaked")
	}
}
