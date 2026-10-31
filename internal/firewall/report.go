// Package firewall reads diagnostic policy only. It never creates or edits rules.
package firewall

import (
	"strconv"
	"strings"
)

type Report struct {
	Available           bool   `json:"available"`
	Port                int    `json:"port"`
	PotentialAllowRules int    `json:"potentialAllowRules"`
	PotentialBlockRules int    `json:"potentialBlockRules"`
	Summary             string `json:"summary"`
}

func portMatch(spec string, port int) bool {
	for _, part := range strings.Split(strings.ToLower(spec), ",") {
		part = strings.TrimSpace(part)
		if part == "*" || part == "any" {
			return true
		}
		if n, e := strconv.Atoi(part); e == nil && n == port {
			return true
		}
		lo, hi, ok := strings.Cut(part, "-")
		if ok {
			a, e := strconv.Atoi(lo)
			b, f := strconv.Atoi(hi)
			if e == nil && f == nil && port >= a && port <= b {
				return true
			}
		}
	}
	return false
}
