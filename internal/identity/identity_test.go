package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistent(t *testing.T) {
	d := t.TempDir()
	a, e := Load(d)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Load(d)
	if e != nil || !a.Equals(b) {
		t.Fatal("identity changed", e)
	}
	if e = os.WriteFile(filepath.Join(d, "identity.key"), []byte("broken"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Load(d); e == nil {
		t.Fatal("corrupt identity accepted")
	}
}

func TestOversizedIdentityIsPreservedAndRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity.key")
	content := make([]byte, 65537)
	if e := os.WriteFile(path, content, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(dir); e == nil {
		t.Fatal("oversized identity accepted")
	}
	info, e := os.Stat(path)
	if e != nil || info.Size() != int64(len(content)) {
		t.Fatal("identity overwritten", e)
	}
}
