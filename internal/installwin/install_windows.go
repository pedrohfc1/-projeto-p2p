// Package installwin implements a per-user installer without changing VPN,
// firewall, machine-wide permissions or the user's persistent application data.
package installwin

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/installtxn"
	"runtime"
	"strings"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const Product = "PeeTwoPee"
const Version = "0.11.0"
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ProjetoP2PProjetoP2P`

func Folder() (string, error) {
	base, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	dir := filepath.Join(base, "Programs", "ProjetoP2P")
	if key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE); err == nil {
		if value, _, err := key.GetStringValue("InstallLocation"); err == nil && filepath.IsAbs(value) {
			dir = filepath.Clean(value)
		}
		key.Close()
	}
	return ValidateFolder(dir)
}

// ValidateFolder never permits executables and persistent data to overlap.
func ValidateFolder(dir string) (string, error) {
	if !filepath.IsAbs(dir) || strings.ContainsAny(dir, "\"\x00") {
		return "", fmt.Errorf("escolha uma pasta absoluta válida")
	}
	dir = filepath.Clean(dir)
	if len(strings.TrimRight(dir, `\/`)) <= len(filepath.VolumeName(dir)) {
		return "", fmt.Errorf("não instale na raiz do disco")
	}
	profile := os.Getenv("PROJETOP2P_DATA")
	if profile == "" {
		config, e := os.UserConfigDir()
		if e != nil {
			return "", e
		}
		profile = filepath.Join(config, "ProjetoP2P")
	}
	var e error
	profile, e = filepath.Abs(profile)
	if e != nil {
		return "", e
	}
	for _, pair := range [][2]string{{dir, profile}, {profile, dir}} {
		rel, e := filepath.Rel(pair[0], pair[1])
		if e == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return "", fmt.Errorf("instalação e perfil precisam estar em diretórios separados")
		}
	}
	return dir, nil
}
func Message(text string, question bool) bool {
	flags := uintptr(0x40)
	if question {
		flags = 0x24
	}
	t, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString("PeeTwoPee — Instalação 0.11.0")
	result, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(title)), flags)
	return !question || result == 6
}
func locations() (start, desktop string, err error) {
	start, err = windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	if err != nil {
		return
	}
	desktop, err = windows.KnownFolderPath(windows.FOLDERID_Desktop, 0)
	return
}
func shortcut(path, target, dir string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := initializeCOM(); e != nil {
		return e
	}
	defer ole.CoUninitialize()
	object, e := oleutil.CreateObject("WScript.Shell")
	if e != nil {
		return e
	}
	defer object.Release()
	shell, e := object.QueryInterface(ole.IID_IDispatch)
	if e != nil {
		return e
	}
	defer shell.Release()
	result, e := oleutil.CallMethod(shell, "CreateShortcut", path)
	if e != nil {
		return e
	}
	defer result.Clear()
	link := result.ToIDispatch()
	for name, value := range map[string]string{"TargetPath": target, "WorkingDirectory": dir, "Description": "PeeTwoPee", "IconLocation": target + ",0"} {
		v, e := oleutil.PutProperty(link, name, value)
		if e != nil {
			return e
		}
		v.Clear()
	}
	v, e := oleutil.CallMethod(link, "Save")
	if v != nil {
		v.Clear()
	}
	return e
}
func write(path string, data []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), "p2p-install-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	from, _ := windows.UTF16PtrFromString(tmp)
	to, _ := windows.UTF16PtrFromString(path)
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func Install(app, uninstaller, license, thirdParty, helper, source []byte, desktopLink bool) error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	return InstallTo(dir, app, uninstaller, license, thirdParty, helper, source, desktopLink, true, nil)
}

func InstallTo(dir string, app, uninstaller, license, thirdParty, helper, source []byte, desktopLink, startLink bool, startup *bool) error {
	dir, e := ValidateFolder(dir)
	if e != nil {
		return e
	}
	start, desktop, e := locations()
	if e != nil {
		return e
	}
	startupDir, e := windows.KnownFolderPath(windows.FOLDERID_Startup, 0)
	if e != nil {
		return e
	}
	profile, e := profileDirectory()
	if e != nil {
		return e
	}
	return installAt(destination{dir: dir, start: start, desktop: desktop, key: uninstallKey, profile: profile, startup: startupDir, autoStart: startup, startMenu: &startLink}, map[string][]byte{"PeeTwoPee.exe": app, "uninstall.exe": uninstaller, "LICENSE.txt": license, "THIRD_PARTY_LICENSES.zip": thirdParty, "easytier/etcat-p2p.exe": helper, "EasyTier-corresponding-source.zip": source}, desktopLink, func(text string) bool { return Message(text, true) })
}
func RuntimeInstalled() bool {
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, path := range []string{`Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, `Software\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`} {
			k, e := registry.OpenKey(root, path, registry.QUERY_VALUE)
			if e != nil {
				continue
			}
			v, _, e := k.GetStringValue("pv")
			k.Close()
			if e == nil && v != "" && v != "0.0.0.0" {
				return true
			}
		}
	}
	return false
}
func EnsureRuntime(bootstrap []byte) error {
	if RuntimeInstalled() {
		return nil
	}
	dir, e := os.MkdirTemp("", "p2p-webview-")
	if e != nil {
		return e
	}
	exe := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")
	defer func() { _ = os.Remove(exe); _ = os.Remove(dir) }()
	if e = os.WriteFile(exe, bootstrap, 0600); e != nil {
		return e
	}
	if e = exec.Command(exe, "/silent", "/install").Run(); e != nil {
		return fmt.Errorf("instalação WebView2 falhou; requer internet: %w", e)
	}
	if !RuntimeInstalled() {
		return fmt.Errorf("WebView2 não foi detectado após instalação")
	}
	return nil
}
func Uninstall() error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	current, e := os.Executable()
	if e != nil {
		return e
	}
	if !strings.EqualFold(filepath.Clean(filepath.Dir(current)), filepath.Clean(dir)) {
		return fmt.Errorf("o desinstalador deve ser executado na pasta instalada")
	}
	release, e := holdUpdate(dir, func(text string) bool { return Message(text, true) })
	if e != nil {
		return e
	}
	defer release()
	if _, e = os.Stat(filepath.Join(dir, ".p2p-update.json")); e == nil {
		return fmt.Errorf("atualização interrompida; execute o instalador antes de desinstalar")
	}
	target := filepath.Join(dir, Product+".exe")
	if e = os.Remove(target); e != nil && !os.IsNotExist(e) {
		return fmt.Errorf("feche o aplicativo e tente novamente: %w", e)
	}
	start, desktop, e := locations()
	if e != nil {
		return e
	}
	for _, p := range []string{filepath.Join(dir, "ProjetoP2P.exe"), filepath.Join(start, "ProjetoP2P.lnk"), filepath.Join(desktop, "ProjetoP2P.lnk"), filepath.Join(dir, "easytier", "etcat-p2p.exe"), filepath.Join(dir, "EasyTier-corresponding-source.zip"), filepath.Join(start, Product+".lnk"), filepath.Join(desktop, Product+".lnk"), filepath.Join(dir, "LICENSE.txt"), filepath.Join(dir, "THIRD_PARTY_LICENSES.zip")} {
		if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	if startupDir, err := windows.KnownFolderPath(windows.FOLDERID_Startup, 0); err == nil {
		path := filepath.Join(startupDir, Product+".lnk")
		if target, err := shortcutTarget(path); err == nil && samePath(target, filepath.Join(dir, Product+".exe")) {
			if err = os.Remove(path); err != nil {
				return err
			}
		}
	}
	if e = registry.DeleteKey(registry.CURRENT_USER, uninstallKey); e != nil && e != registry.ErrNotExist {
		return e
	}
	// A separate tiny copy waits for this process before deleting its executable.
	temp, e := os.MkdirTemp("", "p2p-uninstall-")
	if e != nil {
		return e
	}
	helper := filepath.Join(temp, "cleanup.exe")
	b, e := os.ReadFile(current)
	if e != nil {
		return e
	}
	if e = os.WriteFile(helper, b, 0700); e != nil {
		return e
	}
	return exec.Command(helper, "--cleanup", fmt.Sprint(os.Getpid()), dir).Start()
}
func Cleanup(pid uint32) error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	return CleanupAt(pid, dir)
}

// Bind cleanup to the still-running uninstaller, including custom locations.
func CleanupAt(pid uint32, dir string) error {
	dir, e := ValidateFolder(dir)
	if e != nil {
		return e
	}
	h, e := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if e != nil {
		return e
	}
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if e = windows.QueryFullProcessImageName(h, 0, &buf[0], &n); e != nil || !samePath(windows.UTF16ToString(buf[:n]), filepath.Join(dir, "uninstall.exe")) {
		windows.CloseHandle(h)
		return fmt.Errorf("processo de desinstalação não corresponde à pasta")
	}
	if e == nil {
		// The interactive uninstaller may still be showing its completion dialog.
		// Wait for its actual exit rather than timing out while the user reads it.
		_, _ = windows.WaitForSingleObject(h, windows.INFINITE)
		windows.CloseHandle(h)
	}
	p := filepath.Join(dir, "uninstall.exe")
	for i := 0; i < 30; i++ {
		e = os.Remove(p)
		if e == nil || os.IsNotExist(e) {
			_ = os.Remove(dir)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return e
}
func Hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

// Rollback retains the user profile and restores the previous application inventory and registry/shortcuts.
func Rollback() error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	start, desktop, e := locations()
	if e != nil {
		return e
	}
	startupDir, e := windows.KnownFolderPath(windows.FOLDERID_Startup, 0)
	if e != nil {
		return e
	}
	d := destination{dir: dir, start: start, desktop: desktop, key: uninstallKey, startup: startupDir}
	release, e := holdUpdate(dir, func(text string) bool { return Message(text, true) })
	if e != nil {
		return e
	}
	defer release()
	return installtxn.RollbackLast(dir, func(raw json.RawMessage) error { return restoreMetadata(d, raw) })
}

func initializeCOM() error {
	e := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	var code *ole.OleError
	if errors.As(e, &code) && code.Code() == 1 {
		return nil
	}
	return e
}

func profileDirectory() (string, error) {
	if value := os.Getenv("PROJETOP2P_DATA"); value != "" {
		return filepath.Abs(value)
	}
	base, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(base, "ProjetoP2P"), nil
}
