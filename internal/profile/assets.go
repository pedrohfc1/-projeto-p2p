package profile

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
	"projetop2p/internal/installtxn"
)

const MaxImageBytes = 256 * 1024
const MaxImageSide = 256

const maxAssetFiles = 2048
const maxAssetBytes = 128 * 1024 * 1024
const orphanRetention = 14 * 24 * time.Hour

var ErrHashMismatch = errors.New("image hash mismatch")

type Asset struct {
	Hash string `json:"hash"`
	MIME string `json:"mime"`
}

func validMIME(kind string) bool {
	return kind == "image/png" || kind == "image/jpeg" || kind == "image/webp"
}

func imageAsset(data []byte) (Asset, error) {
	if len(data) == 0 || len(data) > MaxImageBytes {
		return Asset{}, errors.New("profile image exceeds 256 KiB or is empty")
	}
	config, kind, err := image.DecodeConfig(bytes.NewReader(data))
	mime := "image/" + kind
	if err != nil || !validMIME(mime) || config.Width <= 0 || config.Width != config.Height || config.Width > MaxImageSide {
		return Asset{}, errors.New("profile image must be a square PNG, JPEG or WebP up to 256×256")
	}
	if kind == "webp" {
		if err := validateWebP(data, config.Width); err != nil {
			return Asset{}, err
		}
	}
	// Decode pixels only after the dimensions have been bounded.
	decoded, actual, err := image.Decode(bytes.NewReader(data))
	if err != nil || kind != actual || decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return Asset{}, errors.New("invalid profile image pixels")
	}
	return Asset{Hash: hashBytes(data), MIME: mime}, nil
}

func readBounded(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errors.New("invalid or oversized local profile file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err == nil && int64(len(data)) > maximum {
		return nil, errors.New("oversized local profile file")
	}
	return data, err
}

func (s *Store) readImageBytes(hash string) ([]byte, error) {
	if !ValidHash(hash) {
		return nil, errors.New("invalid image hash")
	}
	data, err := readBounded(filepath.Join(s.assets, hash+".asset"), MaxImageBytes)
	if err != nil {
		return nil, err
	}
	if hashBytes(data) != hash {
		return nil, ErrHashMismatch
	}
	return data, nil
}

func (s *Store) HasImage(hash string) bool {
	_, err := s.readImageBytes(hash)
	return err == nil
}

func (s *Store) ReadImage(hash string) (Asset, []byte, error) {
	data, err := s.readImageBytes(hash)
	if err != nil {
		return Asset{}, nil, err
	}
	asset, err := imageAsset(data)
	return asset, data, err
}

func (s *Store) PutImage(data []byte) (Asset, error) {
	asset, err := imageAsset(data)
	if err != nil {
		return Asset{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return asset, s.putImageLocked(asset, data)
}

func (s *Store) PutExpectedImage(hash, mime string, data []byte) (Asset, error) {
	if !ValidHash(hash) || len(data) > MaxImageBytes || hashBytes(data) != hash {
		return Asset{}, ErrHashMismatch
	}
	asset, err := imageAsset(data)
	if err != nil {
		return Asset{}, err
	}
	if mime != "" && mime != asset.MIME {
		return Asset{}, errors.New("profile image MIME mismatch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return asset, s.putImageLocked(asset, data)
}

func (s *Store) putImageLocked(asset Asset, data []byte) error {
	if cached, _, err := s.ReadImage(asset.Hash); err == nil && cached.MIME == asset.MIME {
		return nil
	}
	if err := s.trimLocked(time.Now(), asset.Hash, int64(len(data))); err != nil {
		return err
	}
	return installtxn.Atomic(filepath.Join(s.assets, asset.Hash+".asset"), data)
}

// SetArtworkReferences supplies the complete signed community/channel icon set.
// Until that set is known, the store refuses to prune potential room artwork.
func (s *Store) SetArtworkReferences(hashes []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artwork = make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		if ValidHash(hash) {
			s.artwork[hash] = true
		}
	}
	s.artworkKnown = true
}

func (s *Store) trimLocked(now time.Time, incoming string, size int64) error {
	entries, err := os.ReadDir(s.assets)
	if err != nil {
		return err
	}
	pinned := make(map[string]bool, len(s.peers)+len(s.artwork)+1)
	pinned[s.local.AvatarHash] = true
	for _, metadata := range s.peers {
		pinned[metadata.AvatarHash] = true
	}
	for hash := range s.artwork {
		pinned[hash] = true
	}
	type orphan struct {
		name string
		info os.FileInfo
	}
	var orphans []orphan
	count, total := 1, size
	for _, entry := range entries {
		hash, ok := strings.CutSuffix(entry.Name(), ".asset")
		if !ok || !ValidHash(hash) || hash == incoming {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		count++
		total += info.Size()
		if s.artworkKnown && !pinned[hash] && now.Sub(info.ModTime()) >= orphanRetention {
			orphans = append(orphans, orphan{name: entry.Name(), info: info})
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].info.ModTime().Before(orphans[j].info.ModTime()) })
	for _, entry := range orphans {
		if os.Remove(filepath.Join(s.assets, entry.name)) == nil {
			count--
			total -= entry.info.Size()
		}
	}
	if count > maxAssetFiles || total > maxAssetBytes {
		return errors.New("profile image cache limit reached")
	}
	return nil
}
