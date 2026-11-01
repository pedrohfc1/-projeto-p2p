package overlay

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

func Registry(endpoints []string, pinned string) (string, error) {
	if len(endpoints) == 0 && pinned == "" {
		return DefaultRegistry, nil
	}
	if len(endpoints) == 0 || len(endpoints) > 8 {
		return "", errors.New("configure one to eight authorized relay endpoints")
	}
	probe := ""
	quoted := []string{}
	for _, raw := range endpoints {
		if len(raw) > 512 {
			return "", errors.New("relay URI too long")
		}
		u, e := url.Parse(raw)
		if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Hostname() == "" || !(u.Scheme == "tcp" || u.Scheme == "udp" || u.Scheme == "ws" || u.Scheme == "wss") {
			return "", errors.New("invalid authorized relay URI")
		}
		p := u.Port()
		if p == "" {
			if u.Scheme == "wss" {
				p = "443"
			} else if u.Scheme == "ws" {
				p = "80"
			} else {
				return "", errors.New("relay port required")
			}
		}
		n, e := strconv.Atoi(p)
		if e != nil || n < 1 || n > 65535 {
			return "", errors.New("invalid relay port")
		}
		if probe == "" {
			probe = u.Hostname() + ":" + p
		}
		quoted = append(quoted, strconv.Quote(raw))
	}
	s := "version = 1\n[[relay]]\nid = \"projeto-p2p-pool\"\nregion = \"Configured authorized endpoints\"\npriority = 100\nprobe = " + strconv.Quote(probe) + "\nendpoints = [" + strings.Join(quoted, ",") + "]\n"
	if pinned != "" {
		b, e := base64.StdEncoding.DecodeString(pinned)
		if e != nil || len(b) != 32 {
			return "", errors.New("relay public key must be 32 bytes base64")
		}
		s += "public_key = " + strconv.Quote(pinned) + "\n"
	}
	return s, nil
}
