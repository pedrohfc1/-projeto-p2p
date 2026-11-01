package network

import (
	"context"
	"errors"
	"projetop2p/internal/messaging"
	"sync/atomic"
	"testing"
	"time"
)

func TestCommittedReceiptRetryAndPersistenceFailure(t *testing.T) {
	var attempts atomic.Int32
	var writes atomic.Int32
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, OnMessage: func(m messaging.Message) error {
		if attempts.Add(1) == 1 {
			return errors.New("simulated disk failure")
		}
		writes.Add(1)
		return nil
	}})
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, _ := ParseAddress(b.Addresses()[0])
	if e := a.Connect(ctx, p); e != nil {
		t.Fatal(e)
	}
	m, e := a.Compose(b.Host.ID(), "persistent once")
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Deliver(ctx, m); e == nil {
		t.Fatal("ACK before persistence")
	}
	time.Sleep(120 * time.Millisecond)
	if e = a.Deliver(ctx, m); e != nil {
		t.Fatal("retry after persistence failure", e)
	}
	time.Sleep(120 * time.Millisecond)
	if e = a.Deliver(ctx, m); e != nil {
		t.Fatal("committed receipt retry", e)
	}
	if writes.Load() != 1 || attempts.Load() != 2 || len(b.Messages) != 1 {
		t.Fatal("duplicate callback or event", writes.Load(), attempts.Load(), len(b.Messages))
	}
}
