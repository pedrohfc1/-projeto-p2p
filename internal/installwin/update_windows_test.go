//go:build windows

package installwin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	ole "github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/identity"
	"projetop2p/internal/installtxn"
	"projetop2p/internal/lifecycle"
	"projetop2p/internal/localdb"
	"syscall"
	"testing"
	"time"
)

func TestInstalledProcessChild(t *testing.T) {
	if os.Getenv("P2P_UPDATE_CHILD") != "yes" {
		t.Skip("child fixture only")
	}
	profile, root := os.Getenv("P2P_TEST_PROFILE"), os.Getenv("P2P_TEST_INSTALL")
	if os.Getenv("P2P_TEST_CHILD_MODE") == "legacy" {
		os.MkdirAll(profile, 0700)
		os.WriteFile(os.Getenv("P2P_TEST_READY"), []byte("legacy-ready"), 0600)
		time.Sleep(45 * time.Second)
		return
	}
	g, e := lifecycle.Acquire(profile, root)
	if e != nil {
		os.Exit(23)
	}
	db, e := localdb.Open(profile)
	if e != nil {
		g.Close()
		os.Exit(24)
	}
	if e = os.WriteFile(os.Getenv("P2P_TEST_READY"), []byte("ready"), 0600); e != nil {
		os.Exit(25)
	}
	select {
	case <-g.StopRequests():
	case <-time.After(45 * time.Second):
	}
	_ = db.Close()
	g.Close()
	os.WriteFile(filepath.Join(profile, "shutdown-clean"), []byte("yes"), 0600)
}
func fixture(t *testing.T) (destination, map[string][]byte, string) {
	t.Helper()
	base := t.TempDir()
	d := destination{dir: filepath.Join(base, "installation"), start: filepath.Join(base, "start"), desktop: filepath.Join(base, "desktop"), key: filepath.Join(base, "test-metadata.json"), profile: filepath.Join(base, "profile")}
	for _, p := range []string{d.dir, d.start, d.desktop} {
		if e := os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}

	exe, _ := os.Executable()
	binary, e := os.ReadFile(exe)
	if e != nil {
		t.Fatal(e)
	}
	files := map[string][]byte{"PeeTwoPee.exe": binary, "uninstall.exe": []byte("uninstaller-fixture"), "LICENSE.txt": []byte("license"), "THIRD_PARTY_LICENSES.zip": []byte("notices"), "easytier/etcat-p2p.exe": []byte("helper-fixture"), "EasyTier-corresponding-source.zip": []byte("sources")}
	return d, files, filepath.Join(base, "profile")
}
func startChild(t *testing.T, d destination, profile string, ready string, mode ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(filepath.Join(d.dir, "PeeTwoPee.exe"), "-test.run=^TestInstalledProcessChild$")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.Env = append(os.Environ(), "P2P_UPDATE_CHILD=yes", "P2P_TEST_PROFILE="+profile, "P2P_TEST_INSTALL="+d.dir, "P2P_TEST_READY="+ready)
	if len(mode) > 0 {
		cmd.Env = append(cmd.Env, "P2P_TEST_CHILD_MODE="+mode[0])
	}
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	return cmd
}
func waitFile(t *testing.T, p string) {
	t.Helper()
	for until := time.Now().Add(10 * time.Second); time.Now().Before(until); {
		if _, e := os.Stat(p); e == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("child not ready", p)
}
func TestWindowsCleanUpgradeRunningInstanceProfileAndRollback(t *testing.T) {
	d, files, profile := fixture(t)
	if e := installFixture(d, files, true, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	// Seed a real persistent identity/history in an isolated profile, never the actual user's profile.
	if _, e := identity.Load(profile); e != nil {
		t.Fatal(e)
	}
	db, e := localdb.Open(profile)
	if e != nil {
		t.Fatal(e)
	}
	db.SaveSettings(localdb.Settings{Name: "preserved settings", Port: 4001})
	db.AddMessage(localdb.Message{ID: "old-message", From: "sender", Peer: "contact", Text: "preserved history", Time: 123})
	db.Close()
	identityBefore, _ := os.ReadFile(filepath.Join(profile, "identity.key"))
	dbBefore, _ := os.ReadFile(filepath.Join(profile, "chat.db"))
	m, _ := installtxn.Read(d.dir)
	m.Version = "0.9.1"
	raw, _ := json.Marshal(m)
	installtxn.Atomic(filepath.Join(d.dir, installtxn.ManifestName), raw)
	testMeta, _ := json.Marshal(map[string]string{"version": "0.9.1"})
	os.WriteFile(d.key, testMeta, 0600)
	ready := filepath.Join(profile, "child.ready")
	child := startChild(t, d, profile, ready)
	t.Cleanup(func() {
		if child.ProcessState == nil {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	})
	waitFile(t, ready)
	dbBefore, _ = os.ReadFile(filepath.Join(profile, "chat.db"))
	duplicate := startChild(t, d, profile, filepath.Join(profile, "duplicate.ready"))
	e = duplicate.Wait()
	if e == nil || duplicate.ProcessState.ExitCode() != 23 {
		t.Fatal("duplicate instance accepted", e)
	}
	prompts := 0
	changed := map[string][]byte{}
	for n, b := range files {
		changed[n] = append(append([]byte(nil), b...), []byte("-new-revision")...)
	}
	if e = installFixture(d, changed, true, func(string) bool { prompts++; return true }); e != nil {
		t.Fatal(e)
	}
	if e = child.Wait(); e != nil {
		t.Fatal("graceful child exit failed", e)
	}
	if prompts != 1 {
		t.Fatal("force/extra shutdown prompt", prompts)
	}
	if _, e = os.Stat(filepath.Join(profile, "shutdown-clean")); e != nil {
		t.Fatal("graceful cleanup missing", e)
	}
	identityAfter, _ := os.ReadFile(filepath.Join(profile, "identity.key"))
	dbAfter, _ := os.ReadFile(filepath.Join(profile, "chat.db"))
	if !bytes.Equal(identityBefore, identityAfter) || !bytes.Equal(dbBefore, dbAfter) {
		t.Fatal("profile changed during update")
	}
	if e = installtxn.Verify(d.dir); e != nil {
		t.Fatal(e)
	}
	if e = installtxn.RollbackLast(d.dir, func(raw json.RawMessage) error { return restoreFixture(d, raw) }); e != nil {
		t.Fatal(e)
	}
	m, _ = installtxn.Read(d.dir)
	if m.Version != "0.9.1" {
		t.Fatal("rollback version", m.Version)
	}
	metaBytes, _ := os.ReadFile(d.key)
	var restored map[string]string
	json.Unmarshal(metaBytes, &restored)
	if restored["version"] != "0.9.1" {
		t.Fatal("metadata rollback", restored)
	}
	if e = installtxn.Verify(d.dir); e != nil {
		t.Fatal(e)
	}
	// A process restart using the rolled-back inventory can reacquire the same profile.
	nextReady := filepath.Join(profile, "restarted.ready")
	next := startChild(t, d, profile, nextReady)
	t.Cleanup(func() {
		if next.ProcessState == nil {
			_ = next.Process.Kill()
			_ = next.Wait()
		}
	})
	waitFile(t, nextReady)
	reset, e := lifecycle.RequestStop(d.dir)
	if e != nil {
		t.Fatal(e)
	}
	defer reset()
	if e = next.Wait(); e != nil {
		t.Fatal(e)
	}
	t.Log("PASS Windows isolated clean/overwrite installation, running hidden process graceful shutdown, duplicate rejection, byte-identical identity/history, injected metadata rollback and process restart; not tray UI/system reboot")
}
func TestWindowsLockedFileRevertsPartialReplacement(t *testing.T) {
	d, files, _ := fixture(t)
	if e := installFixture(d, files, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(d.dir, "easytier", "etcat-p2p.exe")
	name, _ := windows.UTF16PtrFromString(p)
	h, e := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer windows.CloseHandle(h)
	newFiles := map[string][]byte{}
	for n, b := range files {
		newFiles[n] = append(append([]byte(nil), b...), []byte("new")...)
	}
	if e = installFixture(d, newFiles, false, func(string) bool { return true }); e == nil {
		t.Fatal("locked update accepted")
	}
	if e = installtxn.Verify(d.dir); e != nil {
		t.Fatal("partial replacement not rolled back", e)
	}
	for n, want := range files {
		got, _ := os.ReadFile(filepath.Join(d.dir, filepath.FromSlash(n)))
		if !bytes.Equal(got, want) {
			t.Fatal("partial payload survived", n)
		}
	}
}

// Files/process lifecycle use real Windows APIs. Shell/registry metadata is injected
// because the desktop sandbox denies WScript.Shell; the separate capability test reports it.
func captureFixture(d destination, desktop bool) (metadata, error) {
	b, e := os.ReadFile(d.key)
	if os.IsNotExist(e) {
		b = []byte(`{}`)
	} else if e != nil {
		return metadata{}, e
	}
	return metadata{Links: map[string]linkValue{"metadata": {Present: true, Bytes: b}}}, nil
}
func finishFixture(d destination, m metadata, desktop bool, total int) error {
	b, _ := json.Marshal(map[string]string{"version": Version})
	return installtxn.Atomic(d.key, b)
}
func restoreFixture(d destination, raw json.RawMessage) error {
	var m metadata
	if e := json.Unmarshal(raw, &m); e != nil {
		return e
	}
	return installtxn.Atomic(d.key, m.Links["metadata"].Bytes)
}
func installFixture(d destination, files map[string][]byte, desktop bool, confirm func(string) bool) error {
	return installWithMetadata(d, files, desktop, confirm, captureFixture, finishFixture, restoreFixture)
}

func TestWindowsConsentLegacyForceAndUnrelatedProcess(t *testing.T) {
	d, files, profile := fixture(t)
	if e := installFixture(d, files, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	other, otherFiles, otherProfile := fixture(t)
	if e := installFixture(other, otherFiles, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	child := startChild(t, d, profile, filepath.Join(profile, "ready"), "legacy")
	defer func() {
		if child.ProcessState == nil {
			child.Process.Kill()
			child.Wait()
		}
	}()
	waitFile(t, filepath.Join(profile, "ready"))
	unrelated := startChild(t, other, otherProfile, filepath.Join(otherProfile, "ready"), "legacy")
	defer func() { unrelated.Process.Kill(); unrelated.Wait() }()
	waitFile(t, filepath.Join(otherProfile, "ready"))
	if e := installFixture(d, files, false, func(string) bool { return false }); e == nil {
		t.Fatal("refused consent updated installation")
	}
	if e := installtxn.Verify(d.dir); e != nil {
		t.Fatal(e)
	}
	items, _ := processes(d.dir)
	if len(items) != 1 {
		t.Fatal("refusal terminated process")
	}
	prompts := 0
	if e := installFixture(d, files, false, func(string) bool { prompts++; return true }); e != nil {
		t.Fatal(e)
	}
	if prompts != 2 {
		t.Fatal("legacy force lacked separate consent", prompts)
	}
	if e := child.Wait(); e != nil {
		t.Fatal(e)
	}
	items, _ = processes(other.dir)
	if len(items) != 1 {
		t.Fatal("unrelated same-name process terminated")
	}
}
func TestWindowsShellRegistryCapability(t *testing.T) {
	d, files, _ := fixture(t)
	d.key = `Software\PeeTwoPee\InstallerTests\Probe` + fmt.Sprint(time.Now().UnixNano())
	defer registry.DeleteKey(registry.CURRENT_USER, d.key)
	capabilityDenied := func(e error) bool {
		var oe *ole.OleError
		return errors.Is(e, windows.ERROR_ACCESS_DENIED) || errors.As(e, &oe) && oe.Code() == 0x80070005
	}
	key, _, e := registry.CreateKey(registry.CURRENT_USER, d.key, registry.SET_VALUE)
	if capabilityDenied(e) {
		t.Skip("HKCU write denied by sandbox; shell/registry integration pending")
	}
	if e != nil {
		t.Fatal(e)
	}
	key.Close()
	registry.DeleteKey(registry.CURRENT_USER, d.key)
	probe := filepath.Join(d.start, "probe.lnk")
	e = shortcut(probe, filepath.Join(d.dir, "PeeTwoPee.exe"), d.dir)
	if capabilityDenied(e) {
		t.Skip("WScript.Shell denied by execution sandbox; actual shell/registry installation remains pending")
	}
	if e != nil {
		t.Fatal(e)
	}
	os.Remove(probe)
	e = installAt(d, files, false, func(string) bool { return true })
	if e != nil {
		t.Fatal(e)
	}
	if e = installtxn.Verify(d.dir); e != nil {
		t.Fatal(e)
	}
}

func TestWindowsShortcutCapability(t *testing.T) {
	d, _, _ := fixture(t)
	path := filepath.Join(d.start, "probe.lnk")
	e := shortcut(path, filepath.Join(d.dir, "PeeTwoPee.exe"), d.dir)
	var oe *ole.OleError
	if e != nil && (errors.Is(e, windows.ERROR_ACCESS_DENIED) || errors.As(e, &oe) && oe.Code() == 0x80070005) {
		t.Skip("WScript.Shell blocked by sandbox; actual native shortcuts pending")
	}
	if e != nil {
		t.Fatal(e)
	}
	target, e := shortcutTarget(path)
	if e != nil || !samePath(target, filepath.Join(d.dir, "PeeTwoPee.exe")) {
		t.Fatal(target, e)
	}
}

// Actual production EXEs/payload bytes, isolated installation, injected metadata only.
func TestActualDesktopPayloadUpgrade(t *testing.T) {
	current := os.Getenv("P2P_ACTUAL_DESKTOP")
	previous := os.Getenv("P2P_PREVIOUS_DESKTOP")
	if current == "" || previous == "" {
		t.Skip("optional built production payload")
	}
	oldVersionCmd := exec.Command(previous, "--version")
	oldVersionCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	versionOut, e := oldVersionCmd.Output()
	if e != nil {
		t.Fatal(e)
	}
	oldVersion := string(bytes.TrimSpace(versionOut))
	if oldVersion == "" || oldVersion == Version {
		t.Fatalf("previous executable must report a distinct version, got %q", oldVersion)
	}
	d, files, profile := fixture(t)
	old, e := os.ReadFile(previous)
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := os.ReadFile(current)
	if e != nil {
		t.Fatal(e)
	}
	payloadDir := os.Getenv("P2P_ACTUAL_PAYLOAD_DIR")
	for name, source := range map[string]string{"uninstall.exe": "uninstall.exe", "LICENSE.txt": "LICENSE.txt", "THIRD_PARTY_LICENSES.zip": "THIRD_PARTY_LICENSES.zip", "easytier/etcat-p2p.exe": "etcat-p2p.exe", "EasyTier-corresponding-source.zip": "EasyTier-corresponding-source.zip"} {
		b, e := os.ReadFile(filepath.Join(payloadDir, source))
		if e != nil {
			t.Fatal(e)
		}
		files[name] = b
	}
	files["PeeTwoPee.exe"] = old
	if e = installFixture(d, files, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	m, _ := installtxn.Read(d.dir)
	m.Version = oldVersion
	raw, _ := json.Marshal(m)
	if e = installtxn.Atomic(filepath.Join(d.dir, installtxn.ManifestName), raw); e != nil {
		t.Fatal(e)
	}
	if _, e = identity.Load(profile); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(filepath.Join(profile, "identity.key"))
	files["PeeTwoPee.exe"] = fresh
	if e = installFixture(d, files, false, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(filepath.Join(profile, "identity.key"))
	if !bytes.Equal(before, after) {
		t.Fatal("identity changed")
	}
	if _, e = installtxn.VerifyExecutable(filepath.Join(d.dir, "PeeTwoPee.exe"), Version); e != nil {
		t.Fatal(e)
	}
	for _, option := range []string{"--version", "--permission-policy"} {
		cmd := exec.Command(filepath.Join(d.dir, "PeeTwoPee.exe"), option)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		out, e := cmd.Output()
		if e != nil {
			t.Fatal(e)
		}
		want := Version
		if option == "--permission-policy" {
			want = "prompt-v1"
		}
		if string(bytes.TrimSpace(out)) != want {
			t.Fatal("stale installed executable", option, string(out))
		}
	}
	t.Logf("PASS real desktop EXE upgrade %s -> %s with current helper/source/licenses/uninstaller, verified installed hashes and preserved isolated identity (simulated shell and registry, no GUI)", oldVersion, Version)
}
