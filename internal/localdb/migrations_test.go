package localdb

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/crypto"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"projetop2p/internal/community"
	"testing"
)

func TestLegacyMigrationAndSafeDowngrade(t *testing.T) {
	dir := t.TempDir()
	raw, e := bolt.Open(filepath.Join(dir, "chat.db"), 0600, nil)
	if e != nil {
		t.Fatal(e)
	}
	e = raw.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{"settings", "contacts", "messages", "messageids"} {
			if _, e := tx.CreateBucket([]byte(name)); e != nil {
				return e
			}
		}
		settings, _ := json.Marshal(Settings{Name: "Original", Mode: "radmin", InterfaceID: "stable-guid", Port: 4001})
		return tx.Bucket([]byte("settings")).Put([]byte("profile"), settings)
	})
	if e != nil {
		t.Fatal(e)
	}
	raw.Close()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	v, _ := d.SchemaVersion()
	s, _ := d.Settings()
	if v != CurrentSchema || s.Name != "Original" || s.Mode != "radmin" || s.InterfaceID != "stable-guid" {
		t.Fatal("migration lost settings", s, v)
	}
	if e = d.MigrateTo(1); e == nil {
		t.Fatal("v0.6 downgrade allowed")
	}
	if e = d.MigrateTo(2); e == nil {
		t.Fatal("policy downgrade allowed")
	}
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	op, _ := community.SignOperation(k, "", "", community.Create, community.Body{Name: "Group"})
	commit, st, e := community.MakeCommit(k, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.CreateOwnedCommunity(CommunityRecord{ID: st.ID, Owner: st.Owner, Root: st.Root, Name: st.Name}, commit); e != nil {
		t.Fatal(e)
	}
	if e = d.MigrateTo(1); e == nil {
		t.Fatal("destructive downgrade allowed")
	}
	r, e := d.Community(st.ID)
	if e != nil || r.State.Head != st.Head {
		t.Fatal("failed downgrade altered community")
	}
}
func TestFutureSchemaRejectedWithoutCreatingBuckets(t *testing.T) {
	dir := t.TempDir()
	b, e := bolt.Open(filepath.Join(dir, "chat.db"), 0600, nil)
	if e != nil {
		t.Fatal(e)
	}
	e = b.Update(func(tx *bolt.Tx) error {
		x, e := tx.CreateBucket([]byte("schema"))
		if e != nil {
			return e
		}
		v := make([]byte, 8)
		binary.BigEndian.PutUint64(v, 99)
		return x.Put([]byte("version"), v)
	})
	if e != nil {
		t.Fatal(e)
	}
	b.Close()
	if d, e := Open(dir); e == nil {
		d.Close()
		t.Fatal("future schema accepted")
	}
	b, e = bolt.Open(filepath.Join(dir, "chat.db"), 0600, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	_ = b.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("settings")) != nil {
			t.Fatal("rejected migration wrote legacy buckets")
		}
		return nil
	})
}

func TestSchemaTwoBackfillsVerifiedControlPrefixIndex(t *testing.T) {
	dir := t.TempDir()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	op, _ := community.SignOperation(key, "", "", community.Create, community.Body{Name: "Legacy schema two"})
	commit, state, e := community.MakeCommit(key, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.CreateOwnedCommunity(CommunityRecord{ID: state.ID, Owner: state.Owner, Root: state.Root, Name: state.Name}, commit); e != nil {
		t.Fatal(e)
	}
	if e = d.db.Update(func(tx *bolt.Tx) error {
		if e := tx.DeleteBucket([]byte("community_heads")); e != nil {
			return e
		}
		return tx.Bucket([]byte("schema")).Put([]byte("version"), seqkey(2))
	}); e != nil {
		t.Fatal(e)
	}
	d.Close()
	d, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	_ = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("community_heads")).Bucket([]byte(state.ID))
		if b == nil || binary.BigEndian.Uint64(b.Get([]byte(state.Head))) != 1 {
			t.Fatal("legacy prefix index absent")
		}
		return nil
	})
	record, e := d.Community(state.ID)
	if e != nil || record.Root != state.Root || record.State.Head != state.Head {
		t.Fatal("migration changed signed history", record, e)
	}
}
