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
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
)

// ExternalAccess limits HTTP access by Host based on system_settings.external_url.
// localhost / loopback is always allowed. Otherwise: BUHGALTER_ALLOWED_HOSTS in .env;
// with external_url set — also the URL hostname.
func ExternalAccess(store *db.Handle, allowedHosts []string) func(http.Handler) http.Handler {
	allowed := allowedHostSet(allowedHosts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/health" && isLoopbackIP(directClientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}

			ok, err := externalAccessAllowed(r.Context(), store.DB(), r, allowed)
			if err != nil {
				apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
				return
			}
			if !ok {
				apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_EXTERNAL_ACCESS_DENIED")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowedHostSet(hosts []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		host = normalizeHost(host)
		if host != "" {
			allowed[host] = struct{}{}
		}
	}
	return allowed
}

func isConfiguredAllowedHost(host string, allowed map[string]struct{}) bool {
	_, ok := allowed[normalizeHost(host)]
	return ok
}

func externalAccessAllowed(ctx context.Context, sqlDB *sql.DB, r *http.Request, allowed map[string]struct{}) (bool, error) {
	externalURL, err := settingscache.ExternalURL(ctx, sqlDB)
	if err != nil {
		return false, err
	}

	configured := externalURL.Valid && strings.TrimSpace(externalURL.String) != ""
	host := requestHost(r, configured)

	if !configured {
		return isAccessAllowedHost(host, allowed), nil
	}

	wantHost, err := hostnameFromExternalURL(externalURL.String)
	if err != nil {
		return false, err
	}
	if hostMatches(host, wantHost) || isAccessAllowedHost(host, allowed) {
		return true, nil
	}
	return false, nil
}

func isAccessAllowedHost(host string, allowed map[string]struct{}) bool {
	return isLocalHost(host) || isConfiguredAllowedHost(host, allowed)
}

func isLocalHost(host string) bool {
	switch normalizeHost(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
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
