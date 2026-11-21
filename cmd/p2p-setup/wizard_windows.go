package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"projetop2p/internal/installwin"
)

//go:embed wizard.ps1 logo.png
var assistant embed.FS

type installOptions struct {
	Directory string `json:"directory"`
	Desktop   bool   `json:"desktop"`
	StartMenu bool   `json:"startMenu"`
	Startup   bool   `json:"startup"`
}

func readOptions(path string) (installOptions, error) {
	var o installOptions
	f, e := os.Open(path)
	if e != nil {
		return o, e
	}
	defer f.Close()
	reader := bufio.NewReader(io.LimitReader(f, 16<<10))
	if bom, _ := reader.Peek(3); bytes.Equal(bom, []byte{0xef, 0xbb, 0xbf}) {
		reader.Discard(3)
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&o); e != nil {
		return o, e
	}
	if decoder.Decode(new(any)) != io.EOF {
		return o, fmt.Errorf("opções inválidas")
	}
	o.Directory, e = installwin.ValidateFolder(o.Directory)
	return o, e
}

// The wizard only selects options. The existing transactional installer owns all writes.
func wizard() error {
	dir, e := os.MkdirTemp("", "peetwopee-setup-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(dir) // private temporary directory created above; never the install/profile directory
	for _, name := range []string{"wizard.ps1", "logo.png"} {
		data, _ := assistant.ReadFile(name)
		if e = os.WriteFile(filepath.Join(dir, name), data, 0600); e != nil {
			return e
		}
	}
	folder, e := installwin.Folder()
	if e != nil {
		return e
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.Command(ps, "-NoLogo", "-NoProfile", "-STA", "-ExecutionPolicy", "RemoteSigned", "-File", filepath.Join(dir, "wizard.ps1"), "-Setup", exe, "-DefaultDirectory", folder, "-Version", installwin.Version, "-WorkDirectory", dir)
	// Keep the console hidden while allowing an independent WinForms dialog.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("assistente Windows: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
