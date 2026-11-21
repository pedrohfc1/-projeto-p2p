package community

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"projetop2p/internal/transportaddr"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSelfContainedSmallInvitationCompatibilityAndBounds(t *testing.T) {
	k := key(t)
	_, s := root(t, k)
	raw := ticket(t, k, s)
	i, e := ParseInvite(raw, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	v2, _ := encodeCompactV2(i)
	legacy, _ := LegacyInvite(raw)
	if len(raw) >= len(v2) {
		t.Fatalf("no reduction: small=%d c2=%d", len(raw), len(v2))
	}
	t.Logf("self-contained small=%d c2=%d legacy=%d", len(raw), len(v2), len(legacy))
	for _, v := range []string{raw, v2, legacy} {
		parsed, e := ParseInvite(v, time.Now())
		if e != nil || !bytes.Equal(parsed.payload(), i.payload()) || !bytes.Equal(parsed.Signature, i.Signature) {
			t.Fatal("signature payload changed", e)
		}
	}
	if _, e = ParseInvite(raw[:len(raw)-5], time.Now()); e == nil {
		t.Fatal("truncation accepted")
	}
	var bomb bytes.Buffer
	w, _ := flate.NewWriter(&bomb, flate.BestSpeed)
	w.Write(bytes.Repeat([]byte{0}, 12001))
	w.Close()
	if _, e = ParseInvite(SmallInvitePrefix+base64.RawURLEncoding.EncodeToString(append([]byte{1, 1}, bomb.Bytes()...)), time.Now()); e == nil {
		t.Fatal("oversized expansion accepted")
	}
	expanded, _ := expandSmallInvite(raw)
	payload, _ := base64.RawURLEncoding.DecodeString(expanded[len(CompactInvitePrefix):])
	payload[len(payload)-1] ^= 1
	if _, e = ParseInvite(SmallInvitePrefix+base64.RawURLEncoding.EncodeToString(append([]byte{1, 0}, payload...)), time.Now()); e == nil {
		t.Fatal("tampering accepted")
	}
}

func TestSmallIntegratedPolicyInvitationsPreserveCapabilities(t *testing.T) {
	k := key(t)
	_, s := root(t, k)
	secret := make([]byte, 160)
	rand.Read(secret)
	route := "etc2" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret))
	address := "/ip6/" + transportaddr.LogicalIP(s.Owner) + "/tcp/" + strconv.Itoa(transportaddr.ServicePort(s.Owner)) + "/p2p/" + s.Owner
	for _, options := range []InviteOptions{{Kind: "single", MaxUses: 1, LifetimeHours: 24}, {Kind: "multi", MaxUses: 25, LifetimeHours: 168}, {Kind: "permanent"}} {
		raw, e := NewPolicyInvite(k, s, "integrated", []string{address}, options, route)
		if e != nil {
			t.Fatal(e)
		}
		i, e := ParseInvite(raw, time.Now())
		if e != nil || i.Policy != options.Kind || i.MaxUses != options.MaxUses || i.Route != route || len(i.Addresses) != 1 || i.Addresses[0] != address {
			t.Fatal("credential/policy/address changed", e)
		}
		v2, _ := encodeCompactV2(i)
		legacy, _ := LegacyInvite(raw)
		if len(raw) >= len(v2) {
			t.Fatal("integrated invitation was not shortened")
		}
		t.Logf("%s small=%d c2=%d legacy=%d", options.Kind, len(raw), len(v2), len(legacy))
		old, e := ParseInvite(v2, time.Now())
		if e != nil || !bytes.Equal(old.payload(), i.payload()) {
			t.Fatal("old format signature differs", e)
		}
	}
}
