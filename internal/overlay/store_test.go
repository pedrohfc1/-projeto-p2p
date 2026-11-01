package overlay

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSealedTransportStateAndPrivateMaterial(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("private beta Windows")
	}
	root, e := filepath.Abs("../../data")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	dir, e := os.MkdirTemp(root, "overlay-store-test-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if filepath.Dir(dir) != root {
			t.Fatal("unexpected temp path")
		}
		os.RemoveAll(dir)
	})
	s, e := loadStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(dir + "/integrated-transport.sealed")
	if e != nil || bytes.Contains(b, []byte("network_secret")) {
		t.Fatal("plaintext transport secrets", e)
	}
	again, e := loadStore(dir)
	if e != nil || !bytes.Equal(s.Server, again.Server) {
		t.Fatal("transport identity not persistent", e)
	}
	path, closeFile, e := privateMaterial(dir, s.Server)
	if e != nil {
		t.Fatal(e)
	}
	st, e := os.Stat(path)
	if e != nil || st.Size() != int64(len(s.Server)) {
		closeFile()
		t.Fatal("temporary material missing", e)
	}
	closeFile()
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("private temporary material not deleted", e)
	}
}
