package identity

import (
	"crypto/rand"
	"fmt"
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
	b, err := os.ReadFile(path)
	if err == nil {
		if err = protect(path, false); err != nil {
			return nil, err
		}
		b, err = unseal(b)
		if err != nil {
			return nil, fmt.Errorf("unlock identity: %w", err)
		}
		return crypto.UnmarshalPrivateKey(b)
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
