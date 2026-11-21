package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstanceLocksAndSeparateProfiles(t *testing.T) {
	root := t.TempDir()
	a, e := Acquire(filepath.Join(root, "one"), root)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if duplicate, e := Acquire(filepath.Join(root, "one"), root); e == nil {
		duplicate.Close()
		t.Fatal("duplicate profile allowed")
	}
	b, e := Acquire(filepath.Join(root, "two"), root)
	if e != nil {
		t.Fatal(e)
	}
	b.Close()
}
func TestCacheRefreshPreservesLocalStoragePermissionsAndProfile(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"identity.key", "chat.db", "cache/webview2/Default/Local Storage/keep", "cache/webview2/Default/Preferences", "cache/webview2/Default/Cache/stale", "cache/webview2/Default/Code Cache/stale", "cache/webview2/EBWebView/Default/Code Cache/stale", "cache/webview2/EBWebView/Default/Local Storage/keep", "cache/webview2/EBWebView/Default/Preferences"} {
		path := filepath.Join(root, filepath.FromSlash(n))
		os.MkdirAll(filepath.Dir(path), 0700)
		os.WriteFile(path, []byte(n), 0600)
	}
	if e := RefreshCache(root, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	for _, n := range []string{"identity.key", "chat.db", "cache/webview2/Default/Local Storage/keep", "cache/webview2/Default/Preferences", "cache/webview2/EBWebView/Default/Local Storage/keep", "cache/webview2/EBWebView/Default/Preferences"} {
		if _, e := os.Stat(filepath.Join(root, filepath.FromSlash(n))); e != nil {
			t.Fatal("user state lost", n, e)
		}
	}
	if _, e := os.Stat(filepath.Join(root, "cache/webview2/Default/Cache/stale")); !os.IsNotExist(e) {
		t.Fatal("stale HTTP cache remained")
	}
	if e := RefreshCache(root, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
}
