package apicache

import (
	"bytes"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

const (
	RefDataTTL = 5 * time.Minute
	DataTTL    = time.Minute
)

type responseRecorder struct {
	http.ResponseWriter
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func (rr *responseRecorder) WriteHeader(status int) {
	if !rr.wroteHeader {
		rr.status = status
		rr.wroteHeader = true
	}
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.wroteHeader {
		rr.WriteHeader(http.StatusOK)
	}
	return rr.body.Write(b)
}

func (rr *responseRecorder) flush() {
	if !rr.wroteHeader {
		rr.WriteHeader(http.StatusOK)
	}
	if rr.Header().Get("Content-Type") == "" {
		rr.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	for key, values := range rr.Header() {
		for _, value := range values {
			rr.ResponseWriter.Header().Add(key, value)
		}
	}
	rr.ResponseWriter.WriteHeader(rr.status)
	_, _ = rr.ResponseWriter.Write(rr.body.Bytes())
}

func writeCached(w http.ResponseWriter, item Response) {
	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(item.Status)
	_, _ = w.Write(item.Body)
}

// Middleware caches successful GET responses and invalidates user cache on writes.
func Middleware(cache *Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				if key, ttl, ok := cacheKey(r); ok {
					if item, hit := cache.Get(key); hit {
						writeCached(w, item)
						return
					}
					rec := &responseRecorder{ResponseWriter: w}
					next.ServeHTTP(rec, r)
					if rec.status == http.StatusOK && rec.body.Len() > 0 {
						cache.Set(key, Response{
							Status:      rec.status,
							Body:        append([]byte(nil), rec.body.Bytes()...),
							ContentType: rec.Header().Get("Content-Type"),
						}, ttl)
					}
					rec.flush()
					return
				}
			}

			rec := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if isMutating(r.Method) {
				invalidateForRequest(cache, r)
			}
			rec.flush()
		})
	}
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func invalidateForRequest(cache *Cache, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/setup") {
		cache.DeletePrefix("g:setup:")
		cache.Clear()
		return
	}
	if info, ok := auth.FromContext(r.Context()); ok {
		cache.DeletePrefix("u:" + info.User.ID + ":")
		if strings.HasPrefix(path, "/api/v1/admin/settings") {
			cache.DeletePrefix("g:setup:")
		}
	}
}

func cacheKey(r *http.Request) (string, time.Duration, bool) {
	path := r.URL.Path
	query := normalizedQuery(r.URL.Query())

	switch path {
	case "/api/v1/banks":
		return "g:banks", RefDataTTL, true
	case "/api/v1/setup/status":
		return "g:setup:status", DataTTL, true
	}

	if strings.HasPrefix(path, "/api/v1/health") ||
		path == "/api/v1/version/check" ||
		path == "/api/v1/export" ||
		strings.Contains(path, "/preview") ||
		strings.HasPrefix(path, "/api/v1/import/jobs/") {
		return "", 0, false
	}

	info, ok := auth.FromContext(r.Context())
	if !ok {
		return "", 0, false
	}

	key := "u:" + info.User.ID + ":" + path
	if query != "" {
		key += "?" + query
	}

	ttl := DataTTL
	if strings.HasPrefix(path, "/api/v1/categories") || path == "/api/v1/debtors" {
		ttl = RefDataTTL
	}
	return key, ttl, true
}

func normalizedQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		vals := append([]string(nil), values[key]...)
		sort.Strings(vals)
		for _, val := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(val))
		}
	}
	return strings.Join(parts, "&")
}
