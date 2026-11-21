package installtxn

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func payload(tag string) map[string][]byte {
	out := map[string][]byte{}
	for _, n := range Names {
		if n != ManifestName && n != "ProjetoP2P.exe" {
			out[n] = []byte(tag + ":" + n)
		}
	}
	return out
}
func TestCleanUpgradeAndUnknownDataPreserved(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "my-data", "identity.key")
	os.MkdirAll(filepath.Dir(profile), 0700)
	os.WriteFile(profile, []byte("private-profile-sentinel"), 0600)
	os.WriteFile(filepath.Join(root, "unrelated.dll"), []byte("leave"), 0600)
	if e := Apply(root, "old", payload("old"), nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := Apply(root, "new", payload("new"), nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := Verify(root); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{profile, filepath.Join(root, "unrelated.dll")} {
		if _, e := os.Stat(name); e != nil {
			t.Fatal("user data removed", e)
		}
	}
	if e := RollbackLast(root, nil); e != nil {
		t.Fatal(e)
	}
	m, e := Read(root)
	if e != nil || m.Version != "old" {
		t.Fatal(m, e)
	}
	if e = Verify(root); e != nil {
		t.Fatal(e)
	}
}
func TestFailedFinalizeRestoresAllFilesAndMetadata(t *testing.T) {
	root := t.TempDir()
	Apply(root, "old", payload("old"), nil, nil, nil)
	state := "old"
	restore := func(raw json.RawMessage) error { return json.Unmarshal(raw, &state) }
	raw, _ := json.Marshal(state)
	e := Apply(root, "new", payload("new"), raw, func() error { state = "partially-changed"; return errors.New("registry failure") }, restore)
	if e == nil || state != "old" {
		t.Fatal(e, state)
	}
	if e = Verify(root); e != nil {
		t.Fatal(e)
	}
	m, _ := Read(root)
	if m.Version != "old" {
		t.Fatal(m)
	}
}
func TestCrashJournalBlocksStartupAndRecovers(t *testing.T) {
	root := t.TempDir()
	Apply(root, "old", payload("old"), nil, nil, nil)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected simulated interruption")
			}
		}()
		Apply(root, "new", payload("new"), nil, func() error { panic("process died") }, nil)
	}()
	if Verify(root) == nil {
		t.Fatal("partial transaction allowed startup")
	}
	if e := Recover(root, nil); e != nil {
		t.Fatal(e)
	}
	if e := Verify(root); e != nil {
		t.Fatal(e)
	}
	m, _ := Read(root)
	if m.Version != "old" {
		t.Fatal(m)
	}
}
func TestBackupCorruptionAndPathEscapeFailClosed(t *testing.T) {
	root := t.TempDir()
	Apply(root, "old", payload("old"), nil, nil, nil)
	func() {
		defer func() { _ = recover() }()
		Apply(root, "new", payload("new"), nil, func() error { panic("crash") }, nil)
	}()
	j, e := readJournal(root)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, ".p2p-backups", j.Backup, "PeeTwoPee.exe"), []byte("damaged"), 0600)
	before, _ := os.ReadFile(filepath.Join(root, "PeeTwoPee.exe"))
	if Recover(root, nil) == nil {
		t.Fatal("corrupt backup accepted")
	}
	after, _ := os.ReadFile(filepath.Join(root, "PeeTwoPee.exe"))
	if string(before) != string(after) {
		t.Fatal("files touched before full backup validation")
	}
	for _, path := range []string{"../profile/chat.db", "/outside", "C:/outside", "..\\outside"} {
		if _, e := Safe(root, path); e == nil {
			t.Fatal("escape accepted", path)
		}
	}
}
func TestInstallIntegrityRejectsMixedHelper(t *testing.T) {
	root := t.TempDir()
	if e := Apply(root, "v", payload("v"), nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "easytier", "etcat-p2p.exe"), []byte("stale helper"), 0600)
	if Verify(root) == nil {
		t.Fatal("mixed installation accepted")
	}
}

func TestInvalidManualBackupLeavesCurrentInstallUsable(t *testing.T) {
	root := t.TempDir()
	if e := Apply(root, "old", payload("old"), nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := Apply(root, "new", payload("new"), nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".p2p-backups"))
	for _, entry := range entries {
		raw, _ := os.ReadFile(filepath.Join(root, ".p2p-backups", entry.Name(), "rollback.json"))
		var j Journal
		json.Unmarshal(raw, &j)
		if j.Previous["PeeTwoPee.exe"].Exists {
			os.WriteFile(filepath.Join(root, ".p2p-backups", entry.Name(), "PeeTwoPee.exe"), []byte("damaged"), 0600)
		}
	}
	if e := RollbackLast(root, nil); e == nil {
		t.Fatal("corrupt manual backup accepted")
	}
	if e := Verify(root); e != nil {
		t.Fatal("failed manual rollback broke current install", e)
	}
}
