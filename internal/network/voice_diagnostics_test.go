package network

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	"testing"
	"time"
)

func TestMediaDiagnosticsBoundedCounters(t *testing.T) {
	n := &Node{}
	id := peer.ID("fixture")
	n.observeMediaOpen(id, time.Millisecond*5, nil)
	n.observeMediaOpen(id, time.Millisecond*10, context.DeadlineExceeded)
	n.observeMediaOpen(id, time.Millisecond, errors.New("ordinary error"))
	v := n.mediaObservations.values[id]
	if v.OpenAttempts != 3 || v.OpenFailures != 2 || v.Timeouts != 1 || *v.OpenMs != 1 {
		t.Fatalf("%+v", v)
	}
	for i := 0; i < 300; i++ {
		n.observeMediaOpen(peer.ID(string(rune(i+1000))), time.Millisecond, nil)
	}
	if len(n.mediaObservations.values) != 128 {
		t.Fatal(len(n.mediaObservations.values))
	}
}
