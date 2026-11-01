package overlay

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealSupervisorRecovery(t *testing.T) {
	dir := os.Getenv("P2P_ETCAT_TEST_DIR")
	if dir == "" {
		t.Skip("real helper optional")
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	profile, e := os.MkdirTemp(root, "overlay-recovery-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(profile)
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	m, e := Open(ctx, profile, filepath.Join(dir, "etcat-p2p.exe"), "")
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	if e = m.Start(ctx, l.Addr().(*net.TCPAddr).Port); e != nil {
		t.Fatal(e)
	}
	token := m.Token()
	m.mu.Lock()
	process := m.server.cmd.Process
	m.mu.Unlock()
	rss, cpu, ok := childUsage(process.Pid)
	t.Logf("helper RSS=%d bytes CPU=%.3f cumulative seconds verified=%v", rss, cpu, ok)
	if e = process.Kill(); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(40 * time.Second)
	for {
		state := m.State()
		if state.Restarts > 0 && state.Running && m.Token() == token {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("supervisor recovery", state)
		}
		time.Sleep(time.Second)
	}
	m.Close()
	if m.State().Running {
		t.Fatal("helper leaked")
	}
	entries, _ := os.ReadDir(profile)
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			t.Fatal("plaintext material leaked")
		}
	}
}
