// Package registry defines integration boundaries, without a blockchain dependency.
package registry

import "context"

type SignedRecord struct {
	Payload   []byte
	Signature []byte
	PublicKey []byte
}
type NodeRegistry interface {
	ResolveNode(context.Context, string) (SignedRecord, error)
}
type DataIntegrityRegistry interface {
	ResolveCommitment(context.Context, string) (SignedRecord, error)
}
type OwnershipRegistry interface {
	ResolveOwner(context.Context, string) (SignedRecord, error)
}
type PermissionRegistry interface {
	ResolvePermissions(context.Context, string) (SignedRecord, error)
}
