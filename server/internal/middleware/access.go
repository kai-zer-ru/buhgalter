package middleware

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

// ExternalAccess limits HTTP access by Host based on system_settings.external_url.
// Empty external_url → localhost and private/LAN addresses (RFC 1918, link-local).
// Set external_url → allowed Host matches URL hostname; localhost/LAN kept for local admin.
func ExternalAccess(store *db.Handle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/health" && isLoopbackIP(directClientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := externalAccessAllowed(r.Context(), store.DB(), r)
			if err != nil {
				apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
				return
			}
			if !allowed {
				apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_EXTERNAL_ACCESS_DENIED")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func externalAccessAllowed(ctx context.Context, sqlDB *sql.DB, r *http.Request) (bool, error) {
	var externalURL sql.NullString
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT external_url FROM system_settings WHERE id = 1`,
	).Scan(&externalURL); err != nil {
		return false, err
	}

	configured := externalURL.Valid && strings.TrimSpace(externalURL.String) != ""
	host := requestHost(r, configured)

	if !configured {
		return isDirectAccessHost(host), nil
	}

	wantHost, err := hostnameFromExternalURL(externalURL.String)
	if err != nil {
		return false, err
	}
	if hostMatches(host, wantHost) || isDirectAccessHost(host) {
		return true, nil
	}
	return false, nil
}

func requestHost(r *http.Request, trustProxy bool) string {
	raw := r.Host
	if trustProxy {
		if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
			raw = strings.TrimSpace(strings.Split(fwd, ",")[0])
		}
	}
	return normalizeHost(raw)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host, "]") {
		// host:port without brackets (not IPv6)
		if strings.Count(host, ":") == 1 {
			host = host[:i]
		}
	}
	host = strings.Trim(host, "[]")
	return strings.ToLower(host)
}

func hostnameFromExternalURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return normalizeHost(u.Host), nil
}

func hostMatches(got, want string) bool {
	got = normalizeHost(got)
	want = normalizeHost(want)
	return got != "" && want != "" && got == want
}

func isLocalHost(host string) bool {
	switch normalizeHost(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

// isDirectAccessHost — loopback, private LAN (10/8, 172.16/12, 192.168/16), link-local.
func isDirectAccessHost(host string) bool {
	if isLocalHost(host) {
		return true
	}
	ip := net.ParseIP(normalizeHost(host))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func isLoopbackIP(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	return parsed != nil && parsed.IsLoopback()
}

func directClientIP(r *http.Request) string {
	addr := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
