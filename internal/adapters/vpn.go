package adapters

import "strings"

// VPNDefinition is shared by adapter classification, selection, invitation
// policy and public-address filtering. Adding a provider does not change chat.
type VPNDefinition struct {
	ID           string
	Name         string
	IPv4Prefix   string
	DriverTokens []string
}

var vpnDefinitions = []VPNDefinition{
	{ID: "hamachi", Name: "Hamachi", IPv4Prefix: "25.", DriverTokens: []string{"hamachi", "hamdrv"}},
	{ID: "radmin", Name: "Radmin VPN", IPv4Prefix: "26.", DriverTokens: []string{"radminvpn"}},
}

func VPN(id string) (VPNDefinition, bool) {
	for _, v := range vpnDefinitions {
		if v.ID == id {
			return v, true
		}
	}
	return VPNDefinition{}, false
}
func IsVPN(id string) bool { _, ok := VPN(id); return ok }
func VirtualIPv4(ip string) bool {
	for _, v := range vpnDefinitions {
		if strings.HasPrefix(ip, v.IPv4Prefix) {
			return true
		}
	}
	return false
}
func Name(id string) string {
	if v, ok := VPN(id); ok {
		return v.Name
	}
	if id == "interface" {
		return "interface manual"
	}
	return "libp2p nativo"
}
func normalized(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(s))
}
func metadataMatches(s string, v VPNDefinition) bool {
	s = normalized(s)
	for _, t := range v.DriverTokens {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
