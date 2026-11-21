//go:build windows && !nosystemaudio

package loopback

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var catalog = struct {
	sync.Mutex
	entries map[int]uint64
}{entries: map[int]uint64{}}

//go:embed capture.bin
var helper []byte

func executable(dir string) (string, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256(helper))
	dir = filepath.Join(dir, "wasapi", hash[:16])
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	p := filepath.Join(dir, "PeeTwoPee-Loopback.exe")
	b, e := os.ReadFile(p)
	if e != nil || sha256.Sum256(b) != sha256.Sum256(helper) {
		if e = os.WriteFile(p, helper, 0600); e != nil {
			return "", e
		}
	}
	return p, nil
}
func command(ctx context.Context, p string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, p, args...)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return c
}
func Sources(ctx context.Context, dir string) ([]Source, error) {
	p, e := executable(dir)
	if e != nil {
		return nil, e
	}
	ctx, stop := context.WithTimeout(ctx, 3*time.Second)
	defer stop()
	b, e := command(ctx, p, "--list").Output()
	if e != nil {
		return nil, e
	}
	sources := []Source{}
	selected := map[int]uint64{}
	for _, line := range strings.Split(string(b), "\n") {
		v := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(v) >= 2 {
			pid, _ := strconv.Atoi(v[0])
			if pid > 0 && !relatedProcess(uint32(pid)) {
				created, h, e := processIdentity(pid)
				if e == nil {
					windows.CloseHandle(h)
					selected[pid] = created
					title := ""
					if len(v) == 3 {
						if decoded, err := base64.StdEncoding.DecodeString(v[2]); err == nil {
							title = string(decoded)
						}
					}
					sources = append(sources, Source{PID: pid, Name: v[1], Title: title})
				}
			}
		}
	}
	catalog.Lock()
	catalog.entries = selected
	catalog.Unlock()
	return sources, nil
}
func Start(parent context.Context, dir, mode string, pid int) (*Session, error) {
	if e := valid(mode, pid); e != nil {
		return nil, e
	}
	var handle windows.Handle
	if mode == "process" {
		if relatedProcess(uint32(pid)) {
			return nil, errors.New("selecione um aplicativo que não contenha nem pertença ao PeeTwoPee")
		}
		created, h, e := processIdentity(pid)
		if e != nil {
			return nil, e
		}
		catalog.Lock()
		expected, ok := catalog.entries[pid]
		catalog.Unlock()
		if !ok || expected != created {
			windows.CloseHandle(h)
			return nil, errors.New("aplicativo alterado ou reiniciado; atualize as fontes e selecione novamente")
		}
		handle = h
	}
	started := false
	defer func() {
		if !started && handle != 0 {
			windows.CloseHandle(handle)
		}
	}()

	if mode == "exclude" {
		pid = os.Getpid()
	}
	p, e := executable(dir)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithCancel(parent)
	c := command(ctx, p, "--capture", mode, strconv.Itoa(pid))
	stdout, e := c.StdoutPipe()
	if e != nil {
		cancel()
		return nil, e
	}
	stderr, e := c.StderrPipe()
	if e != nil {
		cancel()
		return nil, e
	}
	if e = c.Start(); e != nil {
		cancel()
		return nil, e
	}
	started = true
	s := &Session{cancel: cancel, done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 1024), 4096)
		signaled := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "READY ") {
				if !signaled {
					ready <- nil
					signaled = true
				}
			} else if strings.HasPrefix(line, "WASAPI_ERROR ") {
				err := errors.New("Falha WASAPI: " + line)
				s.fail(err)
				if !signaled {
					ready <- err
					signaled = true
				}
			}
		}
		if !signaled {
			ready <- errors.New("WASAPI indisponível; verifique Windows, serviço de áudio e dispositivo de saída")
		}
	}()
	go func() {
		defer func() {
			cancel()
			c.Wait()
			if handle != 0 {
				windows.CloseHandle(handle)
			}
			close(s.done)
		}()
		for {
			b := make([]byte, 1920)
			if _, e := io.ReadFull(stdout, b); e != nil {
				if ctx.Err() == nil {
					s.fail(errors.New("captura WASAPI encerrada"))
				}
				return
			}
			s.push(b)
		}
	}()
	select {
	case e := <-ready:
		if e != nil {
			s.Close()
			return nil, e
		}
		return s, nil
	case <-time.After(7 * time.Second):
		s.Close()
		return nil, errors.New("WASAPI não iniciou dentro do prazo")
	case <-ctx.Done():
		s.Close()
		return nil, ctx.Err()
	}
}

func processIdentity(pid int) (uint64, windows.Handle, error) {
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if e != nil {
		return 0, 0, e
	}
	var created, exit, kernel, user windows.Filetime
	if e = windows.GetProcessTimes(h, &created, &exit, &kernel, &user); e != nil {
		windows.CloseHandle(h)
		return 0, 0, e
	}
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), h, nil
}
func relatedProcess(pid uint32) bool {
	self := uint32(os.Getpid())
	if pid == self {
		return true
	}
	snap, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return true
	}
	defer windows.CloseHandle(snap)
	parents := map[uint32]uint32{}
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	e = windows.Process32First(snap, &entry)
	for e == nil {
		parents[entry.ProcessID] = entry.ParentProcessID
		e = windows.Process32Next(snap, &entry)
	}
	walk := func(start, target uint32) bool {
		for i := 0; i < 64 && start != 0; i++ {
			if start == target {
				return true
			}
			next := parents[start]
			if next == start {
				break
			}
			start = next
		}
		return false
	}
	return walk(pid, self) || walk(self, pid)
}
