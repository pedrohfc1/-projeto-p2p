package communities

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"projetop2p/internal/profile"
)

type artworkTransport interface {
	FetchProfileAsset(context.Context, string, string) (profile.Asset, error)
}

func (s *Service) SaveArtwork(id, channel string, image []byte) (community.Artwork, error) {
	if s.assets == nil {
		return community.Artwork{}, errors.New("cache de imagens indisponível")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	room, err := s.db.Community(id)
	if err != nil {
		return community.Artwork{}, err
	}
	if room.Denied || room.State.Archived || room.Owner != s.id || room.State.Role(s.id) != "owner" {
		return community.Artwork{}, errors.New("somente o proprietário pode alterar a imagem")
	}
	if channel != "" {
		ch, ok := room.State.Channels[channel]
		if !ok || ch.Deleted || ch.Kind != "voice" {
			return community.Artwork{}, errors.New("sala de voz indisponível")
		}
	}
	old, err := s.db.Artwork(id)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return community.Artwork{}, err
	}
	icons := make(map[string]string, len(old.Icons)+1)
	for k, v := range old.Icons {
		if k == "" || !room.State.Channels[k].Deleted {
			icons[k] = v
		}
	}
	if image == nil {
		delete(icons, channel)
	} else {
		asset, err := s.assets.PutImage(image)
		if err != nil {
			return community.Artwork{}, err
		}
		icons[channel] = asset.Hash
	}
	a, err := community.SignArtwork(s.key, room.State, old.Revision+1, icons)
	if err != nil {
		return a, err
	}
	if err = s.db.SaveArtwork(a, s.id); err != nil {
		return a, err
	}
	s.keepArtwork()
	slog.Info("community_icon_updated", "voice_channel", channel != "", "revision", a.Revision)
	return a, nil
}

// AuthorizeArtwork is also used on the bounded asset stream after the requester
// has discovered a hash. Possession of a hash alone does not confer access.
func (s *Service) AuthorizeArtwork(remote, hash string) bool {
	if s.assets == nil || !community.ValidID(hash) {
		return false
	}
	records, err := s.db.Communities()
	if err != nil {
		return false
	}
	for _, r := range records {
		if r.Owner != s.id || r.State.Role(remote) == "" || r.Denied {
			continue
		}
		item, err := s.db.Artwork(r.ID)
		if err != nil {
			continue
		}
		for key, value := range item.Icons {
			if hash == value && (key == "" || !r.State.Channels[key].Deleted) {
				return true
			}
		}
	}
	return false
}

func (s *Service) keepArtwork() {
	if s.assets == nil {
		return
	}
	if hashes, err := s.db.ArtworkHashes(); err == nil {
		s.assets.SetArtworkReferences(hashes)
	}
}

func (s *Service) syncArtwork(ctx context.Context, r localdb.CommunityRecord) {
	if s.assets == nil || r.Denied || r.State.Role(s.id) == "" {
		return
	}
	transport, _ := s.environment()
	fetcher, ok := transport.(artworkTransport)
	if !ok {
		return
	}
	old, _ := s.db.Artwork(r.ID)
	response, err := s.request(ctx, r, community.Request{Method: community.ArtworkRequest, CommunityID: r.ID, After: old.Revision})
	if err != nil || response.Error != "" {
		return
	}
	a := old
	if response.Artwork != nil {
		a = *response.Artwork
		if a.Revision <= old.Revision {
			return
		}
	}
	if a.Verify(r.State) != nil {
		return
	}
	for channel, hash := range a.Icons {
		if channel != "" && r.State.Channels[channel].Deleted {
			continue
		}
		if s.assets.HasImage(hash) {
			continue
		}
		if _, err := fetcher.FetchProfileAsset(ctx, r.Owner, hash); err != nil {
			return
		}
	}
	if a.Revision > old.Revision && s.db.SaveArtwork(a, s.id) == nil {
		s.keepArtwork()
	}
}
