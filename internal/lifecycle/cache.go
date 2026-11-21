package lifecycle

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"projetop2p/internal/installtxn"
	"sort"
	"strings"
)

// AssetRevision identifies the actual embedded frontend, not a manually maintained label.
func AssetRevision(assets fs.FS) (string, error) {
	var names []string
	if e := fs.WalkDir(assets, "frontend/dist", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			names = append(names, p)
		}
		return nil
	}); e != nil {
		return "", e
	}
	sort.Strings(names)
	h := sha256.New()
	for _, p := range names {
		b, e := fs.ReadFile(assets, p)
		if e != nil {
			return "", e
		}
		fmt.Fprintln(h, p)
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RefreshCache renames only HTTP/compiled-code/GPU caches. Preferences, Local Storage,
// cookies, permissions, IndexedDB and the entire identity/chat profile remain untouched.
func RefreshCache(profile, revision string) error {
	if len(revision) != 64 {
		return fmt.Errorf("invalid frontend revision")
	}
	if _, e := hex.DecodeString(revision); e != nil {
		return e
	}
	root := filepath.Join(profile, "cache")
	marker, e := installtxn.Safe(root, "frontend-cache-revision.json")
	if e != nil {
		return e
	}
	if b, e := os.ReadFile(marker); e == nil {
		var old string
		if json.Unmarshal(b, &old) == nil && old == revision {
			return nil
		}
	}
	nonce := make([]byte, 8)
	if _, e := rand.Read(nonce); e != nil {
		return e
	}
	backupID := revision + "-" + hex.EncodeToString(nonce)
	names := []string{"Default/Cache", "Default/Code Cache", "Default/GPUCache", "GPUCache", "ShaderCache", "GrShaderCache"}
	baseNames := append([]string(nil), names...)
	for _, n := range baseNames {
		names = append(names, "EBWebView/"+n)
	}
	for _, n := range names {
		src, e := installtxn.Safe(filepath.Join(root, "webview2"), n)
		if e != nil {
			return e
		}
		if _, e = os.Stat(src); os.IsNotExist(e) {
			continue
		} else if e != nil {
			return e
		}
		dst, e := installtxn.Safe(root, "webview-cache-backups/"+backupID+"/"+n)
		if e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
			return e
		}
		if _, e = os.Stat(dst); e == nil {
			return fmt.Errorf("backup de cache já existe; recuperação manual necessária: %s", strings.TrimPrefix(n, "Default/"))
		}
		if e = os.Rename(src, dst); e != nil {
			return fmt.Errorf("cache bloqueado; encerre a outra instância: %w", e)
		}
	}
	b, _ := json.Marshal(revision)
	return installtxn.Atomic(marker, b)
}
