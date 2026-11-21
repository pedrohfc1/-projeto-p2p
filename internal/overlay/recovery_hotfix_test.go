package overlay

import (
	"context"
	"testing"
	"time"
)

func TestPeerStartupWaitDoesNotHoldGlobalMutex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{ctx: ctx, store: saved{Routes: map[string]string{"peer": "route"}}, sessions: map[string]*session{}, starting: map[string]chan struct{}{"peer": make(chan struct{})}}
	done := make(chan error, 1)
	go func() { _, e := m.dialSession(ctx, "peer", "route"); done <- e }()
	time.Sleep(20 * time.Millisecond)
	status := make(chan struct{})
	go func() { m.mu.Lock(); m.mu.Unlock(); close(status) }()
	select {
	case <-status:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("startup blocked unrelated peers")
	}
	cancel()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("canceled attempt accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("startup waiter leaked")
	}
}
