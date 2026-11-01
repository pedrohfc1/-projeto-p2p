package compactinvite

import (
	"encoding/base64"
	"google.golang.org/protobuf/encoding/protowire"
	"testing"
)

func TestMalformedEnvelope(t *testing.T) {
	for _, b := range [][]byte{{0xff}, {0x0a, 0xff}, {0x0a, 0, 0x0a, 0}, protowire.AppendTag(nil, 99, protowire.BytesType), {8, 1}} {
		if _, e := Decode("p:"+base64.RawURLEncoding.EncodeToString(b), "p:", 100); e == nil {
			t.Fatalf("accepted malformed %x", b)
		}
	}
}
func FuzzDecode(f *testing.F) {
	f.Add("p:AA")
	f.Fuzz(func(t *testing.T, s string) { _, _ = Decode(s, "p:", 24000) })
}
