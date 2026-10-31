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
