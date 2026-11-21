package localdb

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"projetop2p/internal/chat"
	"projetop2p/internal/community"
	"strings"
	"unicode"
)

type NotificationPreferences struct {
	Enabled bool            `json:"enabled"`
	Preview bool            `json:"preview"`
	Typing  bool            `json:"typing"`
	Muted   map[string]bool `json:"muted"`
}
type Notice struct {
	ID      string `json:"id"`
	Context string `json:"context"`
	Kind    string `json:"kind"`
	Body    string `json:"body"`
}
type inboxState struct {
	Cursors map[string]uint64 `json:"cursors"`
	Unread  map[string]int    `json:"unread"`
	Seen    []string          `json:"seen"`
	Queue   []Notice          `json:"queue"`
}

func notificationPrefs(tx *bolt.Tx) NotificationPreferences {
	p := NotificationPreferences{Typing: true, Muted: map[string]bool{}}
	if raw := tx.Bucket([]byte("settings")).Get([]byte("notificationPreferences")); raw != nil {
		_ = json.Unmarshal(raw, &p)
	}
	if p.Muted == nil {
		p.Muted = map[string]bool{}
	}
	return p
}
func (d *DB) NotificationPreferences() (NotificationPreferences, error) {
	var p NotificationPreferences
	e := d.db.View(func(tx *bolt.Tx) error { p = notificationPrefs(tx); return nil })
	return p, e
}
func (d *DB) SaveNotificationPreferences(p NotificationPreferences) error {
	if len(p.Muted) > 512 {
		return errors.New("notification mute quota")
	}
	for k := range p.Muted {
		if len(k) > 256 || (!strings.HasPrefix(k, "g:") && !strings.HasPrefix(k, "d:")) {
			return errors.New("invalid mute context")
		}
	}
	return d.put("settings", "notificationPreferences", p)
}
func readInbox(tx *bolt.Tx) inboxState {
	s := inboxState{Cursors: map[string]uint64{}, Unread: map[string]int{}}
	if raw := tx.Bucket([]byte("settings")).Get([]byte("inboxState")); raw != nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}
func writeInbox(tx *bolt.Tx, s inboxState) error {
	raw, e := json.Marshal(s)
	if e != nil {
		return e
	}
	return tx.Bucket([]byte("settings")).Put([]byte("inboxState"), raw)
}
func (d *DB) Unread() (map[string]int, error) {
	var out map[string]int
	e := d.db.View(func(tx *bolt.Tx) error { out = readInbox(tx).Unread; return nil })
	return out, e
}
func (d *DB) MarkRead(context string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		s := readInbox(tx)
		n := s.Unread[context]
		if n == 0 {
			return nil
		}
		delete(s.Unread, context)
		if strings.HasPrefix(context, "g:") {
			parts := strings.Split(context, ":")
			if len(parts) == 3 {
				key := parts[0] + ":" + parts[1]
				s.Unread[key] -= n
				if s.Unread[key] <= 0 {
					delete(s.Unread, key)
				}
			}
		}
		q := s.Queue[:0]
		for _, v := range s.Queue {
			if v.Context != context {
				q = append(q, v)
			}
		}
		s.Queue = q
		return writeInbox(tx, s)
	})
}
func (d *DB) PollInbox(self string, active map[string]bool) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		initial := tx.Bucket([]byte("settings")).Get([]byte("inboxState")) == nil
		s := readInbox(tx)
		prefs := notificationPrefs(tx)
		seen := map[string]bool{}
		for _, v := range s.Seen {
			seen[v] = true
		}
		add := func(id, context, author, text, kind string) {
			if author == self || seen[id] {
				return
			}
			seen[id] = true
			s.Seen = append(s.Seen, id)
			if len(s.Seen) > 4096 {
				s.Seen = s.Seen[len(s.Seen)-4096:]
			}
			if active[context] {
				return
			}
			if s.Unread[context] < 9999 && (s.Unread[context] > 0 || len(s.Unread) < 512) {
				s.Unread[context]++
			}
			group := context
			if strings.HasPrefix(context, "g:") {
				parts := strings.Split(context, ":")
				if len(parts) == 3 {
					group = parts[0] + ":" + parts[1]
					if s.Unread[group] < 9999 && (s.Unread[group] > 0 || len(s.Unread) < 512) {
						s.Unread[group]++
					}
				}
			}
			if !prefs.Enabled || prefs.Muted[context] || prefs.Muted[group] {
				return
			}
			if strings.HasPrefix(context, "g:") {
				if !strings.Contains(text, "@{"+self+"}") {
					return
				}
				kind = "mention"
			}
			body := "Nova mensagem privada"
			if kind == "mention" {
				body = "Você recebeu uma menção"
			}
			if strings.Contains(text, "peetwopee://join/") || strings.Contains(text, "projetop2p-community:") || strings.Contains(text, "projetop2p:c2:") {
				kind = "invite"
				body = "Novo convite recebido"
			}
			if prefs.Preview {
				text = strings.Map(func(r rune) rune {
					if unicode.IsControl(r) {
						return ' '
					}
					return r
				}, text)
				body = string([]rune(text)[:min(len([]rune(text)), 160)])
			}
			notice := Notice{ID: id, Context: context, Kind: kind, Body: body}
			for i := range s.Queue {
				if s.Queue[i].Context == context {
					s.Queue[i] = notice
					return
				}
			}
			s.Queue = append(s.Queue, notice)
			if len(s.Queue) > 32 {
				s.Queue = s.Queue[len(s.Queue)-32:]
			}
		}
		scan := func(key string, b *bolt.Bucket, baselineNew bool, decode func([]byte)) {
			if b == nil {
				return
			}
			cursor := b.Cursor()
			last, _ := cursor.Last()
			if last == nil {
				return
			}
			_, exists := s.Cursors[key]
			if initial || !exists && baselineNew {
				s.Cursors[key] = binaryValue(last)
				return
			}
			after := s.Cursors[key]
			n := 0
			for k, v := cursor.Seek(seqkey(after + 1)); k != nil && n < 32; k, v = cursor.Next() {
				decode(v)
				s.Cursors[key] = binaryValue(k)
				n++
			}
		}
		scan("dm", tx.Bucket([]byte("messages")), false, func(raw []byte) {
			var m Message
			if json.Unmarshal(raw, &m) != nil {
				return
			}
			var contact Contact
			if json.Unmarshal(tx.Bucket([]byte("contacts")).Get([]byte(m.Peer)), &contact) != nil || contact.Unsolicited {
				return
			}
			add(m.From+":"+m.ID, "d:"+m.Peer, m.From, m.Text, "private")
		})
		if groups := tx.Bucket([]byte("communities")); groups != nil {
			_ = groups.ForEach(func(id, raw []byte) error {
				var r CommunityRecord
				if json.Unmarshal(raw, &r) != nil || r.Denied || r.State.Role(self) == "" {
					return nil
				}
				key := string(id)
				if parent := tx.Bucket([]byte("community_events")); parent != nil {
					scan("owner:"+key, parent.Bucket(id), true, func(raw []byte) {
						var e community.Commit
						if json.Unmarshal(raw, &e) == nil && e.Operation.Kind == community.Text {
							m := e.Operation
							if parent := tx.Bucket([]byte("community_peer_ids")); parent != nil {
								if ids := parent.Bucket(id); ids != nil {
									if sequence := ids.Get([]byte(m.ID)); sequence != nil && binaryValue(sequence) <= s.Cursors["peer:"+key] {
										return
									}
								}
							}
							add(key+":"+m.ID, "g:"+key+":"+m.Body.ChannelID, m.Author, m.Body.Text, "channel")
						}
					})
				}
				if parent := tx.Bucket([]byte("community_peer_text")); parent != nil {
					scan("peer:"+key, parent.Bucket(id), true, func(raw []byte) {
						var e community.PeerTextEntry
						if json.Unmarshal(raw, &e) == nil {
							m := e.Message.Operation
							if parent := tx.Bucket([]byte("community_opids")); parent != nil {
								if ids := parent.Bucket(id); ids != nil && ids.Get([]byte(m.ID)) != nil {
									return
								}
							}
							add(key+":"+m.ID, "g:"+key+":"+m.Body.ChannelID, m.Author, m.Body.Text, "channel")
						}
					})
				}
				return nil
			})
		}
		if parent := tx.Bucket([]byte("activity_log")); parent != nil {
			_ = parent.ForEach(func(context, _ []byte) error {
				scan("activity:"+string(context), parent.Bucket(context), false, func(raw []byte) {
					var entry chat.Entry
					if json.Unmarshal(raw, &entry) != nil || entry.Event.Kind != "reply" {
						return
					}
					e := entry.Event
					key := chat.Context(e.Room, self, e.Actor)
					if e.Room.Group != "" {
						key += ":" + e.Room.Channel
					}
					add("reply:"+e.ID, key, e.Actor, e.Text, "private")
				})
				return nil
			})
		}
		return writeInbox(tx, s)
	})
}
func (d *DB) DrainNotices() ([]Notice, error) {
	out := []Notice{}
	e := d.db.Update(func(tx *bolt.Tx) error {
		s := readInbox(tx)
		if len(s.Queue) == 0 {
			return nil
		}
		if notificationPrefs(tx).Enabled {
			out = s.Queue
		}
		s.Queue = nil
		return writeInbox(tx, s)
	})
	return out, e
}
