package localdb

import (
	"encoding/json"
	"errors"
	"os"
	"projetop2p/internal/community"

	bolt "go.etcd.io/bbolt"
)

var artworkBucket = []byte("community_artwork")

func (d *DB) Artwork(id string) (artwork community.Artwork, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(artworkBucket)
		if bucket == nil {
			return os.ErrNotExist
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return os.ErrNotExist
		}
		return json.Unmarshal(data, &artwork)
	})
	return
}

// SaveArtwork refuses a replay or a different manifest at the same revision.
func (d *DB) SaveArtwork(artwork community.Artwork, self string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		key := []byte(artwork.Community)
		data := tx.Bucket([]byte("communities")).Get(key)
		var room CommunityRecord
		if data == nil {
			return os.ErrNotExist
		}
		if err := json.Unmarshal(data, &room); err != nil {
			return err
		}
		if room.Denied || room.State.Role(self) == "" {
			return errors.New("participação atual exigida")
		}
		if err := artwork.Verify(room.State); err != nil {
			return err
		}
		for id := range artwork.Icons {
			if id != "" && room.State.Channels[id].Kind != "voice" {
				return errors.New("invalid artwork channel")
			}
		}
		bucket, err := tx.CreateBucketIfNotExists(artworkBucket)
		if err != nil {
			return err
		}
		if data := bucket.Get(key); data != nil {
			var old community.Artwork
			if err = json.Unmarshal(data, &old); err != nil {
				return err
			}
			if artwork.Revision <= old.Revision {
				return errors.New("revisão de imagem desatualizada")
			}
		}
		data, err = json.Marshal(artwork)
		if err != nil {
			return err
		}
		return bucket.Put(key, data)
	})
}

func (d *DB) ArtworkHashes() ([]string, error) {
	var hashes []string
	err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(artworkBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(_ []byte, data []byte) error {
			var artwork community.Artwork
			if err := json.Unmarshal(data, &artwork); err != nil {
				return err
			}
			for _, hash := range artwork.Icons {
				hashes = append(hashes, hash)
			}
			return nil
		})
	})
	return hashes, err
}
