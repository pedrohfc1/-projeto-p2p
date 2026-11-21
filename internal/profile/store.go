package profile

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
	"projetop2p/internal/installtxn"
)

const maxProfiles = 1024
const maxMetadataBytes = 2048

type Store struct {
	mu           sync.RWMutex
	key          crypto.PrivKey
	metadata     string
	assets       string
	local        SignedMetadata
	peers        map[string]Metadata
	artwork      map[string]bool
	artworkKnown bool
}

func Open(dir string, key crypto.PrivKey, legacyName string) (*Store, error) {
	if key == nil || key.Type() != crypto.Ed25519 {
		return nil, errors.New("profile requires an Ed25519 identity")
	}
	m, err := newMetadata(key, legacyName)
	if err != nil {
		return nil, err
	}
	s := &Store{
		key: key, metadata: filepath.Join(dir, "profile-metadata"), assets: filepath.Join(dir, "profile-assets"),
		peers: make(map[string]Metadata),
	}
	for _, path := range []string{s.metadata, s.assets} {
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("invalid local profile directory")
		}
	}
	data, err := readBounded(s.selfPath(), maxMetadataBytes)
	if err == nil {
		if err := json.Unmarshal(data, &s.local); err != nil {
			return nil, err
		}
		if err := Verify(m.PeerID, s.local); err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		s.local, err = sign(key, m)
		if err != nil {
			return nil, err
		}
		if err := writeMetadata(s.selfPath(), s.local); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	if err := s.loadPeers(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) selfPath() string {
	return filepath.Join(s.metadata, "self.json")
}

func (s *Store) peerPath(id string) string {
	return filepath.Join(s.metadata, hashBytes([]byte(id))+".json")
}

func (s *Store) loadPeers() error {
	entries, err := os.ReadDir(s.metadata)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		hash, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok || !ValidHash(hash) || len(s.peers) >= maxProfiles {
			continue
		}
		raw, err := readBounded(filepath.Join(s.metadata, entry.Name()), maxMetadataBytes)
		if err != nil {
			continue
		}
		var signed SignedMetadata
		if json.Unmarshal(raw, &signed) != nil || hashBytes([]byte(signed.PeerID)) != hash || Verify(signed.PeerID, signed) != nil {
			continue
		}
		if signed.PeerID != s.local.PeerID {
			s.peers[signed.PeerID] = signed.Metadata
		}
	}
	return nil
}

func writeMetadata(path string, signed SignedMetadata) error {
	data, err := json.Marshal(signed)
	if err != nil {
		return err
	}
	return installtxn.Atomic(path, data)
}

func (s *Store) Local() Metadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.local.Metadata
}

func (s *Store) Signed() SignedMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	signed := s.local
	signed.Signature = append([]byte(nil), signed.Signature...)
	return signed
}

func (s *Store) Get(id string) (Metadata, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id == s.local.PeerID {
		return s.local.Metadata, true
	}
	m, ok := s.peers[id]
	return m, ok
}

// Update publishes one atomic local revision. An empty hash means remove avatar.
func (s *Store) Update(name, font, avatarHash string) (Metadata, error) {
	name = NormalizeName(name)
	if !validateName(name) {
		return Metadata{}, errors.New("use de 1 a 32 caracteres no nome de exibição")
	}
	if font == "" {
		font = "inter"
	}
	if !validFont(font) {
		return Metadata{}, errors.New("identificador de fonte inválido")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.local.Metadata
	if old.DisplayName == name && old.DisplayFontID == font && old.AvatarHash == avatarHash {
		return old, nil
	}
	mime := ""
	if avatarHash != "" {
		asset, _, err := s.ReadImage(avatarHash)
		if err != nil {
			return Metadata{}, err
		}
		mime = asset.MIME
	}
	next, err := nextRevision(old)
	if err != nil {
		return Metadata{}, err
	}
	next.DisplayName, next.DisplayFontID, next.AvatarHash, next.AvatarMIME = name, font, avatarHash, mime
	signed, err := sign(s.key, next)
	if err != nil {
		return Metadata{}, err
	}
	if err := writeMetadata(s.selfPath(), signed); err != nil {
		return Metadata{}, err
	}
	s.local = signed
	slog.Info("profile_updated", "revision", next.Revision)
	if old.AvatarHash != next.AvatarHash {
		slog.Info("avatar_changed", "revision", next.Revision)
	}
	return next, nil
}

func (s *Store) CanReceive(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, known := s.peers[id]
	return id != s.local.PeerID && (known || len(s.peers) < maxProfiles)
}

// Accept publishes remote metadata to readers after the corresponding bytes
// are available and validated. A failed fetch leaves the previous revision live.
func (s *Store) Accept(expected string, signed SignedMetadata) error {
	if err := Verify(expected, signed); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if expected == s.local.PeerID {
		return errors.New("cannot replace the local profile over the network")
	}
	prior, known := s.peers[expected]
	if known && signed.Revision < prior.Revision {
		return errors.New("stale profile revision")
	}
	if known && signed.Revision == prior.Revision {
		if signed.Metadata != prior {
			return errors.New("conflicting profile revision")
		}
		return nil
	}
	if !known && len(s.peers) >= maxProfiles {
		return errors.New("peer profile retention limit reached")
	}
	if signed.AvatarHash != "" {
		asset, _, err := s.ReadImage(signed.AvatarHash)
		if err != nil {
			return err
		}
		if asset.MIME != signed.AvatarMIME {
			return errors.New("profile image MIME mismatch")
		}
	}
	if err := writeMetadata(s.peerPath(expected), signed); err != nil {
		return err
	}
	s.peers[expected] = signed.Metadata
	slog.Info("profile_updated", "revision", signed.Revision)
	if prior.AvatarHash != signed.AvatarHash {
		slog.Info("avatar_changed", "revision", signed.Revision)
	}
	return nil
}
