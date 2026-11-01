package security

import (
	"testing"
	"time"
)

func TestStreamGateBoundsAndRelease(t *testing.T) {
	var g StreamGate
	now := time.Now()
	a, ok := g.Acquire("a", now)
	if !ok {
		t.Fatal("first denied")
	}
	b, ok := g.Acquire("a", now)
	if !ok {
		t.Fatal("second denied")
	}
	if _, ok = g.Acquire("a", now); ok {
		t.Fatal("per-peer stream cap ignored")
	}
	a()
	a()
	b()
	for i := 0; i < 62; i++ {
		release, ok := g.Acquire("a", now)
		if !ok {
			t.Fatal("burst reduced unexpectedly", i)
		}
		release()
	}
	if _, ok = g.Acquire("a", now); ok {
		t.Fatal("rate cap ignored")
	}
	release, ok := g.Acquire("a", now.Add(time.Second))
	if !ok {
		t.Fatal("no refill")
	}
	release()
}
