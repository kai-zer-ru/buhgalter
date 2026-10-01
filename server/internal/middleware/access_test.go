package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
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

func TestExternalAccessDeniesSpoofedLocalHostFromRemote(t *testing.T) {
	settingscache.Invalidate()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	h := ExternalAccess(db.NewHandle(mgr), nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	req.Host = "localhost"
	req.RemoteAddr = "203.0.113.10:54321"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	okReq.Host = "localhost"
	okReq.RemoteAddr = "127.0.0.1:54321"
	okRec := httptest.NewRecorder()
	h.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusNoContent {
		t.Fatalf("loopback status = %d, want 204", okRec.Code)
	}
}

func TestIsAccessAllowedHostRequiresLoopbackForLocalHost(t *testing.T) {
	allowed := allowedHostSet(nil)
	localReq := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/setup/status", nil)
	localReq.Host = "localhost:8765"
	localReq.RemoteAddr = "127.0.0.1:54321"
	if !isAccessAllowedHost(localReq, "localhost:8765", allowed) {
		t.Fatal("expected loopback client with Host localhost to be allowed")
	}
	if !isAccessAllowedHost(localReq, "127.0.0.1", allowed) {
		t.Fatal("expected loopback client with Host 127.0.0.1 to be allowed")
	}

	v6Req := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/setup/status", nil)
	v6Req.Host = "[::1]:8765"
	v6Req.RemoteAddr = "[::1]:54321"
	if !isAccessAllowedHost(v6Req, "[::1]:8765", allowed) {
		t.Fatal("expected IPv6 loopback client with Host ::1 to be allowed")
	}

	remoteReq := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/setup/status", nil)
	remoteReq.Host = "localhost"
	remoteReq.RemoteAddr = "203.0.113.10:54321"
	if isAccessAllowedHost(remoteReq, "localhost", allowed) {
		t.Fatal("expected remote client with spoofed Host localhost to be denied")
	}
	if isAccessAllowedHost(remoteReq, "127.0.0.1", allowed) {
		t.Fatal("expected remote client with spoofed Host 127.0.0.1 to be denied")
	}
	if isAccessAllowedHost(remoteReq, "::1", allowed) {
		t.Fatal("expected remote client with spoofed Host ::1 to be denied")
	}

	allowedLocal := allowedHostSet([]string{"localhost"})
	if !isAccessAllowedHost(remoteReq, "localhost", allowedLocal) {
		t.Fatal("expected remote client with Host localhost when listed in ALLOWED_HOSTS")
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
