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

func TestIsLocalHost(t *testing.T) {
	if !isLocalHost("localhost") || !isLocalHost("127.0.0.1:8765") {
		t.Fatal("expected localhost hosts to be local")
	}
	if isLocalHost("203.0.113.10") || isLocalHost("buhgalter.example.com") {
		t.Fatal("expected public hosts to be non-local")
	}
}

func TestIsDirectAccessHost(t *testing.T) {
	allowed := []string{"127.0.0.1", "192.168.1.8:8765", "10.0.0.5", "172.16.3.1", "169.254.1.1"}
	for _, host := range allowed {
		if !isDirectAccessHost(host) {
			t.Fatalf("expected direct access for %q", host)
		}
	}
	denied := []string{"203.0.113.10", "buhgalter.example.com", "8.8.8.8"}
	for _, host := range denied {
		if isDirectAccessHost(host) {
			t.Fatalf("expected deny for %q", host)
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

func TestHostnameFromExternalURL(t *testing.T) {
	host, err := hostnameFromExternalURL("https://Buhgalter.Example.COM:443/app")
	if err != nil {
		t.Fatal(err)
	}
	if host != "buhgalter.example.com" {
		t.Fatalf("host = %q", host)
	}
}
