package invite

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/transportaddr"
	"strings"
	"testing"
	"time"
)

func TestIntegratedInviteBindsRouteAndIdentity(t *testing.T) {
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	addr := "/ip6/" + transportaddr.LogicalIP(id.String()) + "/tcp/4001/p2p/" + id.String()
	raw, e := NewIntegrated(k, "QA", []string{addr}, "etc2abcdefgh")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(raw, CompactPrefix) {
		t.Fatal("not compact")
	}
	ticket, e := Parse(raw, time.Now())
	if e != nil || ticket.Route != "etc2abcdefgh" {
		t.Fatal(e)
	}
	ticket.Route = "etc2ijklmnop"
	b, _ := json.Marshal(ticket)
	if _, e = Parse(Prefix+base64.RawURLEncoding.EncodeToString(b), time.Now()); e == nil {
		t.Fatal("unsigned credential substitution")
	}
	if _, e = NewIntegrated(k, "QA", []string{"/ip6/fd00::1/tcp/4001/p2p/" + id.String()}, "etc2abcdefgh"); e == nil {
		t.Fatal("address not bound to expected identity")
	}
	if _, e = NewScoped(k, "QA", []string{addr}, "radmin"); e == nil {
		t.Fatal("cross-mode address accepted")
	}
}
