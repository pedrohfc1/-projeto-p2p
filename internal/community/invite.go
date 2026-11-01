package community

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/compactinvite"
	"projetop2p/internal/endpoint"
	"strings"
	"time"
)

const InvitePrefix = "projetop2p-community:"
const CompactInvitePrefix = "projetop2p-community:c2:"

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
	raw, e := encodeCompactInvite(i)
	if e != nil {
		return "", e
	}
	_, e = ParseInvite(raw, time.Now())
	return raw, e
}
func ParseInvite(raw string, now time.Time) (Invite, error) {
	if strings.HasPrefix(raw, CompactInvitePrefix) {
		i, e := decodeCompactInvite(raw)
		if e != nil {
			return Invite{}, e
		}
		b, _ := json.Marshal(i)
		return ParseInvite(InvitePrefix+base64.RawURLEncoding.EncodeToString(b), now)
	}
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

func LegacyInvite(raw string) (string, error) {
	i, e := ParseInvite(raw, time.Now())
	if e != nil {
		return "", e
	}
	b, _ := json.Marshal(i)
	return InvitePrefix + base64.RawURLEncoding.EncodeToString(b), nil
}
func encodeCompactInvite(i Invite) (string, error) {
	f := compactinvite.Fields{}
	key, e := compactinvite.RawKey(i.OwnerKey)
	if e != nil {
		return "", e
	}
	f.Add(1, []byte(i.Name))
	f.Add(2, key)
	f.Add(3, compactinvite.Number(uint64(i.Expires)))
	f.Add(4, []byte(i.Scope))
	f.Add(7, i.Signature)
	for n, s := range map[int]string{8: i.CommunityID, 9: i.Root, 11: i.Nonce} {
		b, e := hex.DecodeString(s)
		if e != nil {
			return "", e
		}
		f.Add(n, b)
	}
	f.Add(10, compactinvite.Number(i.Epoch))
	xs, e := compactinvite.Addresses(i.Addresses, i.Owner)
	if e != nil {
		return "", e
	}
	for _, a := range xs {
		f.Add(5, a)
	}
	return compactinvite.Encode(CompactInvitePrefix, f), nil
}
func decodeCompactInvite(raw string) (Invite, error) {
	i := Invite{Domain: "projetop2p/community-invite/1"}
	f, e := compactinvite.Decode(raw, CompactInvitePrefix, 16000)
	if e != nil {
		return i, e
	}
	if len(f[6]) > 0 || len(f[12]) > 0 {
		return i, errors.New("unexpected community field")
	}
	i.OwnerKey, i.Owner, e = compactinvite.Key(f.One(2))
	if e != nil {
		return i, e
	}
	expires, e := compactinvite.Uint(f.One(3))
	if e != nil || expires > 1<<63-1 {
		return i, errors.New("invalid expiration")
	}
	i.Expires = int64(expires)
	i.Epoch, e = compactinvite.Uint(f.One(10))
	if e != nil {
		return i, e
	}
	i.Name = string(f.One(1))
	i.Scope = string(f.One(4))
	i.Signature = f.One(7)
	i.CommunityID = hex.EncodeToString(f.One(8))
	i.Root = hex.EncodeToString(f.One(9))
	i.Nonce = hex.EncodeToString(f.One(11))
	i.Addresses, e = compactinvite.Restore(f[5], i.Owner)
	return i, e
}
