package middleware

import (
	"net/http"
	"strings"
)

// isNoiseProbe reports automated scanner paths that are not part of the app.
func isNoiseProbe(path string) bool {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	path = strings.TrimSuffix(path, "/")
	seg, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return strings.HasPrefix(seg, ".env") && len(seg) >= 4
}

// RejectNoiseProbes answers common vulnerability scans with 404 before routing.
func RejectNoiseProbes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isNoiseProbe(r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
