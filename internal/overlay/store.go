package overlay

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"net"
	"os"
	"path/filepath"
	"projetop2p/internal/identity"
	"strings"
)

const MaxToken = 2048
const MaxSessions = 8
const DefaultRegistry = `version = 1
[[relay]]
id = "projeto-p2p-pool"
region = "Community (unpinned, best effort)"
probe = "38.76.179.190:11010"
priority = 100
endpoints = ["tcp://38.76.179.190:11010", "udp://38.76.179.190:11010"]
`

type saved struct {
	Version int               `json:"version"`
	Server  json.RawMessage   `json:"server"`
	Routes  map[string]string `json:"routes"`
}

func validToken(t string) bool {
	if len(t) < 8 || len(t) > MaxToken || !strings.HasPrefix(t, "etc2") {
		return false
	}
	for _, r := range t {
		if !(r >= 'a' && r <= 'z' || r >= '2' && r <= '7' || r == '.') {
			return false
		}
	}
	return true
}
func LogicalIP(id string) string {
	h := sha256.Sum256([]byte(id))
	h[0] = 0xfd
	return net.IP(h[:16]).String()
}
func loadStore(profile string) (saved, error) {
	s := saved{Version: 1, Routes: map[string]string{}}
	path := filepath.Join(profile, "integrated-transport.sealed")
	if st, e := os.Lstat(path); e == nil {
		if !st.Mode().IsRegular() || st.Size() > 1<<20 {
			return s, errors.New("invalid sealed transport state")
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return s, e
		}
		b, e = identity.UnsealLocalData(b)
		if e != nil {
			return s, e
		}
		if e = json.Unmarshal(b, &s); e != nil {
			return s, e
		}
		if s.Version != 1 || s.Routes == nil || len(s.Routes) > 256 || len(s.Server) > 16384 {
			return s, errors.New("invalid transport state version/limits")
		}
		for id, t := range s.Routes {
			if _, e = peer.Decode(id); e != nil || !validToken(t) {
				return s, errors.New("invalid sealed route")
			}
		}
		return s, nil
	} else if !os.IsNotExist(e) {
		return s, e
	}
	seed := make([]byte, 32)
	private := make([]byte, 32)
	secret := make([]byte, 32)
	credential := make([]byte, 16)
	for _, b := range [][]byte{seed, private, secret, credential} {
		if _, e := rand.Read(b); e != nil {
			return s, e
		}
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	fp := sha256.Sum256(pub)
	name := "etcat-" + hex.EncodeToString(fp[:16])
	ip := sha256.Sum256([]byte(name))
	port := 30000 + (int(fp[16])<<8|int(fp[17]))%25000
	enc := base64.StdEncoding.EncodeToString
	material := map[string]any{"kind": "server", "identity": map[string]any{"network_name": name, "network_secret": enc(secret), "private_key": enc(private), "signing_key": enc(seed), "hostname": "ProjetoP2P", "server_ipv4": fmt.Sprintf("10.%d.%d.1", ip[0], ip[1])}, "credential_secret": enc(credential), "relay_id": "projeto-p2p-pool", "fixed_relay": true, "gateway_port": port}
	s.Server, _ = json.Marshal(material)
	return s, saveStore(profile, s)
}
func saveStore(profile string, s saved) error {
	b, e := json.Marshal(s)
	if e != nil {
		return e
	}
	b, e = identity.SealLocalData(b)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(profile, ".transport-sealed-")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(tmp, filepath.Join(profile, "integrated-transport.sealed"))
}
