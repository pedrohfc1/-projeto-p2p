package community

import (
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"strings"
	"time"
)

// PeerText is an author-signed journal entry, separate from owner-ordered controls.
// Epoch/root/base bind it to the verified authorization state; it is not an owner receipt.
type PeerText struct {
	Domain    string    `json:"domain"`
	Root      string    `json:"root"`
	Epoch     uint64    `json:"epoch"`
	Operation Operation `json:"operation"`
	Signature []byte    `json:"signature"`
}
type PeerTextEntry struct {
	Sequence uint64   `json:"sequence"`
	Message  PeerText `json:"message"`
}

func (m PeerText) payload() []byte { m.Signature = nil; b, _ := json.Marshal(m); return b }
func NewPeerText(key crypto.PrivKey, s State, body Body) (PeerText, error) {
	o, e := SignOperation(key, s.ID, s.Head, Text, body)
	if e != nil {
		return PeerText{}, e
	}
	m := PeerText{Domain: "projetop2p/community-peer-text/1", Root: s.Root, Epoch: s.InviteEpoch, Operation: o}
	m.Signature, e = key.Sign(m.payload())
	return m, e
}
func VerifyPeerText(m PeerText) error {
	if m.Domain != "projetop2p/community-peer-text/1" || !ValidID(m.Root) || m.Epoch == 0 || m.Operation.Kind != Text || strings.TrimSpace(m.Operation.Body.Text) == "" {
		return errors.New("invalid peer text")
	}
	if e := VerifyOperation(m.Operation); e != nil {
		return e
	}
	return signedBy(m.Operation.Author, m.Operation.PublicKey, m.payload(), m.Signature)
}
func AcceptPeerText(m PeerText, s State, now time.Time) error {
	if e := VerifyPeerText(m); e != nil {
		return e
	}
	o := m.Operation
	ch, ok := s.Channels[o.Body.ChannelID]
	if s.ID != o.CommunityID || s.Root != m.Root || s.InviteEpoch != m.Epoch || !ValidID(o.Base) || s.Role(o.Author) == "" || s.Archived || !ok || ch.Deleted || ch.Kind != "text" {
		return errors.New("sync_failure: peer text requires current verified membership and matching control head")
	}
	if o.Issued > now.Add(5*time.Minute).Unix() || o.Issued < now.Add(-30*24*time.Hour).Unix() {
		return errors.New("peer text timestamp outside window")
	}
	return nil
}
