package localdb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Settings struct {
	Name        string   `json:"name"`
	Bootstrap   []string `json:"bootstrap"`
	Relays      []string `json:"relays"`
	Assisted    bool     `json:"assisted"`
	Mode        string   `json:"mode"`
	InterfaceID string   `json:"interfaceId"`
	Port        int      `json:"port"`
}
type Contact struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Addresses   []string `json:"addresses"`
	PublicKey   []byte   `json:"publicKey"`
	Scope       string   `json:"scope,omitempty"`
	Unsolicited bool     `json:"unsolicited,omitempty"`
}
type Message struct {
	ID    string `json:"id"`
	Peer  string `json:"peer"`
	From  string `json:"from"`
	Text  string `json:"text"`
	Time  int64  `json:"time"`
	State string `json:"state"`
}
type DB struct{ db *bolt.DB }

func Open(dir string) (*DB, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	d, e := bolt.Open(filepath.Join(dir, "chat.db"), 0600, &bolt.Options{Timeout: time.Second})
	if e != nil {
		return nil, errors.New("não foi possível abrir o banco; feche outra instância com o mesmo perfil: " + e.Error())
	}
	s := &DB{d}
	e = s.MigrateTo(CurrentSchema)
	if e != nil {
		d.Close()
		return nil, e
	}
	return s, nil
}
func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Settings() (Settings, error) {
	s := Settings{Name: "Meu perfil", Bootstrap: []string{}, Relays: []string{}, Assisted: true}
	e := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("settings")).Get([]byte("profile"))
		if b == nil {
			return nil
		}
		return json.Unmarshal(b, &s)
	})
	return s, e
}
func (d *DB) SaveSettings(s Settings) error { return d.put("settings", "profile", s) }
func (d *DB) put(bucket, key string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return d.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte(bucket)).Put([]byte(key), b) })
}
func (d *DB) Contact(c Contact) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("contacts"))
		if c.Unsolicited && b.Get([]byte(c.ID)) == nil {
			unknown := 0
			if e := b.ForEach(func(k, v []byte) error {
				var old Contact
				if e := json.Unmarshal(v, &old); e != nil {
					return e
				}
				if old.Unsolicited {
					unknown++
				}
				return nil
			}); e != nil {
				return e
			}
			if unknown >= 16 {
				return errors.New("new unsolicited contact limit reached; add a verified reciprocal invitation")
			}
		}
		if b.Stats().KeyN >= 256 && b.Get([]byte(c.ID)) == nil {
			return errors.New("limite de 256 contatos")
		}
		v, e := json.Marshal(c)
		if e != nil {
			return e
		}
		return b.Put([]byte(c.ID), v)
	})
}
func (d *DB) Contacts() ([]Contact, error) {
	out := []Contact{}
	e := d.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("contacts")).ForEach(func(k, v []byte) error {
			var c Contact
			if e := json.Unmarshal(v, &c); e != nil {
				return e
			}
			out = append(out, c)
			return nil
		})
	})
	return out, e
}
func (d *DB) AddMessage(m Message) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		ids := tx.Bucket([]byte("messageids"))
		idkey := []byte(m.From + ":" + m.ID)
		key := append([]byte(nil), ids.Get(idkey)...)
		if len(key) > 0 {
			var old Message
			if err := json.Unmarshal(b.Get(key), &old); err != nil {
				return err
			}
			if old.From != m.From || old.Peer != m.Peer || old.Text != m.Text || old.Time != m.Time {
				return errors.New("message ID already exists with different immutable content")
			}
		}
		if len(key) == 0 {
			seq, e := b.NextSequence()
			if e != nil {
				return e
			}
			key = make([]byte, 8)
			binary.BigEndian.PutUint64(key, seq)
			if e = ids.Put(idkey, key); e != nil {
				return e
			}
		}
		v, e := json.Marshal(m)
		if e != nil {
			return e
		}
		if e = b.Put(key, v); e != nil {
			return e
		}
		if b.Stats().KeyN > 10000 {
			c := b.Cursor()
			k, old := c.First()
			var previous Message
			if e = json.Unmarshal(old, &previous); e != nil {
				return e
			}
			if e = ids.Delete([]byte(previous.From + ":" + previous.ID)); e != nil {
				return e
			}
			return b.Delete(k)
		}
		return nil
	})
}
func (d *DB) Messages(id string) ([]Message, error) {
	out := []Message{}
	e := d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("messages")).Cursor()
		for k, v := c.Last(); k != nil && len(out) < 300; k, v = c.Prev() {
			var m Message
			if e := json.Unmarshal(v, &m); e != nil {
				return e
			}
			if m.Peer == id {
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
