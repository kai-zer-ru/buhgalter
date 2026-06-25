package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"localhost:8765", "localhost"},
		{"127.0.0.1:8765", "127.0.0.1"},
		{"Buhgalter.Example.COM", "buhgalter.example.com"},
		{"[::1]:8765", "::1"},
	}
	for _, tc := range tests {
		if got := normalizeHost(tc.in); got != tc.want {
			t.Fatalf("normalizeHost(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRequestHostTrustProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/v1/health", nil)
	r.Host = "203.0.113.10:8765"
	r.Header.Set("X-Forwarded-Host", "buhgalter.example.com")

	if got := requestHost(r, false); got != "203.0.113.10" {
		t.Fatalf("untrusted proxy host = %q", got)
	}
	if got := requestHost(r, true); got != "buhgalter.example.com" {
		t.Fatalf("trusted proxy host = %q", got)
	}
}

func TestIsLocalHost(t *testing.T) {
	for _, host := range []string{"localhost", "localhost:8765", "127.0.0.1", "[::1]:8765"} {
		if !isLocalHost(host) {
			t.Fatalf("expected local host for %q", host)
		}
	}
	if isLocalHost("203.0.113.10") || isLocalHost("buhgalter.example.com") {
		t.Fatal("expected public host to be non-local")
	}
}

func TestIsConfiguredAllowedHost(t *testing.T) {
	allowed := allowedHostSet([]string{"203.0.113.10", "Buhgalter.Example.COM:443"})
	if !isConfiguredAllowedHost("203.0.113.10:8765", allowed) {
		t.Fatal("expected configured host to match")
	}
	if isConfiguredAllowedHost("8.8.8.8", allowed) {
		t.Fatal("expected unknown host to be denied")
	}
}

func TestHostnameFromExternalURL(t *testing.T) {
	host, err := hostnameFromExternalURL("https://Buhgalter.Example.COM:443/app")
	if err != nil {
		t.Fatal(err)
	}
	if host != "buhgalter.example.com" {
		t.Fatalf("host = %q", host)
	}
}
