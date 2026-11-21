package localdb

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/chat"
	"projetop2p/internal/community"
	"strings"
	"time"
)

func activityBucket(tx *bolt.Tx, name, context string) (*bolt.Bucket, error) {
	b, e := tx.CreateBucketIfNotExists([]byte(name))
	if e != nil {
		return nil, e
	}
	return b.CreateBucketIfNotExists([]byte(context))
}
func activityRead(tx *bolt.Tx, name, context string) *bolt.Bucket {
	b := tx.Bucket([]byte(name))
	if b == nil {
		return nil
	}
	return b.Bucket([]byte(context))
}
func activityACL(tx *bolt.Tx, r chat.Room, self, remote string) error {
	if r.Group != "" {
		g, e := getRecord(tx, r.Group)
		if e != nil {
			return e
		}
		if g.Denied || g.State.Role(self) == "" || g.State.Role(remote) == "" {
			return errors.New("access_denied: current membership required")
		}
		return nil
	}
	other := r.Peer
	if remote != self {
		other = remote
	}
	if other == self {
		return errors.New("invalid direct recipient")
	}
	b := tx.Bucket([]byte("contacts"))
	raw := b.Get([]byte(other))
	if raw == nil {
		return errors.New("contact required")
	}
	var contact Contact
	if json.Unmarshal(raw, &contact) != nil || contact.Unsolicited {
		return errors.New("verified contact required")
	}
	return nil
}
func (d *DB) ActivityAccess(r chat.Room, self, remote string) error {
	return d.db.View(func(tx *bolt.Tx) error { return activityACL(tx, r, self, remote) })
}
func activitySource(tx *bolt.Tx, e chat.Event, self, context string) (chat.Overlay, error) {
	reference := e.Target
	if e.Room.Group == "" {
		reference = e.TargetAuthor + ":" + e.Target
	}
	if b := activityRead(tx, "activity_projection", context); b != nil {
		if raw := b.Get([]byte(reference)); raw != nil {
			var p chat.Overlay
			err := json.Unmarshal(raw, &p)
			return p, err
		}
	}
	if e.Room.Group == "" {
		ids := tx.Bucket([]byte("messageids"))
		for _, author := range []string{e.TargetAuthor} {
			if key := ids.Get([]byte(author + ":" + e.Target)); key != nil {
				var m Message
				if json.Unmarshal(tx.Bucket([]byte("messages")).Get(key), &m) == nil && chat.Context(chat.Room{Peer: m.Peer}, self, self) == context {
					return chat.Overlay{ID: m.ID, Author: m.From, Text: m.Text, Time: m.Time}, nil
				}
			}
		}
	} else {
		id := e.Room.Group
		if ids := tx.Bucket([]byte("community_opids")).Bucket([]byte(id)); ids != nil {
			if seq := ids.Get([]byte(e.Target)); seq != nil {
				var commit community.Commit
				b := tx.Bucket([]byte("community_events")).Bucket([]byte(id))
				if b != nil && json.Unmarshal(b.Get(seq), &commit) == nil && commit.Operation.Kind == community.Text {
					return chat.Overlay{ID: e.Target, Author: commit.Operation.Author, Text: commit.Operation.Body.Text, Channel: commit.Operation.Body.ChannelID, Time: commit.At}, nil
				}
			}
		}
		if ids := tx.Bucket([]byte("community_peer_ids")).Bucket([]byte(id)); ids != nil {
			if seq := ids.Get([]byte(e.Target)); seq != nil {
				var entry community.PeerTextEntry
				b := tx.Bucket([]byte("community_peer_text")).Bucket([]byte(id))
				if b != nil && json.Unmarshal(b.Get(seq), &entry) == nil {
					return chat.Overlay{ID: e.Target, Author: entry.Message.Operation.Author, Text: entry.Message.Operation.Body.Text, Channel: entry.Message.Operation.Body.ChannelID, Time: entry.Message.Operation.Issued}, nil
				}
			}
		}
	}
	return chat.Overlay{}, errors.New("target_missing: synchronize original message first")
}
func (d *DB) SaveActivity(e chat.Event, self, remote string) error {
	if err := chat.Verify(e, time.Now()); err != nil {
		return err
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		if err := activityACL(tx, e.Room, self, remote); err != nil {
			return err
		}
		context := chat.Context(e.Room, self, e.Actor)
		if e.Room.Group != "" {
			r, err := getRecord(tx, e.Room.Group)
			if err != nil {
				return err
			}
			ch, ok := r.State.Channels[e.Room.Channel]
			heads := tx.Bucket([]byte("community_heads")).Bucket([]byte(r.ID))
			if heads == nil || heads.Get([]byte(e.Head)) == nil || e.Epoch > r.State.InviteEpoch {
				return errors.New("sync_required: missing verified control prefix")
			}
			if r.State.Root != e.Root || r.State.InviteEpoch != e.Epoch || r.State.Archived || r.State.Role(e.Actor) == "" || !ok || ch.Deleted || ch.Kind != "text" || heads == nil || heads.Get([]byte(e.Head)) == nil {
				return errors.New("access_denied: invalid signed community authorization")
			}
		} else if !(e.Actor == self && e.Room.Peer == remote || e.Actor == remote && e.Room.Peer == self || remote == self && e.Actor == self) {
			return errors.New("direct activity audience mismatch")
		}
		if e.Kind == "typing" || e.Kind == "voice" {
			return errors.New("typing must remain ephemeral")
		}
		ids, err := activityBucket(tx, "activity_ids", context)
		if err != nil {
			return err
		}
		if ids.Get([]byte(e.ID)) != nil {
			return nil
		}
		source, err := activitySource(tx, e, self, context)
		if err != nil {
			return err
		}
		if source.Channel != e.Room.Channel {
			return errors.New("cross-channel target")
		}
		if (e.Kind == "edit" || e.Kind == "delete") && source.Author != e.Actor {
			return errors.New("access_denied: only the message author can modify it")
		}
		if source.Deleted && e.Kind != "delete" {
			return errors.New("message deleted")
		}
		b, err := activityBucket(tx, "activity_log", context)
		if err != nil {
			return err
		}
		if b.Stats().KeyN >= 5000 {
			return errors.New("activity event quota reached")
		}
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(chat.Entry{Sequence: seq, Event: e})
		meta := tx.Bucket([]byte("settings"))
		total := binaryValue(meta.Get([]byte("activityBytes")))
		if total+uint64(len(raw)) > 32<<20 {
			return errors.New("activity storage quota reached")
		}
		projection := source
		if e.Kind == "reply" {
			projection = chat.Overlay{ID: e.ID, Author: e.Actor, Channel: e.Room.Channel, Text: e.Text, Time: e.At / 1000, ReplyTo: e.Target}
		} else {
			projection = chat.Apply(source, e)
		}
		value, _ := json.Marshal(projection)
		if len(value) > 128<<10 {
			return errors.New("activity projection quota reached")
		}
		p, err := activityBucket(tx, "activity_projection", context)
		if err != nil {
			return err
		}
		reference := projection.ID
		if e.Room.Group == "" {
			reference = projection.Author + ":" + projection.ID
		}
		previous := len(p.Get([]byte(reference)))
		next := int64(total) + int64(len(raw)) + int64(len(value)-previous)
		if next < 0 || next > 32<<20 {
			return errors.New("activity storage quota reached")
		}
		if e.Kind == "reply" {
			index, err := activityBucket(tx, "activity_replies", context)
			if err != nil {
				return err
			}
			if err = index.Put(seqkey(seq), []byte(reference)); err != nil {
				return err
			}
			next += int64(len(e.ID) + 8)
			if next > 32<<20 {
				return errors.New("activity storage quota reached")
			}
		}
		if err = p.Put([]byte(reference), value); err != nil {
			return err
		}
		if err = b.Put(seqkey(seq), raw); err != nil {
			return err
		}
		if err = ids.Put([]byte(e.ID), seqkey(seq)); err != nil {
			return err
		}
		return meta.Put([]byte("activityBytes"), seqkey(uint64(next)))
	})
}
func (d *DB) Activities(context string, after uint64) ([]chat.Entry, error) {
	out := []chat.Entry{}
	err := d.db.View(func(tx *bolt.Tx) error {
		b := activityRead(tx, "activity_log", context)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(seqkey(after + 1)); k != nil && len(out) < 16; k, v = c.Next() {
			var entry chat.Entry
			if err := json.Unmarshal(v, &entry); err != nil {
				return err
			}
			out = append(out, entry)
		}
		return nil
	})
	return out, err
}
func (d *DB) ActivityOverlays(context string, ids []string, channel string, replies bool) (map[string]chat.Overlay, error) {
	out := map[string]chat.Overlay{}
	err := d.db.View(func(tx *bolt.Tx) error {
		b := activityRead(tx, "activity_projection", context)
		if b == nil {
			return nil
		}
		for _, id := range ids {
			if raw := b.Get([]byte(id)); raw != nil {
				var p chat.Overlay
				if err := json.Unmarshal(raw, &p); err != nil {
					return err
				}
				out[id] = p
			}
		}
		if replies {
			if index := activityRead(tx, "activity_replies", context); index != nil {
				c := index.Cursor()
				n := 0
				for k, id := c.Last(); k != nil && n < 100; k, id = c.Prev() {
					var row chat.Overlay
					if json.Unmarshal(b.Get(id), &row) != nil {
						continue
					}
					if row.Channel == channel {
						out[string(id)] = row
						n++
					}
				}
			}
		}

		return nil
	})
	return out, err
}

func (d *DB) DirectTargetAuthor(self, other, reference string) (string, string, error) {
	id := reference
	author := ""
	if parts := strings.Split(reference, ":"); len(parts) == 2 {
		author, id = parts[0], parts[1]
	}
	found := []string{}
	err := d.db.View(func(tx *bolt.Tx) error {
		for _, candidate := range []string{self, other} {
			if author != "" && candidate != author {
				continue
			}
			context := "d:" + other
			if b := activityRead(tx, "activity_projection", context); b != nil && b.Get([]byte(candidate+":"+id)) != nil {
				found = append(found, candidate)
				continue
			}
			if key := tx.Bucket([]byte("messageids")).Get([]byte(candidate + ":" + id)); key != nil {
				var m Message
				if json.Unmarshal(tx.Bucket([]byte("messages")).Get(key), &m) == nil && m.Peer == other {
					found = append(found, candidate)
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	if len(found) != 1 {
		return "", "", errors.New("target_missing: use an unambiguous author-qualified message reference")
	}
	return found[0], id, nil
}
