package discovery

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseHTTPPort extracts the TCP port from BUHGALTER_ADDR-style values (:8765, host:8765).
func ParseHTTPPort(addr string) (int, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0, fmt.Errorf("empty addr")
	}
	if !strings.Contains(addr, ":") {
		port, err := strconv.Atoi(addr)
		if err != nil || port < 1 || port > 65535 {
			return 0, fmt.Errorf("invalid port %q", addr)
		}
		return port, nil
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	_ = host
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", portStr)
	}
	return port, nil
}
