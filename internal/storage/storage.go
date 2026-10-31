// Package storage reserves a boundary for encrypted content. No replication is implemented.
package storage

import "context"

// Content IDs must commit to ciphertext; keys must travel through a separate authorized channel.
type EncryptedStore interface {
	PutCiphertext(context.Context, []byte) (string, error)
	GetCiphertext(context.Context, string) ([]byte, error)
}
