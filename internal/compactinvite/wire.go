// Package compactinvite implements the bounded Protobuf wire envelope for invitations.
// Cryptographic validation remains in the contact/community packages.
package compactinvite

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/encoding/protowire"
	"strings"
)

type Fields map[int][][]byte

func (f Fields) Add(n int, b []byte) { f[n] = append(f[n], b) }
func (f Fields) One(n int) []byte {
	if len(f[n]) == 0 {
		return nil
	}
	return f[n][0]
}
func Number(v uint64) []byte { return binary.AppendUvarint(nil, v) }
func Uint(b []byte) (uint64, error) {
	v, n := binary.Uvarint(b)
	if n != len(b) || n <= 0 {
		return 0, errors.New("invalid number")
	}
	return v, nil
}
func Encode(prefix string, f Fields) string {
	var out []byte
	for n := 1; n <= 13; n++ {
		for _, b := range f[n] {
			if n == 3 || n == 10 || n == 12 {
				v, _ := Uint(b)
				out = protowire.AppendTag(out, protowire.Number(n), protowire.VarintType)
				out = protowire.AppendVarint(out, v)
			} else {
				out = protowire.AppendTag(out, protowire.Number(n), protowire.BytesType)
				out = protowire.AppendBytes(out, b)
			}
		}
	}
	return prefix + base64.RawURLEncoding.EncodeToString(out)
}
func Decode(raw, prefix string, max int) (Fields, error) {
	f := Fields{}
	if len(raw) > max || !strings.HasPrefix(raw, prefix) {
		return nil, errors.New("invalid compact invitation")
	}
	b, e := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(raw, prefix))
	if e != nil {
		return nil, e
	}
	count := 0
	for len(b) > 0 {
		n, t, k := protowire.ConsumeTag(b)
		if k < 0 || n < 1 || n > 13 {
			return nil, errors.New("invalid invitation field")
		}
		b = b[k:]
		var v []byte
		if n == 3 || n == 10 || n == 12 {
			if t != protowire.VarintType {
				return nil, errors.New("invalid numeric wire type")
			}
			var x uint64
			x, k = protowire.ConsumeVarint(b)
			v = Number(x)
		} else {
			if t != protowire.BytesType {
				return nil, errors.New("invalid bytes wire type")
			}
			v, k = protowire.ConsumeBytes(b)
		}
		if k < 0 {
			return nil, errors.New("truncated invitation")
		}
		b = b[k:]
		count++
		if count > 48 || len(v) > 4096 || (len(f[int(n)]) > 0 && n != 5 && n != 6) {
			return nil, errors.New("duplicate or oversized field")
		}
		f.Add(int(n), v)
	}
	return f, nil
}
func Key(raw []byte) ([]byte, string, error) {
	k, e := crypto.UnmarshalEd25519PublicKey(raw)
	if e != nil {
		return nil, "", e
	}
	pub, e := crypto.MarshalPublicKey(k)
	if e != nil {
		return nil, "", e
	}
	id, e := peer.IDFromPublicKey(k)
	return pub, id.String(), e
}
func RawKey(pub []byte) ([]byte, error) {
	k, e := crypto.UnmarshalPublicKey(pub)
	if e != nil {
		return nil, e
	}
	if k.Type() != crypto.Ed25519 {
		return nil, errors.New("Ed25519 required")
	}
	return k.Raw()
}
func Addresses(xs []string, id string) ([][]byte, error) {
	out := [][]byte{}
	for _, s := range xs {
		a, e := ma.NewMultiaddr(s)
		if e != nil {
			return nil, e
		}
		suffix, e := ma.NewMultiaddr("/p2p/" + id)
		if e != nil {
			return nil, e
		}
		if !strings.HasSuffix(s, suffix.String()) {
			return nil, errors.New("address identity mismatch")
		}
		out = append(out, a.Decapsulate(suffix).Bytes())
	}
	return out, nil
}
func Restore(xs [][]byte, id string) ([]string, error) {
	out := []string{}
	suffix, e := ma.NewMultiaddr("/p2p/" + id)
	if e != nil {
		return nil, e
	}
	for _, b := range xs {
		a, e := ma.NewMultiaddrBytes(b)
		if e != nil {
			return nil, e
		}
		out = append(out, a.Encapsulate(suffix).String())
	}
	return out, nil
}
