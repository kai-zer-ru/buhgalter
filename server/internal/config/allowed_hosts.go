package config

import (
	"encoding/json"
	"net"
	"strings"
)

const allowedHostsEnvKey = "BUHGALTER_ALLOWED_HOSTS"

func ParseHostList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			var out []string
			for _, host := range arr {
				host = strings.TrimSpace(host)
				if host != "" {
					out = append(out, normalizeAllowedHost(host))
				}
			}
			return out
		}
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = normalizeAllowedHost(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeAllowedHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host, "]") && strings.Count(host, ":") == 1 {
		host = host[:i]
	}
	host = strings.Trim(host, "[]")
	return strings.ToLower(host)
}
