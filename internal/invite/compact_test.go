package invite

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/compactinvite"
	"strings"
	"testing"
	"time"
)

func TestCompactContactCompatibilityAndSizes(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	for _, scope := range []string{"radmin", "hamachi", ""} {
		ip := "192.168.1.12"
		if scope == "radmin" {
			ip = "26.10.20.30"
		}
		if scope == "hamachi" {
			ip = "25.10.20.30"
		}
		raw, e := NewScoped(k, "Pessoa", []string{"/ip4/" + ip + "/tcp/4001/p2p/" + id.String()}, scope)
		if e != nil {
			t.Fatal(e)
		}
		legacy, e := Legacy(raw)
		if e != nil {
			t.Fatal(e)
		}
		a, e := Parse(raw, time.Now())
		if e != nil {
			t.Fatal(e)
		}
		b, e := Parse(legacy, time.Now())
		if e != nil || string(a.payload()) != string(b.payload()) {
			t.Fatal("legacy signature payload changed", e)
		}
		if len(raw) >= len(legacy) {
			t.Fatal("no size reduction")
		}
		t.Logf("scope=%q legacy=%d compact=%d reduction=%.2f%%", scope, len(legacy), len(raw), 100*(1-float64(len(raw))/float64(len(legacy))))
		f, e := compactinvite.Decode(raw, CompactPrefix, 24000)
		if e != nil {
			t.Fatal(e)
		}
		f[1] = [][]byte{[]byte("Atacante")}
		if _, e = Parse(compactinvite.Encode(CompactPrefix, f), time.Now()); e == nil {
			t.Fatal("tampering accepted")
		}
		if _, e = Parse(raw, time.Now().Add(9*24*time.Hour)); e == nil {
			t.Fatal("expired accepted")
		}
	}
	// Independently construct the old JSON format rather than only roundtripping the new writer.
	pub, _ := crypto.MarshalPublicKey(k.GetPublic())
	i := Invite{Version: 1, ID: id.String(), Name: "Antigo", Expires: time.Now().Add(time.Hour).Unix(), PublicKey: pub, Addresses: []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + id.String()}, Relays: []string{}}
	i.Signature, _ = k.Sign(i.payload())
	old, _ := json.Marshal(i)
	if _, e := Parse(Prefix+base64.RawURLEncoding.EncodeToString(old), time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e := Parse(CompactPrefix+strings.Repeat("A", 24001), time.Now()); e == nil {
		t.Fatal("oversize accepted")
	}
}
func TestCompactRelayAndNilLists(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	relay, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	rid, _ := peer.IDFromPrivateKey(relay)
	for _, relays := range [][]string{nil, {}, {"/dns4/relay.example.org/tcp/4001/p2p/" + rid.String()}} {
		raw, e := New(k, "Nome", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + id.String(), "/dns4/relay.example.org/tcp/4001/p2p/" + rid.String() + "/p2p-circuit/p2p/" + id.String()}, relays)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = Parse(raw, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
}
