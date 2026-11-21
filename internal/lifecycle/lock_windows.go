//go:build windows

package lifecycle

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"
)

func lockFile(f *os.File, exclusive bool) error {
	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &windows.Overlapped{})
}
func unlockFile(f *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
}
func eventName(install string) *uint16 {
	p, _ := filepath.Abs(install)
	if resolved, e := filepath.EvalSymlinks(p); e == nil {
		p = resolved
	}
	h := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(p))))
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf(`Local\PeeTwoPee-Stop-%x`, h[:16]))
	return name
}
func watchStop(install string) (<-chan struct{}, func(), error) {
	h, e := windows.CreateEvent(nil, 1, 0, eventName(install))
	if e != nil && e != windows.ERROR_ALREADY_EXISTS {
		return nil, nil, e
	}
	stop, done, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(exited)
		for {
			select {
			case <-done:
				return
			default:
			}
			r, e := windows.WaitForSingleObject(h, 250)
			if e != nil {
				return
			}
			if r == windows.WAIT_OBJECT_0 {
				close(stop)
				return
			}
		}
	}()
	var once sync.Once
	return stop, func() { once.Do(func() { close(done); <-exited; windows.CloseHandle(h) }) }, nil
}

// RequestStop is called only after interactive update consent. No force or name-based process kill.
func RequestStop(install string) (func(), error) {
	h, e := windows.CreateEvent(nil, 1, 0, eventName(install))
	if e != nil && e != windows.ERROR_ALREADY_EXISTS {
		return nil, e
	}
	if e = windows.SetEvent(h); e != nil {
		windows.CloseHandle(h)
		return nil, e
	}
	return func() { _ = windows.ResetEvent(h); windows.CloseHandle(h) }, nil
}
func WaitExclusive(path string, timeout time.Duration) (*Lock, error) {
	until := time.Now().Add(timeout)
	for {
		l, e := Exclusive(path)
		if e == nil {
			return l, nil
		}
		if time.Now().After(until) {
			return nil, e
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func ShowError(message string) {
	text, _ := windows.UTF16PtrFromString(message)
	title, _ := windows.UTF16PtrFromString("PeeTwoPee — inicialização")
	windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x30)
}
