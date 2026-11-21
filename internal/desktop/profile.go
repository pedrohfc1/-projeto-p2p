package desktop

import (
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"unicode/utf8"

	"projetop2p/internal/community"
	"projetop2p/internal/profile"
)

type ImageAsset struct {
	MIME string `json:"mime"`
	Data string `json:"data"`
}

type ProfileChange struct {
	DisplayName   string      `json:"displayName"`
	DisplayFontID string      `json:"displayFontId"`
	Avatar        *ImageAsset `json:"avatar,omitempty"`
	RemoveAvatar  bool        `json:"removeAvatar,omitempty"`
}

// Older invitations and community member records carry at most 80 UTF-8 bytes.
func legacyProfileName(name string) string {
	for len(name) > 80 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return strings.TrimSpace(name)
}

func (image *ImageAsset) decode() ([]byte, error) {
	if image == nil {
		return nil, nil
	}
	if len(image.Data) > base64.StdEncoding.EncodedLen(profile.MaxImageBytes) {
		return nil, errors.New("imagem acima de 256 KiB")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(image.Data)
	if err != nil || len(data) == 0 || len(data) > profile.MaxImageBytes {
		return nil, errors.New("imagem inválida ou acima do limite")
	}
	return data, nil
}

func (c *Core) Profiles(ids []string) ([]profile.Metadata, error) {
	out := []profile.Metadata{}
	if len(ids) > 200 {
		return out, errors.New("consulte até 200 perfis de cada vez")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if len(id) > 128 || seen[id] {
			continue
		}
		seen[id] = true
		if item, ok := c.profiles.Get(id); ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (c *Core) ProfileAsset(hash string) (*ImageAsset, error) {
	asset, data, err := c.profiles.ReadImage(hash)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ImageAsset{MIME: asset.MIME, Data: base64.StdEncoding.EncodeToString(data)}, nil
}

func (c *Core) SaveProfile(change ProfileChange) (profile.Metadata, error) {
	data, err := change.Avatar.decode()
	if err != nil {
		return profile.Metadata{}, err
	}
	if change.RemoveAvatar && data != nil {
		return profile.Metadata{}, errors.New("selecione ou remova o avatar")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return profile.Metadata{}, errors.New("perfil fechado")
	}
	current := c.profiles.Local()
	hash := current.AvatarHash
	if change.RemoveAvatar {
		hash = ""
	}
	if data != nil {
		asset, err := c.profiles.PutImage(data)
		if err != nil {
			return profile.Metadata{}, err
		}
		hash = asset.Hash
	}
	settings, err := c.db.Settings()
	if err != nil {
		return profile.Metadata{}, err
	}
	next, err := c.profiles.Update(change.DisplayName, change.DisplayFontID, hash)
	if err != nil {
		return profile.Metadata{}, err
	}
	if next.Revision != current.Revision && c.node != nil {
		c.node.ProfileChanged()
	}
	if name := legacyProfileName(next.DisplayName); settings.Name != name {
		settings.Name = name
		if err = c.db.SaveSettings(settings); err != nil {
			return next, err
		}
	}
	return next, nil
}

func (c *Core) SaveCommunityArtwork(id, channel string, image *ImageAsset) (community.Artwork, error) {
	data, err := image.decode()
	if err != nil {
		return community.Artwork{}, err
	}
	return c.groups.SaveArtwork(id, channel, data)
}
