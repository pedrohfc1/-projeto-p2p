package transportaddr

import (
	"crypto/sha256"
	"net"
	"strings"
)

// LogicalIP identifies a peer only inside the custom libp2p dialer, never an OS route.
func LogicalIP(id string) string {
	h := sha256.Sum256([]byte(id))
	h[0] = 0xfd
	return net.IP(h[:16]).String()
}
func ValidToken(t string) bool {
	if len(t) < 8 || len(t) > 2048 || !strings.HasPrefix(t, "etc2") {
		return false
	}
	for _, r := range t {
		if !(r >= 'a' && r <= 'z' || r >= '2' && r <= '7' || r == '.') {
			return false
		}
	}
	return true
}
