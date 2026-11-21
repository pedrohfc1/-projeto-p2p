package localdb

import (
	bolt "go.etcd.io/bbolt"
)

func (d *DB) ActivityCursor(key string) (uint64, error) {
	var out uint64
	e := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("activity_cursors"))
		if b != nil {
			out = binaryValue(b.Get([]byte(key)))
		}
		return nil
	})
	return out, e
}
func (d *DB) SaveActivityCursor(key string, value uint64) error {
	if len(key) > 384 {
		return nil
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte("activity_cursors"))
		if e != nil {
			return e
		}
		if b.Get([]byte(key)) == nil && b.Stats().KeyN >= 8192 {
			return nil
		}
		return b.Put([]byte(key), seqkey(value))
	})
}
func (d *DB) ActivityHead(context string) uint64 {
	var out uint64
	_ = d.db.View(func(tx *bolt.Tx) error {
		if b := activityRead(tx, "activity_log", context); b != nil {
			k, _ := b.Cursor().Last()
			out = binaryValue(k)
		}
		return nil
	})
	return out
}
