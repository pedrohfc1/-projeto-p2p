//go:build !windows

package main

import "os"

func protectDirectory(path string) error { return os.Chmod(path, 0700) }

func driverLoaded() bool { return false }
