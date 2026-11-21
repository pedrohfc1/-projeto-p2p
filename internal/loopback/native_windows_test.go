//go:build windows && !nosystemaudio

package loopback

import (
	"context"
	"os"
	"testing"
)

func TestWindowsHelperCatalogAndOwnProcessRejection(t *testing.T) {
	sources, e := Sources(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal("native helper/.NET source enumeration failed", e)
	}
	t.Logf("native helper executed; selectable applications=%d; no capture was started", len(sources))
	if !relatedProcess(uint32(os.Getpid())) {
		t.Fatal("own process not excluded")
	}
	if _, e = Start(context.Background(), t.TempDir(), "process", os.Getpid()); e == nil {
		t.Fatal("captured own application")
	}
}
