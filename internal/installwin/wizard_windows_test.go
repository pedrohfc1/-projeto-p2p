//go:build windows

package installwin

import (
	"encoding/json"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"projetop2p/internal/installtxn"
	"testing"
)

func TestWizardCustomFolderAndProfileIsolation(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	t.Setenv("PROJETOP2P_DATA", profile)
	for _, bad := range []string{profile, filepath.Dir(profile), filepath.Join(profile, "app"), `C:\`, "relative"} {
		if _, e := ValidateFolder(bad); e == nil {
			t.Fatalf("accepted unsafe %s", bad)
		}
	}
	wanted := filepath.Join(t.TempDir(), "custom app")
	if got, e := ValidateFolder(wanted); e != nil || got != wanted {
		t.Fatalf("%s %v", got, e)
	}
}
func TestWizardShortcutOptionsRollback(t *testing.T) {
	d, files, _ := fixture(t)
	d.key = `Software\PeeTwoPeeWizardTest\` + filepath.Base(t.TempDir())
	defer registry.DeleteKey(registry.CURRENT_USER, d.key)
	d.startup = filepath.Join(filepath.Dir(d.dir), "startup")
	os.MkdirAll(d.startup, 0700)
	on, off := true, false
	d.autoStart = &on
	d.startMenu = &on
	if e := installAt(d, files, true, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"start", "desktop", "startup"} {
		if _, e := os.Stat(d.linkPaths()[id]); e != nil {
			t.Fatal(id, e)
		}
	}
	d.autoStart = &off
	d.startMenu = &off
	if e := installAt(d, files, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"start", "desktop", "startup"} {
		if _, e := os.Stat(d.linkPaths()[id]); !os.IsNotExist(e) {
			t.Fatal("option did not remove owned shortcut", id, e)
		}
	}
	if e := installtxn.RollbackLast(d.dir, func(raw json.RawMessage) error { return restoreMetadata(d, raw) }); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"start", "desktop", "startup"} {
		target, e := shortcutTarget(d.linkPaths()[id])
		if e != nil || !samePath(target, filepath.Join(d.dir, Product+".exe")) {
			t.Fatal("shortcut not restored", id, e)
		}
	}
}
