package security

import (
	"fmt"
	"testing"
	"time"
)

func TestGuard(t *testing.T) {
	var g Guard
	now := time.Now()
	if !g.Accept("a", "1", now) {
		t.Fatal("first rejected")
	}
	if g.Accept("a", "1", now.Add(time.Second)) {
		t.Fatal("replay")
	}
	if g.Accept("a", "2", now.Add(time.Millisecond)) {
		t.Fatal("rate limit")
	}
	if !g.Accept("b", "1", now) {
		t.Fatal("senders conflated")
	}
	for i := 0; i < 254; i++ {
		if !g.Accept(fmt.Sprint(i), "x", now) {
			t.Fatal(i)
		}
	}
	if g.Accept("overflow", "x", now) {
		t.Fatal("unbounded senders")
	}
	if !g.Accept("a", "capacity-existing", now.Add(time.Second)) {
		t.Fatal("existing sender rejected at capacity")
	}
	if !g.Accept("new", "x", now.Add(6*time.Minute)) {
		t.Fatal("no expiry")
	}
}
