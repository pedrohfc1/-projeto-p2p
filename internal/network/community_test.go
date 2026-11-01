package network

import (
	"projetop2p/internal/community"
	"strings"
	"testing"
)

func TestCommunityFrameBoundsAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"method":"sync","communityId":"x","admin":true}` + "\n", `{"method":"sync"} {}` + "\n", strings.Repeat("x", community.MaxRequest+1) + "\n"} {
		var req community.Request
		if e := readCommunityFrame(strings.NewReader(raw), community.MaxRequest, &req); e == nil {
			t.Fatal("invalid frame accepted")
		}
	}
	var req community.Request
	if e := readCommunityFrame(strings.NewReader(`{"method":"sync","communityId":"x","after":0,"head":""}`+"\n"), community.MaxRequest, &req); e != nil {
		t.Fatal(e)
	}
}
