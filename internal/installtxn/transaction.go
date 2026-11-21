// Package installtxn updates only an explicit installation inventory. Profiles are never an inventory entry.
package installtxn

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ManifestName = "install-state.json"
const JournalName = ".p2p-update.json"

var Names = []string{"PeeTwoPee.exe", "ProjetoP2P.exe", "uninstall.exe", "easytier/etcat-p2p.exe", "EasyTier-corresponding-source.zip", "LICENSE.txt", "THIRD_PARTY_LICENSES.zip", ManifestName}

type Manifest struct {
	Product string            `json:"product"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}
type Previous struct {
	Exists bool   `json:"exists"`
	Hash   string `json:"hash,omitempty"`
}
type Journal struct {
	Checksum string              `json:"checksum,omitempty"`
	Product  string              `json:"product"`
	Backup   string              `json:"backup"`
	Previous map[string]Previous `json:"previous"`
	Metadata json.RawMessage     `json:"metadata,omitempty"`
}

func Hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func allowed(n string) bool {
	for _, name := range Names {
		if name == n {
			return true
		}
	}
	return false
}
func Safe(root, name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) || strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return "", errors.New("invalid installation path")
	}
	base, e := filepath.Abs(root)
	if e != nil {
		return "", e
	}
	target := filepath.Join(base, filepath.FromSlash(name))
	rel, e := filepath.Rel(base, target)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("installation path escape")
	}
	for current := target; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("reparse/symlink installation path refused")
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return target, nil
}
func Atomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".p2p-write-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	if closeErr := f.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func copyFile(from, to string) error {
	f, e := os.Open(from)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = os.MkdirAll(filepath.Dir(to), 0700); e != nil {
		return e
	}
	out, e := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, f)
	if e == nil {
		e = out.Sync()
	}
	if ce := out.Close(); e == nil {
		e = ce
	}
	return e
}
func fileHash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func Read(root string) (Manifest, error) {
	var m Manifest
	p, e := Safe(root, ManifestName)
	if e != nil {
		return m, e
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return m, e
	}
	if len(b) > 32768 {
		return m, errors.New("manifest size")
	}
	e = json.Unmarshal(b, &m)
	if e != nil || m.Product != "PeeTwoPee" || len(m.Files) != len(Names)-2 {
		return m, errors.New("invalid install manifest")
	}
	for n, h := range m.Files {
		if !allowed(n) || n == ManifestName || n == "ProjetoP2P.exe" || len(h) != 64 {
			return m, errors.New("invalid manifest inventory")
		}
	}
	return m, nil
}
func Verify(root string) error {
	if _, e := os.Stat(filepath.Join(root, JournalName)); e == nil {
		return errors.New("atualização interrompida; execute o instalador para recuperar")
	} else if !os.IsNotExist(e) {
		return e
	}
	m, e := Read(root)
	if e != nil {
		return e
	}
	for n, want := range m.Files {
		p, e := Safe(root, n)
		if e != nil {
			return e
		}
		got, e := fileHash(p)
		if e != nil || got != want {
			return fmt.Errorf("integridade inválida: %s; execute o instalador", n)
		}
	}
	return nil
}
func readJournal(root string) (Journal, error) {
	var j Journal
	p, e := Safe(root, JournalName)
	if e != nil {
		return j, e
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return j, e
	}
	if len(b) > 1<<20 {
		return j, errors.New("journal size")
	}
	if json.Unmarshal(b, &j) != nil || j.Product != "PeeTwoPee" || len(j.Backup) != 32 || len(j.Previous) != len(Names) {
		return j, errors.New("invalid update journal")
	}
	if _, e = hex.DecodeString(j.Backup); e != nil {
		return j, e
	}
	for n, v := range j.Previous {
		if !allowed(n) || v.Exists && len(v.Hash) != 64 {
			return j, errors.New("invalid rollback inventory")
		}
	}
	if !journalValid(j) {
		return j, errors.New("update journal integrity failed")
	}
	return j, nil
}
func Recover(root string, restoreMetadata func(json.RawMessage) error) error {
	j, e := readJournal(root)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	// Verify the complete backup before touching any installed file.
	for n, old := range j.Previous {
		if !old.Exists {
			continue
		}
		b, e := Safe(root, ".p2p-backups/"+j.Backup+"/"+n)
		if e != nil {
			return e
		}
		h, e := fileHash(b)
		if e != nil || h != old.Hash {
			return fmt.Errorf("backup inválido: %s", n)
		}
	}
	for _, n := range Names {
		p, e := Safe(root, n)
		if e != nil {
			return e
		}
		old := j.Previous[n]
		if old.Exists {
			if h, e := fileHash(p); e == nil && h == old.Hash {
				continue
			}
			b, e := os.ReadFile(filepath.Join(root, ".p2p-backups", j.Backup, filepath.FromSlash(n)))
			if e != nil {
				return e
			}
			if e = Atomic(p, b); e != nil {
				return fmt.Errorf("rollback %s: %w", n, e)
			}
		} else if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	if restoreMetadata != nil {
		if e = restoreMetadata(j.Metadata); e != nil {
			return e
		}
	}
	return os.Remove(filepath.Join(root, JournalName))
}

// Apply keeps a verified previous inventory for recovery; finalize must be reversible through metadata.
// The caller holds exclusive installation/startup gates and stops owned processes before calling.
func Apply(root, version string, files map[string][]byte, metadata json.RawMessage, finalize func() error, restore func(json.RawMessage) error) error {
	if len(metadata) > 512<<10 || len(metadata) > 0 && !json.Valid(metadata) {
		return errors.New("invalid update metadata")
	}
	if e := Recover(root, restore); e != nil {
		return e
	}
	for n := range files {
		if !allowed(n) || n == ManifestName || n == "ProjetoP2P.exe" {
			return errors.New("unowned update file")
		}
	}
	if len(files) != len(Names)-2 {
		return errors.New("incomplete installation payload")
	}
	m := Manifest{Product: "PeeTwoPee", Version: version, Files: map[string]string{}}
	for n, b := range files {
		m.Files[n] = Hash(b)
	}
	manifest, _ := json.MarshalIndent(m, "", "  ")
	payload := map[string][]byte{}
	for n, b := range files {
		payload[n] = b
	}
	payload[ManifestName] = manifest
	nonce := make([]byte, 16)
	if _, e := rand.Read(nonce); e != nil {
		return e
	}
	j := Journal{Product: "PeeTwoPee", Backup: hex.EncodeToString(nonce), Previous: map[string]Previous{}, Metadata: metadata}
	backupRoot, e := Safe(root, ".p2p-backups/"+j.Backup)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(backupRoot), 0700); e != nil {
		return e
	}
	if e = os.Mkdir(backupRoot, 0700); e != nil {
		return e
	}
	activated := false
	defer func() {
		if !activated {
			cleanBackup(root, j.Backup)
		}
	}()
	for _, n := range Names {
		target, e := Safe(root, n)
		if e != nil {
			return e
		}
		hash, e := fileHash(target)
		if e == nil {
			j.Previous[n] = Previous{true, hash}
			backup, e := Safe(root, ".p2p-backups/"+j.Backup+"/"+n)
			if e != nil {
				return e
			}
			if e = copyFile(target, backup); e != nil {
				return e
			}
			if got, e := fileHash(backup); e != nil || got != hash {
				return errors.New("backup verification failed")
			}
		} else if os.IsNotExist(e) {
			j.Previous[n] = Previous{}
		} else {
			return e
		}
	}
	j.Checksum = journalHash(j)
	raw, e := json.Marshal(j)
	if e != nil {
		return e
	}
	receipt, e := Safe(root, ".p2p-backups/"+j.Backup+"/rollback.json")
	if e != nil {
		return e
	}
	if e = Atomic(receipt, raw); e != nil {
		return e
	}
	journal, e := Safe(root, JournalName)
	if e != nil {
		return e
	}
	if e = Atomic(journal, raw); e != nil {
		return e
	}
	activated = true
	rollback := func(cause error) error {
		if e := Recover(root, restore); e != nil {
			return fmt.Errorf("%v; recuperação pendente: %w", cause, e)
		}
		return fmt.Errorf("atualização revertida: %w", cause)
	}
	// Preflight/stage all new bytes before the first replacement, then deterministic replacement.
	stage := filepath.Join(root, ".p2p-backups", j.Backup, "new")
	names := make([]string, 0, len(payload))
	for n := range payload {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		path, e := Safe(root, ".p2p-backups/"+j.Backup+"/new/"+n)
		if e != nil {
			return rollback(e)
		}
		if e = Atomic(path, payload[n]); e != nil {
			return rollback(e)
		}
		if h, e := fileHash(path); e != nil || h != Hash(payload[n]) {
			return rollback(errors.New("stage verification failed"))
		}
	}
	for _, n := range names {
		if n == ManifestName {
			continue
		}
		target, _ := Safe(root, n)
		if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			return rollback(e)
		}
		if e = os.Rename(filepath.Join(stage, filepath.FromSlash(n)), target); e != nil {
			return rollback(fmt.Errorf("arquivo bloqueado %s: %w", n, e))
		}
	}
	legacy, _ := Safe(root, "ProjetoP2P.exe")
	if e = os.Remove(legacy); e != nil && !os.IsNotExist(e) {
		return rollback(e)
	}
	for n, want := range m.Files {
		path, _ := Safe(root, n)
		if h, e := fileHash(path); e != nil || h != want {
			return rollback(fmt.Errorf("installed integrity failed: %s", n))
		}
	}
	if e = os.Rename(filepath.Join(stage, ManifestName), filepath.Join(root, ManifestName)); e != nil {
		return rollback(e)
	}
	if finalize != nil {
		if e = finalize(); e != nil {
			return rollback(e)
		}
	}
	// Last commit point. If interrupted earlier the next installer restores files and metadata.
	if e = os.Remove(journal); e != nil {
		return rollback(e)
	}
	if e = Verify(root); e != nil {
		return e
	}
	Prune(root, 2)
	return nil
}

// RollbackLast is explicitly requested by the user, under the same installation gates.
func RollbackLast(root string, restore func(json.RawMessage) error) error {
	if e := Recover(root, restore); e != nil {
		return e
	}
	entries, e := os.ReadDir(filepath.Join(root, ".p2p-backups"))
	if e != nil {
		return e
	}
	type candidate struct {
		name string
		when int64
	}
	var items []candidate
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) != 32 {
			continue
		}
		if _, e := hex.DecodeString(entry.Name()); e != nil {
			continue
		}
		path, e := Safe(root, ".p2p-backups/"+entry.Name()+"/rollback.json")
		if e != nil {
			return e
		}
		info, e := os.Stat(path)
		if e == nil {
			items = append(items, candidate{entry.Name(), info.ModTime().UnixNano()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].when > items[j].when })
	if len(items) == 0 {
		return errors.New("nenhum backup disponível")
	}
	receipt, _ := Safe(root, ".p2p-backups/"+items[0].name+"/rollback.json")
	raw, e := os.ReadFile(receipt)
	if e != nil {
		return e
	}
	var j Journal
	if json.Unmarshal(raw, &j) != nil || j.Backup != items[0].name || !journalValid(j) {
		return errors.New("invalid rollback receipt")
	}
	if !j.Previous["PeeTwoPee.exe"].Exists && !j.Previous["ProjetoP2P.exe"].Exists {
		return errors.New("não existe instalação anterior neste backup; use o desinstalador")
	}
	if j.Product != "PeeTwoPee" || len(j.Previous) != len(Names) {
		return errors.New("invalid rollback inventory")
	}
	for n, old := range j.Previous {
		if !allowed(n) {
			return errors.New("unknown rollback file")
		}
		if old.Exists {
			path, e := Safe(root, ".p2p-backups/"+j.Backup+"/"+n)
			if e != nil {
				return e
			}
			if h, e := fileHash(path); e != nil || h != old.Hash {
				return errors.New("backup corrompido; instalação atual preservada")
			}
		}
	}
	if e = Atomic(filepath.Join(root, JournalName), raw); e != nil {
		return e
	}
	return Recover(root, restore)
}

// Delete only inventoried bytes in older installer-owned backups, never recursive user directories.
func Prune(root string, keep int) {
	entries, e := os.ReadDir(filepath.Join(root, ".p2p-backups"))
	if e != nil {
		return
	}
	type item struct {
		name string
		when int64
	}
	var items []item
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) != 32 {
			continue
		}
		if _, e := hex.DecodeString(entry.Name()); e != nil {
			continue
		}
		path, e := Safe(root, ".p2p-backups/"+entry.Name()+"/rollback.json")
		if e != nil {
			continue
		}
		raw, e := os.ReadFile(path)
		if e != nil || len(raw) > 1<<20 {
			continue
		}
		var j Journal
		if json.Unmarshal(raw, &j) != nil || j.Product != "PeeTwoPee" || j.Backup != entry.Name() || !journalValid(j) {
			continue
		}
		info, e := os.Stat(path)
		if e == nil {
			items = append(items, item{entry.Name(), info.ModTime().UnixNano()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].when > items[j].when })
	for i := keep; i < len(items); i++ {
		cleanBackup(root, items[i].name)
	}
}

func VerifyExecutable(executable, version string) (string, error) {
	root := filepath.Dir(executable)
	m, e := Read(root)
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	h, he := fileHash(executable)
	if he != nil {
		return "", he
	}
	if os.IsNotExist(e) {
		return h, nil
	}
	if m.Version != version {
		return "", errors.New("versão do executável não corresponde à instalação; use o atalho atualizado")
	}
	if h != m.Files["PeeTwoPee.exe"] {
		return "", errors.New("executável diferente do instalado; use o atalho atualizado ou reinstale")
	}
	return h, nil
}

// VerifyRuntime gates only executable components at startup; all distributed files
// are verified at install commit. Missing documentation must not disable a call.
func VerifyRuntime(root string) error {
	if _, e := os.Stat(filepath.Join(root, JournalName)); e == nil {
		return errors.New("atualização interrompida; execute o instalador")
	} else if !os.IsNotExist(e) {
		return e
	}
	m, e := Read(root)
	if e != nil {
		return e
	}
	for _, n := range []string{"PeeTwoPee.exe", "easytier/etcat-p2p.exe"} {
		p, e := Safe(root, n)
		if e != nil {
			return e
		}
		if h, e := fileHash(p); e != nil || h != m.Files[n] {
			return fmt.Errorf("componente executável inválido: %s; reinstale", n)
		}
	}
	return nil
}

func cleanBackup(root, id string) {
	if len(id) != 32 {
		return
	}
	if _, e := hex.DecodeString(id); e != nil {
		return
	}
	prefix := ".p2p-backups/" + id + "/"
	for _, name := range Names {
		for _, rel := range []string{name, "new/" + name} {
			if path, e := Safe(root, prefix+rel); e == nil {
				_ = os.Remove(path)
			}
		}
	}
	for _, name := range []string{"easytier", "new/easytier", "new", "rollback.json", ""} {
		if path, e := Safe(root, prefix+name); e == nil {
			_ = os.Remove(path)
		}
	}
}

func journalHash(j Journal) string {
	j.Checksum = ""
	raw, e := json.Marshal(j)
	if e != nil {
		return ""
	}
	return Hash(raw)
}
func journalValid(j Journal) bool { return len(j.Checksum) == 64 && j.Checksum == journalHash(j) }
