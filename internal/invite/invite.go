package invite

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"projetop2p/internal/adapters"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const Prefix = "projetop2p:"

type Invite struct {
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
func newInvite(k crypto.PrivKey, name string, addresses, relays []string, scope string) (string, error) {
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
	i.Signature, e = k.Sign(i.payload())
	if e != nil {
		return "", e
	}
	b, e := json.Marshal(i)
	if e != nil {
		return "", e
	}
	s := Prefix + base64.RawURLEncoding.EncodeToString(b)
	_, e = Parse(s, time.Now())
	return s, e
}
func Parse(s string, now time.Time) (Invite, error) {
	var i Invite
	if len(s) > 24000 || !strings.HasPrefix(s, Prefix) {
		return i, errors.New("convite inválido ou muito grande")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, Prefix))
	if e != nil {
		return i, e
	}
	if e = json.Unmarshal(b, &i); e != nil {
		return i, e
	}
	if i.Version != 1 || len(i.Name) > 80 || !utf8.ValidString(i.Name) || i.Expires < now.Unix() || i.Expires > now.Add(8*24*time.Hour).Unix() || len(i.Addresses) > 16 || len(i.Relays) > 16 {
		return i, errors.New("convite expirado ou campos inválidos")
	}
	if i.Scope != "" && !adapters.IsVPN(i.Scope) && i.Scope != "interface" {
		return i, errors.New("escopo de convite inválido")
	}
	if i.Scope != "" && (len(i.Relays) > 0 || len(i.Addresses) == 0) {
		return i, errors.New("convite de interface não pode conter relays nem omitir endereço")
	}
	id, e := peer.Decode(i.ID)
	if e != nil {
		return i, e
	}
	pub, e := crypto.UnmarshalPublicKey(i.PublicKey)
	if e != nil {
		return i, e
	}
	pid, e := peer.IDFromPublicKey(pub)
	if e != nil || pid != id {
		return i, errors.New("identidade do convite não confere")
	}
	ok, e := pub.Verify(i.payload(), i.Signature)
	if e != nil || !ok {
		return i, errors.New("assinatura do convite inválida")
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
		if contains(i.Addresses, s) && p.ID != id {
			return i, errors.New("endereço pertence a outra identidade")
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
