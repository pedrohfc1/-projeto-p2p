package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const version = 1
const signatureDomain = "PeeTwoPee profile v1\x00"

type Metadata struct {
	Version       uint8  `json:"version"`
	PeerID        string `json:"peerId"`
	DisplayName   string `json:"displayName"`
	DisplayFontID string `json:"displayFontId"`
	AvatarHash    string `json:"avatarHash"`
	AvatarMIME    string `json:"avatarMime"`
	Revision      uint64 `json:"revision"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type SignedMetadata struct {
	Metadata
	Signature []byte `json:"signature"`
}

func NormalizeName(name string) string {
	// Keep contextual joiners and normal Unicode. Drop control characters and
	// explicit bidi overrides which can mask the identity in call controls.
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069 {
			return -1
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, name))
}

func validateName(name string) bool {
	return utf8.ValidString(name) && len(name) > 0 && utf8.RuneCountInString(name) <= 32 && NormalizeName(name) == name
}

func validFont(font string) bool {
	if len(font) > 32 {
		return false
	}
	for i, r := range font {
		if r >= 'a' && r <= 'z' || i > 0 && (r == '-' || r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func newMetadata(key crypto.PrivKey, name string) (Metadata, error) {
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return Metadata{}, err
	}
	name = NormalizeName(name)
	if !validateName(name) {
		name = strings.TrimSpace(string([]rune(name)[:min(utf8.RuneCountInString(name), 32)]))
	}
	if !validateName(name) {
		name = "Meu perfil"
	}
	return Metadata{Version: version, PeerID: id.String(), DisplayName: name, DisplayFontID: "inter", Revision: 1, UpdatedAt: time.Now().UnixMilli()}, nil
}

func (m Metadata) payload() ([]byte, error) {
	raw, err := json.Marshal(m)
	return append([]byte(signatureDomain), raw...), err
}

func sign(key crypto.PrivKey, m Metadata) (SignedMetadata, error) {
	body, err := m.payload()
	if err != nil {
		return SignedMetadata{}, err
	}
	signature, err := key.Sign(body)
	return SignedMetadata{Metadata: m, Signature: signature}, err
}

func Verify(expected string, signed SignedMetadata) error {
	m := signed.Metadata
	if m.Version != version || m.PeerID != expected || m.Revision == 0 || m.Revision > 1<<53-1 || m.UpdatedAt <= 0 || !validateName(m.DisplayName) || !validFont(m.DisplayFontID) {
		return errors.New("invalid signed profile metadata")
	}
	if m.AvatarHash == "" && m.AvatarMIME != "" || m.AvatarHash != "" && (!ValidHash(m.AvatarHash) || !validMIME(m.AvatarMIME)) {
		return errors.New("invalid profile asset reference")
	}
	id, err := peer.Decode(expected)
	if err != nil {
		return errors.New("invalid profile peer ID")
	}
	pub, err := id.ExtractPublicKey()
	if err != nil || pub.Type() != crypto.Ed25519 || len(signed.Signature) != 64 {
		return errors.New("profile requires the authenticated Ed25519 identity")
	}
	body, err := m.payload()
	if err != nil {
		return err
	}
	ok, err := pub.Verify(body, signed.Signature)
	if err != nil || !ok {
		return errors.New("invalid profile signature")
	}
	return nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ValidHash(hash string) bool {
	if len(hash) != sha256.Size*2 {
		return false
	}
	for _, r := range hash {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func nextRevision(m Metadata) (Metadata, error) {
	if m.Revision >= 1<<53-1 || m.UpdatedAt == math.MaxInt64 {
		return m, errors.New("profile revision exhausted")
	}
	m.Revision++
	m.UpdatedAt = max(m.UpdatedAt+1, time.Now().UnixMilli())
	return m, nil
}
