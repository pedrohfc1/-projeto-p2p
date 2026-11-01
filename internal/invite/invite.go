package invite

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"projetop2p/internal/adapters"
	"projetop2p/internal/compactinvite"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/transportaddr"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const Prefix = "projetop2p:"
const CompactPrefix = "projetop2p:c2:"

type Invite struct {
	Route     string   `json:"route,omitempty"`
	Version   int      `json:"version"`
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Relays    []string `json:"relays"`
	Expires   int64    `json:"expires"`
	PublicKey []byte   `json:"publicKey"`
	Signature []byte   `json:"signature"`
	Scope     string   `json:"scope,omitempty"`
}

func (i Invite) payload() []byte { i.Signature = nil; b, _ := json.Marshal(i); return b }
func New(k crypto.PrivKey, name string, addresses, relays []string) (string, error) {
	return newInvite(k, name, addresses, relays, "")
}
func NewScoped(k crypto.PrivKey, name string, addresses []string, scope string) (string, error) {
	return newInvite(k, name, addresses, nil, scope)
}
func newInvite(k crypto.PrivKey, name string, addresses, relays []string, scope string, routes ...string) (string, error) {
	id, e := peer.IDFromPrivateKey(k)
	if e != nil {
		return "", e
	}
	pub, e := crypto.MarshalPublicKey(k.GetPublic())
	if e != nil {
		return "", e
	}
	i := Invite{Version: 1, ID: id.String(), Name: name, Addresses: addresses, Relays: relays, Expires: time.Now().Add(7 * 24 * time.Hour).Unix(), PublicKey: pub}
	i.Scope = scope
	if len(routes) > 0 {
		i.Route = routes[0]
	}
	i.Signature, e = k.Sign(i.payload())
	if e != nil {
		return "", e
	}
	s, e := encodeCompact(i)
	if e != nil {
		return "", e
	}
	_, e = Parse(s, time.Now())
	return s, e
}
func Parse(s string, now time.Time) (Invite, error) {
	if strings.HasPrefix(s, CompactPrefix) {
		i, e := decodeCompact(s)
		if e != nil {
			return Invite{}, e
		}
		b, _ := json.Marshal(i)
		return Parse(Prefix+base64.RawURLEncoding.EncodeToString(b), now)
	}
	var i Invite
	if len(s) > 24000 || !strings.HasPrefix(s, Prefix) {
		return i, errors.New("convite invÃ¡lido ou muito grande")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, Prefix))
	if e != nil {
		return i, e
	}
	if e = json.Unmarshal(b, &i); e != nil {
		return i, e
	}
	if i.Version != 1 || len(i.Name) > 80 || !utf8.ValidString(i.Name) || i.Expires < now.Unix() || i.Expires > now.Add(8*24*time.Hour).Unix() || len(i.Addresses) > 16 || len(i.Relays) > 16 {
		return i, errors.New("convite expirado ou campos invÃ¡lidos")
	}
	if i.Scope != "" && !adapters.IsVPN(i.Scope) && i.Scope != "interface" && i.Scope != "integrated" {
		return i, errors.New("escopo de convite invÃ¡lido")
	}
	if i.Scope != "" && (len(i.Relays) > 0 || len(i.Addresses) == 0) {
		return i, errors.New("convite de interface nÃ£o pode conter relays nem omitir endereÃ§o")
	}
	if (i.Scope == "integrated" && !transportaddr.ValidToken(i.Route)) || (i.Scope != "integrated" && i.Route != "") {
		return i, errors.New("invalid integrated credential")
	}
	id, e := peer.Decode(i.ID)
	if e != nil {
		return i, e
	}
	if len(i.PublicKey) > 128 || len(i.Signature) != 64 {
		return i, errors.New("invalid signing key or signature size")
	}
	pub, e := crypto.UnmarshalPublicKey(i.PublicKey)
	if e != nil {
		return i, e
	}
	if pub.Type() != crypto.Ed25519 {
		return i, errors.New("Ed25519 required")
	}
	pid, e := peer.IDFromPublicKey(pub)
	if e != nil || pid != id {
		return i, errors.New("identidade do convite nÃ£o confere")
	}
	ok, e := pub.Verify(i.payload(), i.Signature)
	if e != nil || !ok {
		return i, errors.New("assinatura do convite invÃ¡lida")
	}
	for _, s := range append(append([]string{}, i.Addresses...), i.Relays...) {
		a, e := ma.NewMultiaddr(s)
		if e != nil {
			return i, e
		}
		p, e := peer.AddrInfoFromP2pAddr(a)
		if e != nil {
			return i, e
		}
		if contains(i.Addresses, s) && i.Scope != "" && !endpoint.Allowed(a, i.Scope) {
			return i, errors.New("convite de VPN/interface contÃ©m endereÃ§o incompatÃ­vel, link-local ou porta TCP invÃ¡lida")
		}
		if i.Scope == "integrated" {
			ip, _ := a.ValueForProtocol(ma.P_IP6)
			if ip != transportaddr.LogicalIP(id.String()) {
				return i, errors.New("logical address does not match identity")
			}
		}
		if contains(i.Addresses, s) && p.ID != id {
			return i, errors.New("endereÃ§o pertence a outra identidade")
		}
	}
	return i, nil
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// Legacy preserves the original signed payload for peers older than 0.5.1.
func Legacy(raw string) (string, error) {
	i, e := Parse(raw, time.Now())
	if e != nil {
		return "", e
	}
	b, _ := json.Marshal(i)
	return Prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
func encodeCompact(i Invite) (string, error) {
	key, e := compactinvite.RawKey(i.PublicKey)
	if e != nil {
		return "", e
	}
	f := compactinvite.Fields{}
	f.Add(1, []byte(i.Name))
	f.Add(2, key)
	f.Add(3, compactinvite.Number(uint64(i.Expires)))
	f.Add(4, []byte(i.Scope))
	f.Add(7, i.Signature)
	if i.Route != "" {
		f.Add(13, []byte(i.Route))
	}
	xs, e := compactinvite.Addresses(i.Addresses, i.ID)
	if e != nil {
		return "", e
	}
	for _, a := range xs {
		f.Add(5, a)
	}
	for _, a := range i.Relays {
		m, e := ma.NewMultiaddr(a)
		if e != nil {
			return "", e
		}
		f.Add(6, m.Bytes())
	}
	var flags uint64
	if i.Relays == nil {
		flags |= 1
	}
	if i.Addresses == nil {
		flags |= 2
	}
	f.Add(12, compactinvite.Number(flags))
	return compactinvite.Encode(CompactPrefix, f), nil
}
func decodeCompact(raw string) (Invite, error) {
	i := Invite{Version: 1}
	f, e := compactinvite.Decode(raw, CompactPrefix, 24000)
	if e != nil {
		return i, e
	}
	if len(f[14]) > 0 {
		return i, errors.New("unexpected contact policy field")
	}
	for n := 8; n <= 11; n++ {
		if len(f[n]) > 0 {
			return i, errors.New("unexpected contact field")
		}
	}
	i.PublicKey, i.ID, e = compactinvite.Key(f.One(2))
	if e != nil {
		return i, e
	}
	expires, e := compactinvite.Uint(f.One(3))
	if e != nil || expires > 1<<63-1 {
		return i, errors.New("invalid expiration")
	}
	i.Expires = int64(expires)
	i.Name = string(f.One(1))
	i.Scope = string(f.One(4))
	i.Signature = f.One(7)
	i.Route = string(f.One(13))
	i.Addresses, e = compactinvite.Restore(f[5], i.ID)
	if e != nil {
		return i, e
	}
	i.Relays = []string{}
	for _, b := range f[6] {
		a, e := ma.NewMultiaddrBytes(b)
		if e != nil {
			return i, e
		}
		i.Relays = append(i.Relays, a.String())
	}
	flags, e := compactinvite.Uint(f.One(12))
	if e != nil || flags > 3 {
		return i, errors.New("invalid flags")
	}
	if flags&1 != 0 {
		if len(i.Relays) > 0 {
			return i, errors.New("invalid relays")
		}
		i.Relays = nil
	}
	if flags&2 != 0 {
		if len(i.Addresses) > 0 {
			return i, errors.New("invalid addresses")
		}
		i.Addresses = nil
	}
	return i, nil
}

func NewIntegrated(k crypto.PrivKey, name string, addresses []string, route string) (string, error) {
	return newInvite(k, name, addresses, nil, "integrated", route)
}
