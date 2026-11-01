package network

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	net "github.com/libp2p/go-libp2p/core/network"
	"projetop2p/internal/messaging"
	"strings"
	"testing"
	"time"
)

func wait(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("condition timed out")
}

func TestRejectInvalidFrames(t *testing.T) {
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, e := ParseAddress(b.Addresses()[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Connect(ctx, p); e != nil {
		t.Fatal(e)
	}
	m, e := messaging.New(a.Host.Peerstore().PrivKey(a.Host.ID()), b.Host.ID(), "original")
	if e != nil {
		t.Fatal(e)
	}
	m.Text = "tampered"
	bad, _ := json.Marshal(m)
	for _, frame := range []string{"not-json\n", strings.Repeat("x", messaging.MaxFrame+1) + "\n", string(bad) + "\n"} {
		s, e := a.Host.NewStream(ctx, b.Host.ID(), Protocol)
		if e != nil {
			t.Fatal(e)
		}
		s.SetDeadline(time.Now().Add(time.Second))
		_, _ = s.Write([]byte(frame))
		ack, _ := bufio.NewReader(s).ReadString('\n')
		s.Close()
		if ack == "OK\n" {
			t.Fatal("invalid frame acknowledged")
		}
	}
	select {
	case m := <-b.Messages:
		t.Fatal("invalid message delivered", m)
	default:
	}
}

func TestIndependentBootstrapFallback(t *testing.T) {
	cfg := func() Config {
		return Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, Interval: time.Second}
	}
	gone := create(t, cfg())
	dead := gone.Addresses()
	if e := gone.Close(); e != nil {
		t.Fatal(e)
	}
	live := create(t, cfg())
	defer live.Close()
	c := cfg()
	c.Bootstrap = append(dead, live.Addresses()...)
	n := create(t, c)
	defer n.Close()
	wait(t, func() bool { return n.Host.Network().Connectedness(live.Host.ID()) == net.Connected })
}

func TestNoAckWhenDurableReceiverFails(t *testing.T) {
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, OnMessage: func(messaging.Message) error { return errors.New("disk unavailable") }})
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, e := ParseAddress(b.Addresses()[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Connect(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e = a.Send(ctx, b.Host.ID(), "must not acknowledge"); e == nil {
		t.Fatal("failed persistence was acknowledged")
	}
	select {
	case <-b.Messages:
		t.Fatal("failed message was delivered")
	default:
	}
}
func create(t *testing.T, c Config) *Node {
	t.Helper()
	n, e := New(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func TestThreeNodesDiscoveryRecovery(t *testing.T) {
	if !DHTEnabled {
		t.Skip("provider DHT quarantined in beta; exercised by -tags legacydht")
	}
	cfg := func() Config {
		return Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, Interval: time.Second}
	}
	a := create(t, cfg())
	defer a.Close()
	cb := cfg()
	cb.Bootstrap = a.Addresses()
	b := create(t, cb)
	defer b.Close()
	cc := cfg()
	cc.Bootstrap = a.Addresses()
	c := create(t, cc)
	defer func() {
		if c != nil {
			c.Close()
		}
	}()
	// B and C have only A's address: their mutual connection must come from DHT discovery.
	wait(t, func() bool { return b.Host.Network().Connectedness(c.Host.ID()) == net.Connected })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e := b.Send(ctx, c.Host.ID(), "três nós"); e != nil {
		t.Fatal(e)
	}
	select {
	case m := <-c.Messages:
		if m.Text != "três nós" {
			t.Fatal(m)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Replay exact same signed frame: only its first delivery may be acknowledged.
	time.Sleep(150 * time.Millisecond)
	m, e := messaging.New(b.Host.Peerstore().PrivKey(b.Host.ID()), c.Host.ID(), "replay")
	if e != nil {
		t.Fatal(e)
	}
	transmit := func() bool {
		s, e := b.Host.NewStream(ctx, c.Host.ID(), Protocol)
		if e != nil {
			t.Fatal(e)
		}
		defer s.Close()
		s.SetDeadline(time.Now().Add(time.Second))
		if e = json.NewEncoder(s).Encode(m); e != nil {
			t.Fatal(e)
		}
		ack, e := bufio.NewReader(s).ReadString('\n')
		return e == nil && ack == "OK\n"
	}
	if !transmit() {
		t.Fatal("first delivery failed")
	}
	<-c.Messages
	time.Sleep(150 * time.Millisecond)
	if !transmit() {
		t.Fatal("committed replay not acknowledged")
	}
	select {
	case <-c.Messages:
		t.Fatal("replay delivered twice")
	default:
	}
	// Unexpected peer loss, same identity restarted at a new ephemeral port.
	oldID := c.Host.ID()
	oldAddr := c.Addresses()[0]
	if e = c.Close(); e != nil {
		t.Fatal(e)
	}
	c = nil
	wait(t, func() bool { return b.Host.Network().Connectedness(oldID) != net.Connected })
	c = create(t, cc)
	if c.Host.ID() != oldID {
		t.Fatal("restart changed identity")
	}
	if c.Addresses()[0] == oldAddr {
		t.Log("OS reused port; changed-address case not exercised")
	}
	wait(t, func() bool { return b.Host.Network().Connectedness(c.Host.ID()) == net.Connected })
	if e := c.Send(context.Background(), b.Host.ID(), "recuperado"); e != nil {
		t.Fatal(e)
	}
	select {
	case <-b.Messages:
	case <-time.After(5 * time.Second):
		t.Fatal("no recovered message")
	}
	// Force disconnection while both nodes run: known-peer retries restore connectivity.
	_ = b.Host.Network().ClosePeer(c.Host.ID())
	wait(t, func() bool { return b.Host.Network().Connectedness(c.Host.ID()) == net.Connected })
}
