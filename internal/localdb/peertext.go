package localdb

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/community"
	"time"
)

// Atomic deduplication, authorization and shared profile quotas. No route credentials are stored here.
func (d *DB) SavePeerText(m community.PeerText, self string) error {
	if e := community.VerifyPeerText(m); e != nil {
		return e
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		r, e := getRecord(tx, m.Operation.CommunityID)
		if e != nil {
			return e
		}
		if r.Denied || r.State.Role(self) == "" {
			return errors.New("current membership required")
		}
		ids, e := sub(tx, "community_peer_ids", r.ID)
		if e != nil {
			return e
		}
		if ids.Get([]byte(m.Operation.ID)) != nil {
			return nil
		}
		heads := tx.Bucket([]byte("community_heads")).Bucket([]byte(r.ID))
		if heads == nil || heads.Get([]byte(m.Operation.Base)) == nil {
			return errors.New("sync_failure: unknown owner control prefix")
		}
		if e = community.AcceptPeerText(m, r.State, time.Now()); e != nil {
			return e
		}
		b, e := sub(tx, "community_peer_text", r.ID)
		if e != nil {
			return e
		}
		if b.Stats().KeyN >= 10000 {
			return errors.New("community peer-text event limit reached")
		}
		seq, e := b.NextSequence()
		if e != nil {
			return e
		}
		raw, e := json.Marshal(community.PeerTextEntry{Sequence: seq, Message: m})
		if e != nil {
			return e
		}
		meta := tx.Bucket([]byte("schema"))
		total := uint64(0)
		if v := meta.Get([]byte("communityBytes")); len(v) == 8 {
			total = binaryValue(v)
		}
		if total+uint64(len(raw)) > maxTotalCommunityLog || r.LogBytes+uint64(len(raw)) > maxCommunityLog {
			return errors.New("community storage quota reached")
		}
		if e = b.Put(seqkey(seq), raw); e != nil {
			return e
		}
		if e = ids.Put([]byte(m.Operation.ID), seqkey(seq)); e != nil {
			return e
		}
		r.LogBytes += uint64(len(raw))
		if e = putRecord(tx, r); e != nil {
			return e
		}
		return meta.Put([]byte("communityBytes"), seqkey(total+uint64(len(raw))))
	})
}
func binaryValue(v []byte) uint64 {
	var n uint64
	for _, x := range v {
		n = n<<8 | uint64(x)
	}
	return n
}

// Arrival sequence is local to this replica, used only as a replication cursor.
func (d *DB) PeerTexts(id string, after uint64, limit int) ([]community.PeerTextEntry, error) {
	if limit < 1 || limit > 100 {
		limit = 16
	}
	out := []community.PeerTextEntry{}
	e := d.db.View(func(tx *bolt.Tx) error {
		p := tx.Bucket([]byte("community_peer_text"))
		if p == nil {
			return nil
		}
		b := p.Bucket([]byte(id))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(seqkey(after + 1)); k != nil && len(out) < limit; k, v = c.Next() {
			var m community.PeerTextEntry
			if e := json.Unmarshal(v, &m); e != nil {
				return e
			}
			out = append(out, m)
		}
		return nil
	})
	return out, e
}
func (d *DB) PeerTextHistory(id, channel string, before uint64) ([]community.PeerTextEntry, error) {
	out := []community.PeerTextEntry{}
	e := d.db.View(func(tx *bolt.Tx) error {
		p := tx.Bucket([]byte("community_peer_text"))
		if p == nil {
			return nil
		}
		b := p.Bucket([]byte(id))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		k, v := c.Last()
		if before > 0 {
			k, v = c.Seek(seqkey(before))
			if k == nil {
				k, v = c.Last()
			} else {
				k, v = c.Prev()
			}
		}
		for ; k != nil && len(out) < 100; k, v = c.Prev() {
			var m community.PeerTextEntry
			if e := json.Unmarshal(v, &m); e != nil {
				return e
			}
			if m.Message.Operation.Body.ChannelID == channel {
				out = append(out, m)
			}
		}
		return nil
	})
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, e
}
