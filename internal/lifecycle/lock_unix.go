//go:build !windows

package lifecycle

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func lockFile(f *os.File, exclusive bool) error {
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	return unix.Flock(int(f.Fd()), mode|unix.LOCK_NB)
}
func unlockFile(f *os.File)                             { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }
func watchStop(string) (<-chan struct{}, func(), error) { return nil, func() {}, nil }

func ShowError(message string) { fmt.Fprintln(os.Stderr, message) }
