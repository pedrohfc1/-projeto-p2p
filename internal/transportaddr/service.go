package transportaddr

import (
	"crypto/sha256"
	"encoding/binary"
)

// ServicePort is a stable, unprivileged userspace service endpoint, not an OS route.
// It is independent of the user's LAN address and of native-mode port settings.
func ServicePort(id string) int {
	h := sha256.Sum256([]byte(id))
	return 12000 + int(binary.BigEndian.Uint32(h[:4])%48000)
}
