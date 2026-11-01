package identity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Load creates an exclusive identity file; existing invalid keys are never replaced.
func Load(dir string) (crypto.PrivKey, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := protect(dir, true); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "identity.key")
	if info, e := os.Lstat(path); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("identity file must not be a symbolic link")
	}
	b, err := readIdentity(path)
	if err == nil {
		if err = protect(path, false); err != nil {
			return nil, err
		}
		b, err = unseal(b)
		if err != nil {
			return nil, fmt.Errorf("unlock identity: %w", err)
		}
		key, e := crypto.UnmarshalPrivateKey(b)
		if e == nil && key.Type() != crypto.Ed25519 {
			return nil, errors.New("identity must use Ed25519; existing file was preserved")
		}
		return key, e
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err = crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, err
	}
	b, err = seal(b)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}
	if _, err = f.Write(b); err != nil {
		f.Close()
		return nil, err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	if err = protect(path, false); err != nil {
		return nil, err
	}
	return key, nil
}

func readIdentity(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil {
		return nil, e
	}
	if len(b) > 65536 {
		return nil, errors.New("identity file exceeds 64 KiB; existing file was preserved")
	}
	return b, nil
}
