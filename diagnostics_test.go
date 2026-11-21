package main

import (
	"context"
	"projetop2p/internal/desktop"
	"projetop2p/internal/diagnostics"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticSamplerUsesRealCoreAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	core, err := desktop.Open(ctx, t.TempDir())
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	a := &App{core: core}
	diagnostics.Local.Clear()
	a.startDiagnostics(ctx)
	deadline := time.Now().Add(6 * time.Second)
	var b diagnostics.Batch
	for time.Now().Before(deadline) {
		b = a.DiagnosticEvents(0)
		if len(b.Events) >= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	a.backgroundWG.Wait()
	defer core.Close()
	if len(b.Events) < 3 {
		t.Fatal("no real operational snapshots", b)
	}
	found := map[string]bool{}
	for _, e := range b.Events {
		found[e.Category] = true
	}
	if !found["voice"] || !found["video"] || !found["network"] {
		t.Fatal(found)
	}
	last := a.ClearDiagnostics()
	if len(a.DiagnosticEvents(last).Events) != 0 {
		t.Fatal("sampler did not stop or clear failed")
	}
}

func TestDiagnosticClipboardRejectsUnboundedInputBeforeOSAccess(t *testing.T) {
	a := &App{}
	if a.CopyDiagnostics(strings.Repeat("x", 12*1024*1024+1)) == nil {
		t.Fatal("unbounded clipboard request")
	}
}
