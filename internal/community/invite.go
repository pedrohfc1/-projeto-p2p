package community

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"io"
	"projetop2p/internal/compactinvite"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/transportaddr"
	"strconv"
	"strings"
	"time"
)

const InvitePrefix = "projetop2p-community:"
const SmallInvitePrefix = "peetwopee://join/"
const CompactInvitePrefix = "projetop2p-community:c2:"

type InviteOptions struct {
	Kind          string `json:"kind"`
	MaxUses       uint32 `json:"maxUses"`
	LifetimeHours int    `json:"lifetimeHours"`
}
type Invite struct {
	Policy      string   `json:"policy,omitempty"`
	MaxUses     uint32   `json:"maxUses,omitempty"`
	Route       string   `json:"route,omitempty"`
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
func NewInvite(key crypto.PrivKey, s State, scope string, addresses []string, routes ...string) (string, error) {
	return newInvite(key, s, scope, addresses, InviteOptions{}, routes...)
}
func NewPolicyInvite(key crypto.PrivKey, s State, scope string, addresses []string, options InviteOptions, routes ...string) (string, error) {
	if err := options.Validate(); err != nil {
		return "", err
	}
	return newInvite(key, s, scope, addresses, options, routes...)
}
func (o InviteOptions) Validate() error {
	if o.Kind != "single" && o.Kind != "multi" && o.Kind != "permanent" {
		return errors.New("invalid invitation policy")
	}
	if o.Kind == "single" && o.MaxUses != 1 || o.Kind == "multi" && (o.MaxUses < 2 || o.MaxUses > 100) || o.Kind == "permanent" && o.MaxUses != 0 {
		return errors.New("invalid invitation use limit")
	}
	switch o.LifetimeHours {
	case 0, 1, 24, 168, 720:
	default:
		return errors.New("invalid invitation lifetime")
	}
	return nil
}
func newInvite(key crypto.PrivKey, s State, scope string, addresses []string, options InviteOptions, routes ...string) (string, error) {
	id, _ := peer.IDFromPrivateKey(key)
	if id.String() != s.Owner || s.Archived {
		return "", errors.New("owner signing key required")
	}
	pub, _ := crypto.MarshalPublicKey(key.GetPublic())
	if len(addresses) > 8 {
		addresses = addresses[:8]
	}
	i := Invite{Domain: "projetop2p/community-invite/1", CommunityID: s.ID, Name: s.Name, Owner: s.Owner, OwnerKey: pub, Root: s.Root, Epoch: s.InviteEpoch, Nonce: RandomID(), Expires: time.Now().Add(24 * time.Hour).Unix(), Scope: scope, Addresses: addresses}
	if options.Kind != "" {
		i.Policy = options.Kind
		i.MaxUses = options.MaxUses
		i.Expires = 0
		if options.LifetimeHours > 0 {
			i.Expires = time.Now().Add(time.Duration(options.LifetimeHours) * time.Hour).Unix()
		}
	}
	if len(routes) > 0 {
		i.Route = routes[0]
	}
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
	if strings.HasPrefix(raw, SmallInvitePrefix) {
		expanded, err := expandSmallInvite(raw)
		if err != nil {
			return Invite{}, err
		}
		return ParseInvite(expanded, now)
	}
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
	if i.Domain != "projetop2p/community-invite/1" || !ValidID(i.CommunityID) || !ValidID(i.Root) || !NameOK(i.Name) || len(i.Nonce) != 32 || i.Epoch == 0 || len(i.Addresses) == 0 || len(i.Addresses) > 8 {
		return i, errors.New("expired or invalid community invitation")
	}
	if i.Policy == "" {
		if i.MaxUses != 0 || i.Expires == 0 || i.Expires > now.Add(8*24*time.Hour).Unix() {
			return i, errors.New("invalid legacy invitation")
		}
	} else {
		if e := (InviteOptions{Kind: i.Policy, MaxUses: i.MaxUses}).Validate(); e != nil {
			return i, e
		}
		if i.Expires > now.Add(31*24*time.Hour).Unix() {
			return i, errors.New("invalid invitation expiration")
		}
	}
	if i.Expires != 0 && i.Expires < now.Unix() {
		return i, errors.New("invite_expired: convite expirado")
	}
	if i.Scope != "auto" && i.Scope != "radmin" && i.Scope != "hamachi" && i.Scope != "interface" && i.Scope != "integrated" {
		return i, errors.New("invalid invitation network scope")
	}
	if (i.Scope == "integrated" && !transportaddr.ValidToken(i.Route)) || (i.Scope != "integrated" && i.Route != "") {
		return i, errors.New("invalid integrated community credential")
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
		if i.Scope == "integrated" {
			ip, _ := a.ValueForProtocol(ma.P_IP6)
			if ip != transportaddr.LogicalIP(i.Owner) {
				return i, errors.New("logical endpoint does not match owner")
			}
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
func encodeCompactV2(i Invite) (string, error) {
	f := compactinvite.Fields{}
	key, e := compactinvite.RawKey(i.OwnerKey)
	if e != nil {
		return "", e
	}
	if i.Policy != "" {
		policy, _ := json.Marshal(InviteOptions{Kind: i.Policy, MaxUses: i.MaxUses})
		f.Add(14, policy)
	}
	f.Add(1, []byte(i.Name))
	f.Add(2, key)
	f.Add(3, compactinvite.Number(uint64(i.Expires)))
	f.Add(4, []byte(i.Scope))
	f.Add(7, i.Signature)
	if i.Route != "" {
		f.Add(13, []byte(i.Route))
	}
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
	i.Route = string(f.One(13))
	if len(f[14]) > 0 {
		var policy InviteOptions
		if len(f.One(14)) > 128 || json.Unmarshal(f.One(14), &policy) != nil {
			return i, errors.New("invalid compact policy")
		}
		i.Policy = policy.Kind
		i.MaxUses = policy.MaxUses
	}
	i.CommunityID = hex.EncodeToString(f.One(8))
	i.Root = hex.EncodeToString(f.One(9))
	i.Nonce = hex.EncodeToString(f.One(11))
	i.Addresses, e = compactinvite.Restore(f[5], i.Owner)
	return i, e
}

// Small invitations carry the complete signed c2 payload, never an opaque server code.
// Compression only changes the envelope: domain, signature and authorization stay identical.
func encodeCompactInvite(i Invite) (string, error) {
	old, err := encodeCompactV2(i)
	if err != nil {
		return "", err
	}
	fields, err := compactinvite.Decode(old, CompactInvitePrefix, 16000)
	if err != nil {
		return "", err
	}
	scopes := []string{"auto", "integrated", "radmin", "hamachi", "interface"}
	for n, scope := range scopes {
		if i.Scope == scope {
			fields[4] = [][]byte{{byte(n)}}
		}
	}
	if i.Policy != "" {
		policies := map[string]byte{"single": 1, "multi": 2, "permanent": 3}
		fields[14] = [][]byte{{policies[i.Policy], byte(i.MaxUses)}}
	}
	if i.Scope == "integrated" && len(i.Addresses) == 1 && i.Addresses[0] == "/ip6/"+transportaddr.LogicalIP(i.Owner)+"/tcp/"+strconv.Itoa(transportaddr.ServicePort(i.Owner))+"/p2p/"+i.Owner {
		delete(fields, 5)
		fields[6] = [][]byte{{1}}
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(compactinvite.Encode("", fields))
	if err != nil {
		return "", err
	}
	var compressed bytes.Buffer
	writer, _ := flate.NewWriter(&compressed, flate.BestCompression)
	if _, err = writer.Write(raw); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	envelope := append([]byte{2, 0}, raw...)
	if compressed.Len() < len(raw) {
		envelope = append([]byte{2, 1}, compressed.Bytes()...)
	}
	return SmallInvitePrefix + base64.RawURLEncoding.EncodeToString(envelope), nil
}
func expandSmallInvite(raw string) (string, error) {
	if len(raw) > 16000 {
		return "", errors.New("invitation too large")
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(raw, SmallInvitePrefix))
	if err != nil || len(b) < 3 || (b[0] != 1 && b[0] != 2) {
		return "", errors.New("invalid self-contained invitation version")
	}
	var payload []byte
	switch b[1] {
	case 0:
		payload = b[2:]
	case 1:
		source := bytes.NewReader(b[2:])
		reader := flate.NewReader(source)
		payload, err = io.ReadAll(io.LimitReader(reader, 12001))
		closeErr := reader.Close()
		if err != nil || closeErr != nil || source.Len() != 0 {
			return "", errors.New("invalid compressed invitation")
		}
	default:
		return "", errors.New("unsupported invitation encoding")
	}
	if len(payload) > 12000 {
		return "", errors.New("expanded invitation too large")
	}
	if b[0] == 2 {
		fields, e := compactinvite.Decode(base64.RawURLEncoding.EncodeToString(payload), "", 16000)
		if e != nil {
			return "", e
		}
		scopes := []string{"auto", "integrated", "radmin", "hamachi", "interface"}
		scope := fields.One(4)
		if len(scope) != 1 || int(scope[0]) >= len(scopes) {
			return "", errors.New("invalid compact scope")
		}
		fields[4] = [][]byte{[]byte(scopes[scope[0]])}
		if len(fields[14]) > 0 {
			policy := fields.One(14)
			if len(policy) != 2 || policy[0] < 1 || policy[0] > 3 {
				return "", errors.New("invalid compact policy")
			}
			names := []string{"", "single", "multi", "permanent"}
			raw, _ := json.Marshal(InviteOptions{Kind: names[policy[0]], MaxUses: uint32(policy[1])})
			fields[14] = [][]byte{raw}
		}
		if len(fields[6]) > 0 {
			if !bytes.Equal(fields.One(6), []byte{1}) || len(fields[5]) != 0 || string(fields.One(4)) != "integrated" {
				return "", errors.New("invalid derived endpoint")
			}
			_, id, e := compactinvite.Key(fields.One(2))
			if e != nil {
				return "", e
			}
			addresses, e := compactinvite.Addresses([]string{"/ip6/" + transportaddr.LogicalIP(id) + "/tcp/" + strconv.Itoa(transportaddr.ServicePort(id)) + "/p2p/" + id}, id)
			if e != nil {
				return "", e
			}
			fields[5] = addresses
			delete(fields, 6)
		}
		return compactinvite.Encode(CompactInvitePrefix, fields), nil
	}
	return CompactInvitePrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}
