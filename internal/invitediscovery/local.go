// Package invitediscovery is a LOCAL prototype. It does not publish or resolve DHT records.
package invitediscovery

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/community"
	"sync"
	"time"
)

type Record struct {
	Domain    string
	Owner     string
	Key       []byte
	Nonce     string
	Invite    string
	Expires   int64
	Signature []byte
}

func (r Record) payload() []byte { r.Signature = nil; b, _ := json.Marshal(r); return b }
func (r Record) Code() string {
	h := sha256.Sum256(r.payload())
	return "P2P-C1-" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(h[:20])
}
func New(key crypto.PrivKey, raw string, ttl time.Duration, now time.Time) (Record, error) {
	i, e := community.ParseInvite(raw, now)
	if e != nil {
		return Record{}, e
	}
	id, _ := peer.IDFromPrivateKey(key)
	if id.String() != i.Owner || ttl <= 0 || ttl > 8*24*time.Hour || now.Add(ttl).Unix() > i.Expires {
		return Record{}, errors.New("owner and bounded expiration required")
	}
	pub, _ := crypto.MarshalPublicKey(key.GetPublic())
	nonce := make([]byte, 16)
	if _, e = rand.Read(nonce); e != nil {
		return Record{}, e
	}
	r := Record{Domain: "projetop2p/invite-record/1", Owner: i.Owner, Key: pub, Nonce: hex.EncodeToString(nonce), Invite: raw, Expires: now.Add(ttl).Unix()}
	r.Signature, e = key.Sign(r.payload())
	return r, e
}
func (r Record) Verify(code string, now time.Time) (community.Invite, error) {
	i, e := community.ParseInvite(r.Invite, now)
	if e != nil {
		return i, e
	}
	if len(r.Invite) > 16000 || r.Domain != "projetop2p/invite-record/1" || len(r.Nonce) != 32 || code != r.Code() || r.Expires <= now.Unix() || r.Expires > i.Expires || r.Owner != i.Owner || len(r.Key) > 128 || len(r.Signature) != 64 {
		return i, errors.New("invalid record")
	}
	k, e := crypto.UnmarshalPublicKey(r.Key)
	if e != nil {
		return i, e
	}
	id, e := peer.IDFromPublicKey(k)
	if e != nil || id.String() != i.Owner || k.Type() != crypto.Ed25519 {
		return i, errors.New("record owner mismatch")
	}
	ok, e := k.Verify(r.payload(), r.Signature)
	if e != nil || !ok {
		return i, errors.New("invalid record signature")
	}
	return i, nil
}

type Revocation struct {
	Domain    string
	Code      string
	Owner     string
	Signature []byte
}

func (r Revocation) payload() []byte { r.Signature = nil; b, _ := json.Marshal(r); return b }
func Revoke(key crypto.PrivKey, r Record) (Revocation, error) {
	id, _ := peer.IDFromPrivateKey(key)
	if id.String() != r.Owner {
		return Revocation{}, errors.New("owner required")
	}
	v := Revocation{Domain: "projetop2p/invite-revocation/1", Code: r.Code(), Owner: r.Owner}
	var e error
	v.Signature, e = key.Sign(v.payload())
	return v, e
}

type Mirror struct {
	mu      sync.Mutex
	records map[string]Record
	revoked map[string]bool
}

func NewMirror() *Mirror { return &Mirror{records: map[string]Record{}, revoked: map[string]bool{}} }

// Put requires a state reconstructed from the local verified event log, NEVER a remote snapshot.
func (m *Mirror) Put(r Record, verifiedLocalState community.State, localPeer string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, e := r.Verify(r.Code(), now)
	if e != nil {
		return e
	}
	s := verifiedLocalState
	if s.ID != i.CommunityID || s.Owner != i.Owner || s.Root != i.Root || s.InviteEpoch != i.Epoch || s.Archived || s.Role(localPeer) == "" {
		return errors.New("unauthorized mirror or obsolete ticket")
	}
	if m.revoked[r.Code()] {
		return errors.New("revoked record")
	}
	if _, ok := m.records[r.Code()]; !ok && len(m.records) >= 64 {
		return errors.New("local mirror limit")
	}
	m.records[r.Code()] = r
	return nil
}
func (m *Mirror) Resolve(code string, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[code]
	if !ok || m.revoked[code] {
		return Record{}, errors.New("not found or revoked")
	}
	_, e := r.Verify(code, now)
	return r, e
}
func (m *Mirror) ApplyRevocation(v Revocation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[v.Code]
	if !ok || v.Domain != "projetop2p/invite-revocation/1" || v.Owner != r.Owner || len(v.Signature) != 64 {
		return errors.New("unknown revocation")
	}
	k, e := crypto.UnmarshalPublicKey(r.Key)
	if e != nil {
		return e
	}
	ok, e = k.Verify(v.payload(), v.Signature)
	if e != nil || !ok {
		return errors.New("invalid revocation")
	}
	m.revoked[v.Code] = true
	return nil
}
