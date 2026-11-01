package localdb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/community"
)

const CurrentSchema uint64 = 3

var communityBuckets = []string{"communities", "community_events", "community_opids", "community_messages", "community_outbox", "community_peer_text", "community_peer_ids", "community_heads"}

func (d *DB) SchemaVersion() (uint64, error) {
	var version uint64
	e := d.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("schema")); b != nil {
			v := b.Get([]byte("version"))
			if v != nil && len(v) != 8 {
				return errors.New("corrupt schema version; database preserved")
			}
			if len(v) == 8 {
				version = binary.BigEndian.Uint64(v)
			}
		}
		if version == 0 {
			version = 1
		}
		return nil
	})
	return version, e
}
func (d *DB) MigrateTo(target uint64) error {
	if target < 1 || target > CurrentSchema {
		return errors.New("unsupported schema target")
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		version := uint64(1)
		if b := tx.Bucket([]byte("schema")); b != nil {
			v := b.Get([]byte("version"))
			if v != nil && len(v) != 8 {
				return errors.New("corrupt schema version; database preserved")
			}
			if len(v) == 8 {
				version = binary.BigEndian.Uint64(v)
			}
		}
		if version > CurrentSchema {
			return errors.New("database is newer than this application; preserved without migration")
		}
		for _, name := range []string{"settings", "contacts", "messages", "messageids", "schema"} {
			if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
				return e
			}
		}
		if target < version {
			if version >= 3 {
				return errors.New("v0.6 community policies cannot be downgraded; restore a backup instead")
			}
			for _, name := range communityBuckets {
				if b := tx.Bucket([]byte(name)); b != nil {
					key, _ := b.Cursor().First()
					if key != nil {
						return errors.New("cannot downgrade nonempty community data; export/backup first")
					}
				}
			}
			for _, name := range communityBuckets {
				if tx.Bucket([]byte(name)) != nil {
					if e := tx.DeleteBucket([]byte(name)); e != nil {
						return e
					}
				}
			}
			if e := tx.Bucket([]byte("schema")).Delete([]byte("communityBytes")); e != nil {
				return e
			}
		} else if target >= 2 {
			for _, name := range communityBuckets {
				if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
					return e
				}
			}
		}
		if target >= 3 {
			parent := tx.Bucket([]byte("community_events"))
			if parent != nil {
				if e := parent.ForEach(func(id, v []byte) error {
					events := parent.Bucket(id)
					if events == nil {
						return nil
					}
					heads, e := sub(tx, "community_heads", string(id))
					if e != nil {
						return e
					}
					if heads.Stats().KeyN >= events.Stats().KeyN {
						return nil
					}
					return events.ForEach(func(k, v []byte) error {
						var c community.Commit
						if e := json.Unmarshal(v, &c); e != nil {
							return e
						}
						return heads.Put([]byte(c.Hash), k)
					})
				}); e != nil {
					return e
				}
			}
		}
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, target)
		return tx.Bucket([]byte("schema")).Put([]byte("version"), b)
	})
}
