package firewall

import "testing"

func TestPortRules(t *testing.T) {
	for _, s := range []string{"4001", "4000-4010", "80,4001,443", "*", "Any"} {
		if !portMatch(s, 4001) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"4002", "1-1024", "invalid", ""} {
		if portMatch(s, 4001) {
			t.Fatal(s)
		}
	}
}
