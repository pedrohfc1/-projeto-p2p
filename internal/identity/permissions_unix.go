//go:build !windows

package identity

import "os"

func seal(b []byte) ([]byte, error)   { return b, nil }
func unseal(b []byte) ([]byte, error) { return b, nil }

func protect(path string, directory bool) error {
	if directory {
		return os.Chmod(path, 0700)
	}
	return os.Chmod(path, 0600)
}
