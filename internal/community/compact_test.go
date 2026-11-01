package community

import (
	"encoding/base64"
	"encoding/json"
	"projetop2p/internal/compactinvite"
	"testing"
	"time"
)

func TestCompactCommunityCompatibilityAndSizes(t *testing.T) {
	k := key(t)
	_, s := root(t, k)
	raw := ticket(t, k, s)
	legacy, e := LegacyInvite(raw)
	if e != nil {
		t.Fatal(e)
	}
	a, e := ParseInvite(raw, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	b, e := ParseInvite(legacy, time.Now())
	if e != nil || string(a.payload()) != string(b.payload()) {
		t.Fatal("signature payload changed", e)
	}
	t.Logf("community legacy=%d compact=%d reduction=%.2f%%", len(legacy), len(raw), 100*(1-float64(len(raw))/float64(len(legacy))))
	if len(raw) >= len(legacy) {
		t.Fatal("no reduction")
	}
	f, _ := compactinvite.Decode(raw, CompactInvitePrefix, 16000)
	f[10] = [][]byte{compactinvite.Number(a.Epoch + 1)}
	if _, e = ParseInvite(compactinvite.Encode(CompactInvitePrefix, f), time.Now()); e == nil {
		t.Fatal("epoch tampering accepted")
	}
	if _, e = ParseInvite(raw, time.Now().Add(9*24*time.Hour)); e == nil {
		t.Fatal("expired accepted")
	}
	data, _ := json.Marshal(b)
	if _, e = ParseInvite(InvitePrefix+base64.RawURLEncoding.EncodeToString(data), time.Now()); e != nil {
		t.Fatal("old reader", e)
	}
}
