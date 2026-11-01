package localdb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/community"
	"sort"
	"time"
)

const MaxCommunities = 32
const maxCommunityLog = 16 << 20
const maxTotalCommunityLog = 64 << 20
const maxOutboxBytes = 2 << 20

type CommunityRecord struct {
	ID        string          `json:"id"`
	Owner     string          `json:"owner"`
	Root      string          `json:"root"`
	Name      string          `json:"name"`
	Scope     string          `json:"scope"`
	Addresses []string        `json:"addresses"`
	State     community.State `json:"state"`
	LogBytes  uint64          `json:"logBytes"`
	LastSync  int64           `json:"lastSync"`
	LastError string          `json:"lastError"`
	Denied    bool            `json:"denied"`
}
type PendingCommunity struct {
	Order     uint64              `json:"order"`
	Operation community.Operation `json:"operation"`
	Status    string              `json:"status"`
	Error     string              `json:"error"`
}

func seqkey(n uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, n); return b }
func getRecord(tx *bolt.Tx, id string) (CommunityRecord, error) {
	var r CommunityRecord
	b := tx.Bucket([]byte("communities")).Get([]byte(id))
	if b == nil {
		return r, errors.New("community not found")
	}
	e := json.Unmarshal(b, &r)
	return r, e
}
func putRecord(tx *bolt.Tx, r CommunityRecord) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	return tx.Bucket([]byte("communities")).Put([]byte(r.ID), b)
}
func sub(tx *bolt.Tx, name, id string) (*bolt.Bucket, error) {
	return tx.Bucket([]byte(name)).CreateBucketIfNotExists([]byte(id))
}
func (d *DB) Community(id string) (CommunityRecord, error) {
	var r CommunityRecord
	e := d.db.View(func(tx *bolt.Tx) error { var e error; r, e = getRecord(tx, id); return e })
	return r, e
}
func (d *DB) Communities() ([]CommunityRecord, error) {
	out := []CommunityRecord{}
	e := d.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("communities")).ForEach(func(k, v []byte) error {
			var r CommunityRecord
			if e := json.Unmarshal(v, &r); e != nil {
				return e
			}
			out = append(out, r)
			return nil
		})
	})
	return out, e
}

// A new record pins owner/root from a verified invitation. Only local code invokes this method.
func (d *DB) TrackCommunity(r CommunityRecord) error {
	if !community.ValidID(r.ID) || !community.ValidID(r.Root) {
		return errors.New("invalid community identity")
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		if old, e := getRecord(tx, r.ID); e == nil {
			if old.Owner != r.Owner || old.Root != r.Root {
				return errors.New("community root/owner mismatch")
			}
			old.Scope = r.Scope
			old.Addresses = r.Addresses
			old.Denied = false
			return putRecord(tx, old)
		}
		if tx.Bucket([]byte("communities")).Stats().KeyN >= MaxCommunities {
			return errors.New("community limit reached")
		}
		return putRecord(tx, r)
	})
}
func (d *DB) CommunityCommit(c community.Commit) (bool, error) {
	if e := community.VerifyCommit(c); e != nil {
		return false, e
	}
	applied := false
	e := d.db.Update(func(tx *bolt.Tx) error { return applyCommunityTx(tx, c, &applied) })
	return applied, e
}
func (d *DB) FindCommunityOperation(id, operationID string) (uint64, error) {
	var n uint64
	e := d.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("community_opids")).Bucket([]byte(id)); b != nil {
			if raw := b.Get([]byte(operationID)); len(raw) == 8 {
				n = binary.BigEndian.Uint64(raw)
			}
		}
		return nil
	})
	return n, e
}
func (d *DB) CommunityEvents(id string, after uint64, limit int) ([]community.Commit, error) {
	if limit < 1 || limit > community.BatchSize {
		limit = community.BatchSize
	}
	out := []community.Commit{}
	e := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("community_events")).Bucket([]byte(id))
		if b == nil {
			return nil
		}
		cursor := b.Cursor()
		for k, v := cursor.Seek(seqkey(after + 1)); k != nil && len(out) < limit; k, v = cursor.Next() {
			var c community.Commit
			if e := json.Unmarshal(v, &c); e != nil {
				return e
			}
			out = append(out, c)
		}
		return nil
	})
	return out, e
}
func (d *DB) CommunityEvent(id string, n uint64) (community.Commit, error) {
	var c community.Commit
	e := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("community_events")).Bucket([]byte(id))
		if b == nil || b.Get(seqkey(n)) == nil {
			return errors.New("event not found")
		}
		return json.Unmarshal(b.Get(seqkey(n)), &c)
	})
	return c, e
}
func (d *DB) CommunityHistory(id, channel string, before uint64, limit int) ([]community.Commit, error) {
	if !community.ChannelID(channel) {
		return nil, errors.New("invalid channel")
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}
	out := []community.Commit{}
	e := d.db.View(func(tx *bolt.Tx) error {
		parent := tx.Bucket([]byte("community_messages")).Bucket([]byte(id))
		if parent == nil {
			return nil
		}
		b := parent.Bucket([]byte(channel))
		if b == nil {
			return nil
		}
		events := tx.Bucket([]byte("community_events")).Bucket([]byte(id))
		cur := b.Cursor()
		k, _ := cur.Last()
		if before > 0 {
			k, _ = cur.Seek(seqkey(before))
			if k == nil {
				k, _ = cur.Last()
			} else {
				k, _ = cur.Prev()
			}
		}
		for ; k != nil && len(out) < limit; k, _ = cur.Prev() {
			var c community.Commit
			if e := json.Unmarshal(events.Get(k), &c); e != nil {
				return e
			}
			out = append(out, c)
		}
		return nil
	})
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, e
}
func (d *DB) QueueCommunity(o community.Operation) error {
	if e := community.VerifyOperation(o); e != nil {
		return e
	}
	raw, e := json.Marshal(PendingCommunity{Operation: o, Status: "pendente"})
	if e != nil {
		return e
	}
	if len(raw) > community.MaxRequest {
		return errors.New("pending operation too large")
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		if _, e := getRecord(tx, o.CommunityID); e != nil {
			return e
		}
		b, e := sub(tx, "community_outbox", o.CommunityID)
		if e != nil {
			return e
		}
		if b.Get([]byte(o.ID)) != nil {
			return nil
		}
		if b.Stats().KeyN >= 64 {
			return errors.New("pending operation limit reached")
		}
		total := 0
		e = tx.Bucket([]byte("community_outbox")).ForEach(func(k, v []byte) error {
			return tx.Bucket([]byte("community_outbox")).Bucket(k).ForEach(func(k, v []byte) error { total += len(v); return nil })
		})
		if e != nil {
			return e
		}
		if total+len(raw) > maxOutboxBytes {
			return errors.New("profile pending-operation quota reached")
		}
		order, e := b.NextSequence()
		if e != nil {
			return e
		}
		raw, e = json.Marshal(PendingCommunity{Operation: o, Status: "pendente", Order: order})
		if e != nil {
			return e
		}
		if total+len(raw) > maxOutboxBytes {
			return errors.New("profile pending-operation quota reached")
		}
		return b.Put([]byte(o.ID), raw)
	})
}
func (d *DB) CommunityPending(id string) ([]PendingCommunity, error) {
	out := []PendingCommunity{}
	e := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("community_outbox")).Bucket([]byte(id))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var q PendingCommunity
			if e := json.Unmarshal(v, &q); e != nil {
				return e
			}
			out = append(out, q)
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Operation.ID < out[j].Operation.ID
		}
		return out[i].Order < out[j].Order
	})
	return out, e
}
func (d *DB) CommunityPendingStatus(id, op, status, message string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("community_outbox")).Bucket([]byte(id))
		if b == nil {
			return nil
		}
		if status == "delete" {
			return b.Delete([]byte(op))
		}
		var q PendingCommunity
		if e := json.Unmarshal(b.Get([]byte(op)), &q); e != nil {
			return e
		}
		q.Status = status
		if len(message) > 1024 {
			message = message[:1024]
		}
		q.Error = message
		raw, e := json.Marshal(q)
		if e != nil {
			return e
		}
		return b.Put([]byte(op), raw)
	})
}
func (d *DB) CommunityStatus(id, message string, denied bool) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		r, e := getRecord(tx, id)
		if e != nil {
			return e
		}
		if len(message) > 1024 {
			message = message[:1024]
		}
		r.LastError = message
		r.Denied = denied
		r.LastSync = time.Now().Unix()
		return putRecord(tx, r)
	})
}

func applyCommunityTx(tx *bolt.Tx, c community.Commit, applied *bool) error {
	r, e := getRecord(tx, c.CommunityID)
	if e != nil {
		return e
	}
	events, e := sub(tx, "community_events", r.ID)
	if e != nil {
		return e
	}
	if old := events.Get(seqkey(c.Sequence)); old != nil {
		var previous community.Commit
		if e = json.Unmarshal(old, &previous); e != nil {
			return e
		}
		if previous.Hash != c.Hash {
			return errors.New("fork detected at existing sequence")
		}
		return nil
	}
	if r.State.Sequence >= 50000 {
		return errors.New("community event limit reached")
	}
	if r.Owner != c.Owner || (r.State.Sequence == 0 && r.Root != c.Hash) {
		return errors.New("commit does not match pinned root/owner")
	}
	next, e := community.Apply(r.State, c)
	if e != nil {
		return e
	}
	ids, e := sub(tx, "community_opids", r.ID)
	if e != nil {
		return e
	}
	if ids.Get([]byte(c.Operation.ID)) != nil {
		return errors.New("operation ID already committed at another sequence")
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return e
	}
	if r.LogBytes+uint64(len(raw)) > maxCommunityLog {
		return errors.New("community log quota reached")
	}
	meta := tx.Bucket([]byte("schema"))
	total := uint64(0)
	if v := meta.Get([]byte("communityBytes")); len(v) == 8 {
		total = binary.BigEndian.Uint64(v)
	}
	if total+uint64(len(raw)) > maxTotalCommunityLog {
		return errors.New("profile community log quota reached")
	}
	if e = events.Put(seqkey(c.Sequence), raw); e != nil {
		return e
	}
	if e = ids.Put([]byte(c.Operation.ID), seqkey(c.Sequence)); e != nil {
		return e
	}
	if c.Operation.Kind == community.Text {
		parent, e := sub(tx, "community_messages", r.ID)
		if e != nil {
			return e
		}
		messages, e := parent.CreateBucketIfNotExists([]byte(c.Operation.Body.ChannelID))
		if e != nil {
			return e
		}
		if e = messages.Put(seqkey(c.Sequence), []byte(c.Operation.ID)); e != nil {
			return e
		}
	}
	if parent := tx.Bucket([]byte("community_outbox")).Bucket([]byte(r.ID)); parent != nil {
		if e = parent.Delete([]byte(c.Operation.ID)); e != nil {
			return e
		}
	}
	r.State = next
	r.Name = next.Name
	r.LogBytes += uint64(len(raw))
	r.LastError = ""
	r.Denied = false
	if e = putRecord(tx, r); e != nil {
		return e
	}
	if e = meta.Put([]byte("communityBytes"), seqkey(total+uint64(len(raw)))); e != nil {
		return e
	}
	*applied = true
	return nil
}

func (d *DB) CreateOwnedCommunity(r CommunityRecord, c community.Commit) error {
	if e := community.VerifyCommit(c); e != nil {
		return e
	}
	if c.Sequence != 1 || r.ID != c.CommunityID || r.Owner != c.Owner || r.Root != c.Hash {
		return errors.New("invalid owned community genesis")
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("communities")).Get([]byte(r.ID)) != nil {
			return errors.New("community already exists")
		}
		if tx.Bucket([]byte("communities")).Stats().KeyN >= MaxCommunities {
			return errors.New("community limit reached")
		}
		if e := putRecord(tx, r); e != nil {
			return e
		}
		applied := false
		return applyCommunityTx(tx, c, &applied)
	})
}
