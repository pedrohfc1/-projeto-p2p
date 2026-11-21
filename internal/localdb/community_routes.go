package localdb

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/identity"
)

// Signed endpoint capabilities are sealed at rest; never copied into diagnostics.
func (d *DB) CommunityRoutes(id string) (map[string]string, error) {
	out := map[string]string{}
	var data []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		data = append([]byte(nil), tx.Bucket([]byte("settings")).Get([]byte("community-routes:"+id))...)
		return nil
	})
	if err != nil || len(data) == 0 {
		return out, err
	}
	if len(data) > 512<<10 {
		return out, errors.New("route cache too large")
	}
	raw, err := identity.UnsealLocalData(data)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]string{}
	}
	if len(out) > 100 {
		return map[string]string{}, errors.New("route cache count limit")
	}
	return out, err
}
func (d *DB) SaveCommunityRoutes(id string, routes map[string]string) error {
	if len(routes) > 100 {
		return errors.New("route cache limit")
	}
	raw, err := json.Marshal(routes)
	if err != nil || len(raw) > 320000 {
		return errors.New("route cache size limit")
	}
	sealed, err := identity.SealLocalData(raw)
	if err != nil {
		return err
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("settings")).Put([]byte("community-routes:"+id), sealed)
	})
}
