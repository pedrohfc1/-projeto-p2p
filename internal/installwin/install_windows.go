// Package installwin implements a per-user installer without changing VPN,
// firewall, machine-wide permissions or the user's persistent application data.
package installwin

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const Product = "ProjetoP2P"
const Version = "0.5.0"
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ProjetoP2PProjetoP2P`

func Folder() (string, error) {
	base, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(base, "Programs", Product), nil
}
func Message(text string, question bool) bool {
	flags := uintptr(0x40)
	if question {
		flags = 0x24
	}
	t, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString("Projeto P2P â€” InstalaÃ§Ã£o 0.4")
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
	if e := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); e != nil {
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
	for name, value := range map[string]string{"TargetPath": target, "WorkingDirectory": dir, "Description": "Projeto P2P", "IconLocation": target + ",0"} {
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
func Install(app, uninstaller, license, thirdParty []byte, desktopLink bool) error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	start, desktop, e := locations()
	if e != nil {
		return e
	}
	target := filepath.Join(dir, Product+".exe")
	if e = write(target, app); e != nil {
		return fmt.Errorf("feche o aplicativo antes de instalar/atualizar: %w", e)
	}
	if e = write(filepath.Join(dir, "uninstall.exe"), uninstaller); e != nil {
		return e
	}
	for name, data := range map[string][]byte{"LICENSE.txt": license, "THIRD_PARTY_LICENSES.zip": thirdParty} {
		if e = write(filepath.Join(dir, name), data); e != nil {
			return e
		}
	}
	if e = shortcut(filepath.Join(start, Product+".lnk"), target, dir); e != nil {
		return e
	}
	if desktopLink {
		if e = shortcut(filepath.Join(desktop, Product+".lnk"), target, dir); e != nil {
			return e
		}
	}
	key, _, e := registry.CreateKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer key.Close()
	fields := map[string]string{"DisplayName": "Projeto P2P", "DisplayVersion": Version, "Publisher": "Projeto P2P", "InstallLocation": dir, "DisplayIcon": target, "UninstallString": `"` + filepath.Join(dir, "uninstall.exe") + `"`, "QuietUninstallString": `"` + filepath.Join(dir, "uninstall.exe") + `" --silent`}
	for k, v := range fields {
		if e = key.SetStringValue(k, v); e != nil {
			return e
		}
	}
	if e = key.SetDWordValue("NoModify", 1); e != nil {
		return e
	}
	if e = key.SetDWordValue("NoRepair", 1); e != nil {
		return e
	}
	return key.SetDWordValue("EstimatedSize", uint32((len(app)+len(uninstaller)+len(license)+len(thirdParty))/1024))
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
		return fmt.Errorf("instalaÃ§Ã£o WebView2 falhou; requer internet: %w", e)
	}
	if !RuntimeInstalled() {
		return fmt.Errorf("WebView2 nÃ£o foi detectado apÃ³s instalaÃ§Ã£o")
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
	target := filepath.Join(dir, Product+".exe")
	if e = os.Remove(target); e != nil && !os.IsNotExist(e) {
		return fmt.Errorf("feche o aplicativo e tente novamente: %w", e)
	}
	start, desktop, e := locations()
	if e != nil {
		return e
	}
	for _, p := range []string{filepath.Join(start, Product+".lnk"), filepath.Join(desktop, Product+".lnk"), filepath.Join(dir, "LICENSE.txt"), filepath.Join(dir, "THIRD_PARTY_LICENSES.zip")} {
		if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
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
	return exec.Command(helper, "--cleanup", fmt.Sprint(os.Getpid())).Start()
}
func Cleanup(pid uint32) error {
	dir, e := Folder()
	if e != nil {
		return e
	}
	h, e := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
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
