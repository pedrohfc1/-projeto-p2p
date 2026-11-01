package endpoint

import (
	ma "github.com/multiformats/go-multiaddr"
	"testing"
)

func TestVPNEndpointPolicy(t *testing.T) {
	for _, tc := range []struct {
		address                       string
		auto, radmin, hamachi, manual bool
	}{
		{"/ip4/26.10.20.30/tcp/4001", false, true, false, true},
		{"/ip4/25.10.20.30/tcp/4001", false, false, true, true},
		{"/ip4/169.254.10.2/tcp/4001", false, false, false, false},
		{"/ip6/fe80::1/tcp/4001", false, false, false, false},
		{"/ip4/192.168.1.2/tcp/4001", true, false, false, true},
		{"/ip4/10.5.0.2/tcp/4001", true, false, false, true},
		{"/ip4/26.10.20.30/tcp/0", false, false, false, false},
		{"/ip4/26.10.20.30/udp/4001/quic-v1", false, false, false, false},
		{"/dns4/example.org/tcp/4001", true, false, false, false},
	} {
		a := ma.StringCast(tc.address)
		for scope, want := range map[string]bool{"auto": tc.auto, "radmin": tc.radmin, "hamachi": tc.hamachi, "interface": tc.manual} {
			if got := Allowed(a, scope); got != want {
				t.Errorf("%s %s got %v want %v", scope, a, got, want)
			}
		}
	}
}
