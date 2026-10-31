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
