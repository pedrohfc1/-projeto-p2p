package installwin

import (
	"encoding/json"
	"fmt"
	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"projetop2p/internal/installtxn"
	"projetop2p/internal/localdb"
	"runtime"
)

var stringFields = []string{"DisplayName", "DisplayVersion", "Publisher", "InstallLocation", "DisplayIcon", "UninstallString", "QuietUninstallString"}
var numberFields = []string{"NoModify", "NoRepair", "EstimatedSize"}

type regValue struct {
	Present bool   `json:"present"`
	Kind    uint32 `json:"kind,omitempty"`
	Text    string `json:"text,omitempty"`
	Number  uint64 `json:"number,omitempty"`
}
type linkValue struct {
	Present bool `json:"present"`

	Bytes []byte `json:"bytes,omitempty"`
}
type metadata struct {
	KeyExisted bool                 `json:"keyExisted"`
	Values     map[string]regValue  `json:"values"`
	Links      map[string]linkValue `json:"links"`
}
type destination struct {
	dir, start, desktop, key, profile, startup string
	autoStart, startMenu                       *bool
}

func (d destination) linkPaths() map[string]string {
	paths := map[string]string{"start": filepath.Join(d.start, Product+".lnk"), "desktop": filepath.Join(d.desktop, Product+".lnk"), "startLegacy": filepath.Join(d.start, "ProjetoP2P.lnk"), "desktopLegacy": filepath.Join(d.desktop, "ProjetoP2P.lnk")}
	if d.startup != "" {
		paths["startup"] = filepath.Join(d.startup, Product+".lnk")
	}
	return paths
}
func shortcutTarget(path string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := initializeCOM(); e != nil {
		return "", e
	}
	defer ole.CoUninitialize()
	object, e := oleutil.CreateObject("WScript.Shell")
	if e != nil {
		return "", e
	}
	defer object.Release()
	shell, e := object.QueryInterface(ole.IID_IDispatch)
	if e != nil {
		return "", e
	}
	defer shell.Release()
	result, e := oleutil.CallMethod(shell, "CreateShortcut", path)
	if e != nil {
		return "", e
	}
	defer result.Clear()
	v, e := oleutil.GetProperty(result.ToIDispatch(), "TargetPath")
	if e != nil {
		return "", e
	}
	defer v.Clear()
	return v.ToString(), nil
}
func snapshot(d destination, desktop bool) (metadata, error) {
	m := metadata{Values: map[string]regValue{}, Links: map[string]linkValue{}}
	key, e := registry.OpenKey(registry.CURRENT_USER, d.key, registry.QUERY_VALUE)
	if e != nil && e != registry.ErrNotExist {
		return m, e
	}
	if e == nil {
		m.KeyExisted = true
		defer key.Close()
		for _, name := range stringFields {
			v, kind, e := key.GetStringValue(name)
			if e != nil && e != registry.ErrNotExist {
				return m, e
			}
			m.Values[name] = regValue{Present: e == nil, Text: v, Kind: kind}
		}
		for _, name := range numberFields {
			v, kind, e := key.GetIntegerValue(name)
			if e != nil && e != registry.ErrNotExist {
				return m, e
			}
			m.Values[name] = regValue{Present: e == nil, Number: v, Kind: kind}
		}
	}
	for id, path := range d.linkPaths() {

		if _, e := installtxn.Safe(filepath.Dir(path), filepath.Base(path)); e != nil {
			return m, e
		}
		b, e := os.ReadFile(path)
		if os.IsNotExist(e) {
			if id == "start" || id == "desktop" || id == "startup" {
				m.Links[id] = linkValue{}
			}
			continue
		}
		if e != nil {
			return m, e
		}
		if len(b) > 64<<10 {
			return m, fmt.Errorf("atalho excessivo")
		}
		target, e := shortcutTarget(path)
		if e != nil {
			return m, e
		}
		if !samePath(target, filepath.Join(d.dir, "PeeTwoPee.exe")) && !samePath(target, filepath.Join(d.dir, "ProjetoP2P.exe")) {
			if id == "startLegacy" || id == "desktopLegacy" {
				continue
			}
			return m, fmt.Errorf("atalho existente aponta para outra instalação: %s", path)
		}
		m.Links[id] = linkValue{true, b}
	}
	return m, nil
}
func restoreMetadata(d destination, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > 3<<20 {
		return fmt.Errorf("metadata size")
	}
	var m metadata
	if json.Unmarshal(raw, &m) != nil || len(m.Values) > len(stringFields)+len(numberFields) || len(m.Links) > 5 {
		return fmt.Errorf("invalid update metadata")
	}
	known := map[string]bool{}
	for _, n := range stringFields {
		known[n] = true
	}
	for _, n := range numberFields {
		known[n] = true
	}
	for n := range m.Values {
		if !known[n] {
			return fmt.Errorf("unknown metadata field")
		}
	}
	for id, v := range m.Links {
		if _, ok := d.linkPaths()[id]; !ok || len(v.Bytes) > 64<<10 {
			return fmt.Errorf("invalid shortcut snapshot")
		}
	}
	key, e := registry.OpenKey(registry.CURRENT_USER, d.key, registry.SET_VALUE|registry.QUERY_VALUE)
	if e == registry.ErrNotExist && m.KeyExisted {
		key, _, e = registry.CreateKey(registry.CURRENT_USER, d.key, registry.SET_VALUE|registry.QUERY_VALUE)
	}
	keyOpen := e == nil
	if e != nil && !(e == registry.ErrNotExist && !m.KeyExisted) {
		return e
	}
	if keyOpen {
		defer func() {
			if keyOpen {
				key.Close()
			}
		}()
		for _, n := range stringFields {
			old := m.Values[n]
			if old.Present {
				if old.Kind == registry.EXPAND_SZ {
					e = key.SetExpandStringValue(n, old.Text)
				} else {
					e = key.SetStringValue(n, old.Text)
				}
			} else {
				e = key.DeleteValue(n)
			}
			if e != nil && e != registry.ErrNotExist {
				return e
			}
		}
		for _, n := range numberFields {
			old := m.Values[n]
			if old.Present {
				if old.Kind == registry.QWORD {
					e = key.SetQWordValue(n, old.Number)
				} else {
					e = key.SetDWordValue(n, uint32(old.Number))
				}
			} else {
				e = key.DeleteValue(n)
			}
			if e != nil && e != registry.ErrNotExist {
				return e
			}
		}
	}
	for id, v := range m.Links {
		path, ok := d.linkPaths()[id]
		if !ok {
			return fmt.Errorf("unknown shortcut")
		}
		if _, e = installtxn.Safe(filepath.Dir(path), filepath.Base(path)); e != nil {
			return e
		}
		if v.Present {
			e = write(path, v.Bytes)
		} else {
			e = os.Remove(path)
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	if !m.KeyExisted && keyOpen {
		names, e := key.ReadValueNames(-1)
		if e != nil && e != registry.ErrNotExist {
			return e
		}
		info, statErr := key.Stat()
		if statErr != nil {
			return statErr
		}
		if len(names) == 0 && info.SubKeyCount == 0 {
			key.Close()
			keyOpen = false
			e = registry.DeleteKey(registry.CURRENT_USER, d.key)
			if e != nil && e != registry.ErrNotExist {
				return e
			}
		}
	}
	return nil
}
func finalizeMetadata(d destination, m metadata, desktop bool, total int) error {
	target := filepath.Join(d.dir, Product+".exe")
	if d.startMenu == nil || *d.startMenu {
		if e := shortcut(filepath.Join(d.start, Product+".lnk"), target, d.dir); e != nil {
			return fmt.Errorf("atalho do menu: %w", e)
		}
	} else if m.Links["start"].Present {
		if e := os.Remove(d.linkPaths()["start"]); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	if desktop {
		if e := shortcut(filepath.Join(d.desktop, Product+".lnk"), target, d.dir); e != nil {
			return e
		}
	}
	if !desktop && m.Links["desktop"].Present {
		if e := os.Remove(d.linkPaths()["desktop"]); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	if d.autoStart != nil && d.startup != "" {
		if *d.autoStart {
			if e := shortcut(d.linkPaths()["startup"], target, d.dir); e != nil {
				return e
			}
		} else if m.Links["startup"].Present {
			if e := os.Remove(d.linkPaths()["startup"]); e != nil && !os.IsNotExist(e) {
				return e
			}
		}
	}
	for _, id := range []string{"startLegacy", "desktopLegacy"} {
		if m.Links[id].Present {
			if e := os.Remove(d.linkPaths()[id]); e != nil && !os.IsNotExist(e) {
				return e
			}
		}
	}
	key, _, e := registry.CreateKey(registry.CURRENT_USER, d.key, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer key.Close()
	fields := map[string]string{"DisplayName": Product, "DisplayVersion": Version, "Publisher": Product, "InstallLocation": d.dir, "DisplayIcon": target, "UninstallString": `"` + filepath.Join(d.dir, "uninstall.exe") + `"`, "QuietUninstallString": `"` + filepath.Join(d.dir, "uninstall.exe") + `" --silent`}
	for n, v := range fields {
		if e = key.SetStringValue(n, v); e != nil {
			return e
		}
	}
	for n, v := range map[string]uint32{"NoModify": 1, "NoRepair": 1, "EstimatedSize": uint32(total / 1024)} {
		if e = key.SetDWordValue(n, v); e != nil {
			return e
		}
	}
	return nil
}
func installAt(d destination, files map[string][]byte, desktop bool, confirm func(string) bool) error {
	return installWithMetadata(d, files, desktop, confirm, snapshot, finalizeMetadata, restoreMetadata)
}
func installWithMetadata(d destination, files map[string][]byte, desktop bool, confirm func(string) bool, capture func(destination, bool) (metadata, error), finish func(destination, metadata, bool, int) error, recoverMeta func(destination, json.RawMessage) error) error {
	release, e := holdUpdate(d.dir, confirm)
	if e != nil {
		return e
	}
	defer release()
	if d.profile != "" {
		if e = localdb.CheckProfileIdle(d.profile); e != nil {
			return fmt.Errorf("outra versão pode estar aberta fora desta instalação; encerre pela bandeja antes de atualizar: %w", e)
		}
	}
	restore := func(raw json.RawMessage) error { return recoverMeta(d, raw) }
	if e = installtxn.Recover(d.dir, restore); e != nil {
		return e
	}
	m, e := capture(d, desktop)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(m)
	if e != nil {
		return e
	}
	total := 0
	for _, data := range files {
		total += len(data)
	}
	return installtxn.Apply(d.dir, Version, files, raw, func() error { return finish(d, m, desktop, total) }, restore)
}
