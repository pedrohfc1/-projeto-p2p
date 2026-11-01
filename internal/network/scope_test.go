package network

import (
	ma "github.com/multiformats/go-multiaddr"
	"testing"
)

func TestNoVirtualAddressesInNativeDiscovery(t *testing.T) {
	in := []ma.Multiaddr{ma.StringCast("/ip4/25.10.20.30/tcp/4001"), ma.StringCast("/ip4/26.10.20.30/tcp/4001"), ma.StringCast("/ip4/127.0.0.1/tcp/4001"), ma.StringCast("/dns4/relay.example.com/tcp/4001")}
	if len(withoutVPNAddresses(in)) != 2 {
		t.Fatal("virtual address advertised")
	}
	out := publicAddresses(in)
	if len(out) != 1 || out[0].String() != in[3].String() {
		t.Fatal("private/virtual address leaked into public discovery", out)
	}
}

func TestRadminGaterAndFixedAnnouncePort(t *testing.T) {
	g := interfaceGate{ip: "26.10.20.30", scope: "radmin"}
	for _, raw := range []string{"/ip4/169.254.1.2/tcp/4001", "/ip4/192.168.1.2/tcp/4001", "/ip4/10.5.0.2/tcp/4001", "/ip4/25.1.2.3/tcp/4001"} {
		if g.InterceptAddrDial("", ma.StringCast(raw)) {
			t.Fatal("gater allowed wrong interface", raw)
		}
	}
	if !g.InterceptAddrDial("", ma.StringCast("/ip4/26.1.2.3/tcp/4001")) {
		t.Fatal("Radmin destination blocked")
	}
	as := []ma.Multiaddr{ma.StringCast("/ip4/26.10.20.30/tcp/4001"), ma.StringCast("/ip4/26.10.20.30/tcp/58999"), ma.StringCast("/ip4/25.1.2.3/tcp/4001"), ma.StringCast("/ip4/169.254.1.2/tcp/4001")}
	out := scopedAddresses(as, g.ip, []string{"/ip4/26.10.20.30/tcp/4001"})
	if len(out) != 1 || out[0].String() != as[0].String() {
		t.Fatal("observed ephemeral port or wrong IP advertised", out)
	}
	if len(withoutVPNAddresses(as)) != 0 {
		t.Fatal("APIPA/VPN leaked into native announcements")
	}
}
