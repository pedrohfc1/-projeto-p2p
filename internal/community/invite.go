package community

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/endpoint"
	"strings"
	"time"
)

const InvitePrefix = "projetop2p-community:"

type Invite struct {
	Domain      string   `json:"domain"`
	CommunityID string   `json:"communityId"`
	Name        string   `json:"name"`
	Owner       string   `json:"owner"`
	OwnerKey    []byte   `json:"ownerKey"`
	Root        string   `json:"root"`
	Epoch       uint64   `json:"epoch"`
	Nonce       string   `json:"nonce"`
	Expires     int64    `json:"expires"`
	Scope       string   `json:"scope"`
	Addresses   []string `json:"addresses"`
	Signature   []byte   `json:"signature"`
}

func (i Invite) payload() []byte { i.Signature = nil; b, _ := json.Marshal(i); return b }
func NewInvite(key crypto.PrivKey, s State, scope string, addresses []string) (string, error) {
	id, _ := peer.IDFromPrivateKey(key)
	if id.String() != s.Owner || s.Archived {
		return "", errors.New("owner signing key required")
	}
	pub, _ := crypto.MarshalPublicKey(key.GetPublic())
	if len(addresses) > 8 {
		addresses = addresses[:8]
	}
	i := Invite{Domain: "projetop2p/community-invite/1", CommunityID: s.ID, Name: s.Name, Owner: s.Owner, OwnerKey: pub, Root: s.Root, Epoch: s.InviteEpoch, Nonce: RandomID(), Expires: time.Now().Add(24 * time.Hour).Unix(), Scope: scope, Addresses: addresses}
	var e error
	i.Signature, e = key.Sign(i.payload())
	if e != nil {
		return "", e
	}
	b, _ := json.Marshal(i)
	raw := InvitePrefix + base64.RawURLEncoding.EncodeToString(b)
	_, e = ParseInvite(raw, time.Now())
	return raw, e
}
func ParseInvite(raw string, now time.Time) (Invite, error) {
	var i Invite
	if len(raw) > 16000 || !strings.HasPrefix(raw, InvitePrefix) {
		return i, errors.New("invalid community invitation")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(raw, InvitePrefix))
	if e != nil {
		return i, e
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&i); e != nil {
		return i, e
	}
	if i.Domain != "projetop2p/community-invite/1" || !ValidID(i.CommunityID) || !ValidID(i.Root) || !NameOK(i.Name) || len(i.Nonce) != 32 || i.Epoch == 0 || i.Expires < now.Unix() || i.Expires > now.Add(8*24*time.Hour).Unix() || len(i.Addresses) == 0 || len(i.Addresses) > 8 {
		return i, errors.New("expired or invalid community invitation")
	}
	if i.Scope != "auto" && i.Scope != "radmin" && i.Scope != "hamachi" && i.Scope != "interface" {
		return i, errors.New("invalid invitation network scope")
	}
	if e = signedBy(i.Owner, i.OwnerKey, i.payload(), i.Signature); e != nil {
		return i, e
	}
	for _, raw := range i.Addresses {
		if len(raw) > 1024 {
			return i, errors.New("address too long")
		}
		a, e := ma.NewMultiaddr(raw)
		if e != nil || !endpoint.Allowed(a, i.Scope) {
			return i, errors.New("address incompatible with invitation scope")
		}
		info, e := peer.AddrInfoFromP2pAddr(a)
		if e != nil || info.ID.String() != i.Owner {
			return i, errors.New("address does not belong to owner")
		}
	}
	return i, nil
}
