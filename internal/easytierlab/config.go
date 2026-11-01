// Package easytierlab is an isolated experimental transport, not the community network.
package easytierlab

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const CoreSHA256 = "da7eb2d24b5416f3d3407636949e964a0750e3f9dc53a828cb6799a57ead445d"
const PacketSHA256 = "c7c03a87eac7243ccbe331554624b18803010b740e311fc8cfddb573096eacac"
const CLISHA256 = "d8783e851e944b44a9b71b39fd02f227ec0a2a82b3165c55ead5dd32dcde53a1"
const InvitePrefix = "projetop2p-lab:e1:"

type Config struct {
	ForceRelay    bool     `json:"forceRelay"`
	BinaryDir     string   `json:"binaryDir"`
	Network       string   `json:"network"`
	Secret        string   `json:"secret"`
	VirtualIP     string   `json:"virtualIp"`
	AppPort       int      `json:"appPort"`
	TransportPort int      `json:"transportPort"`
	SocksPort     int      `json:"socksPort"`
	RPCPort       int      `json:"rpcPort"`
	BindIP        string   `json:"bindIp"`
	Peers         []string `json:"peers"`
	STUN          []string `json:"stun"`
}

func Virtual(ip string) bool {
	v := net.ParseIP(ip)
	return v != nil && v.To4() != nil && strings.HasPrefix(v.String(), "10.203.") && v.To4()[3] != 0 && v.To4()[3] != 255
}
func (c Config) Validate() error {
	if len(c.Network) < 16 || len(c.Network) > 64 || len(c.Secret) < 32 || len(c.Secret) > 128 {
		return errors.New("use network name 16-64 and secret 32-128 characters")
	}
	for _, r := range c.Network + c.Secret {
		if r < 33 || r > 126 {
			return errors.New("printable ASCII credentials required")
		}
	}
	if !Virtual(c.VirtualIP) || !(c.BindIP == "127.0.0.1" || c.BindIP == "0.0.0.0") {
		return errors.New("invalid lab address")
	}
	seen := map[int]bool{}
	for _, v := range []int{c.AppPort, c.TransportPort, c.SocksPort, c.RPCPort} {
		if v < 1024 || v > 65535 || seen[v] {
			return errors.New("use four distinct unprivileged ports")
		}
		seen[v] = true
	}
	if len(c.Peers) > 8 || len(c.STUN) > 8 {
		return errors.New("maximum eight known peers and STUN endpoints")
	}
	for _, s := range c.Peers {
		u, e := url.Parse(s)
		if e != nil || len(s) > 256 || !(u.Scheme == "tcp" || u.Scheme == "udp") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("invalid authorized peer URI")
		}
		n, e := strconv.Atoi(u.Port())
		if e != nil || n < 1 || n > 65535 {
			return errors.New("peer port required")
		}
	}
	for _, s := range c.STUN {
		h, p, e := net.SplitHostPort(s)
		n, pe := strconv.Atoi(p)
		if e != nil || pe != nil || h == "" || n < 1 || n > 65535 || len(s) > 256 {
			return errors.New("invalid authorized STUN endpoint")
		}
	}
	return nil
}
func VerifyBinary(path, want string) error {
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || st.Size() > 100<<20 {
		return errors.New("invalid helper file")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, io.LimitReader(f, 100<<20+1)); e != nil {
		return e
	}
	if hex.EncodeToString(h.Sum(nil)) != want {
		return errors.New("EasyTier helper SHA-256 mismatch; execution blocked")
	}
	return nil
}
func (c Config) TOML() string {
	q := strconv.Quote
	s := fmt.Sprintf("hostname = \"projetop2p-lab\"\nipv4 = %s\nlisteners = [%s, %s]\nsocks5_proxy = %s\ntcp_whitelist = [%s]\nudp_whitelist = [\"0\"]\nstun_servers = [", q(c.VirtualIP+"/24"), q(fmt.Sprintf("tcp://%s:%d", c.BindIP, c.TransportPort)), q(fmt.Sprintf("udp://%s:%d", c.BindIP, c.TransportPort)), q(fmt.Sprintf("socks5://127.0.0.1:%d", c.SocksPort)), q(strconv.Itoa(c.AppPort)))
	for i, v := range c.STUN {
		if i > 0 {
			s += ","
		}
		s += q(v)
	}
	s += "]\nstun_servers_v6 = []\n[flags]\nno_tun = true\nbind_device = false\nuse_smoltcp = true\nenable_ipv6 = false\ndisable_upnp = true\nenable_udp_broadcast_relay = false\nproxy_forward_by_system = false\nenable_encryption = true\naccept_dns = false\nenable_exit_node = false\nprivate_mode = true\nforeign_relay_bps_limit = 0\ninstance_recv_bps_limit = 1048576\ndisable_kcp_input = true\ndisable_quic_input = true\n"
	s += "relay_network_whitelist = " + q(c.Network) + "\n"
	if c.ForceRelay {
		s += "disable_p2p = true\nneed_p2p = false\n"
	}
	for _, v := range c.Peers {
		s += "[[peer]]\nuri = " + q(v) + "\n"
	}
	return s
}
func helperPaths(dir string) (string, string, error) {
	dir, e := filepath.Abs(dir)
	if e != nil {
		return "", "", e
	}
	if e = VerifyBinary(filepath.Join(dir, "Packet.dll"), PacketSHA256); e != nil {
		return "", "", e
	}
	core := filepath.Join(dir, "easytier-core.exe")
	cli := filepath.Join(dir, "easytier-cli.exe")
	if e = VerifyBinary(core, CoreSHA256); e != nil {
		return "", "", e
	}
	if e = VerifyBinary(cli, CLISHA256); e != nil {
		return "", "", e
	}
	return core, cli, nil
}
