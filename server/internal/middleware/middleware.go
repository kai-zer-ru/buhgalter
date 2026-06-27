package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

func RequestID(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

func RequestIDToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.GetReqID(r.Context())
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logger(logger *slog.Logger, verbose bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			respBuf := &bytes.Buffer{}
			ww.Tee(respBuf)
			next.ServeHTTP(ww, r)

			id, _ := r.Context().Value(RequestIDKey).(string)
			auth := redactAuth(r.Header.Get("Authorization"))
			cookie := redactCookie(r.Header.Get("Cookie"))
			if verbose {
				auth = r.Header.Get("Authorization")
				cookie = r.Header.Get("Cookie")
			}
			attrs := []any{
				"request_id", id,
				"method", r.Method,
				"path", redactPath(r.URL.Path),
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"authorization", auth,
				"cookie", cookie,
			}
			if msg, code, field := parseErrorResponse(respBuf.Bytes()); msg != "" {
				attrs = append(attrs, "error_code", code, "error_message", msg)
				if field != "" {
					attrs = append(attrs, "error_field", field)
				}
			}
			if ww.Status() >= http.StatusInternalServerError {
				logger.Error("request", attrs...)
				return
			}
			logger.Info("request", attrs...)
		})
	}
}

func parseErrorResponse(payload []byte) (message, code, field string) {
	if len(payload) == 0 {
		return "", "", ""
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Field   string `json:"field"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return "", "", ""
	}
	return body.Error.Message, body.Error.Code, body.Error.Field
}

func redactPath(path string) string {
	return path
}

func redactAuth(v string) string {
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "Bearer ") {
		return "Bearer [REDACTED]"
	}
	return "[REDACTED]"
}

func redactCookie(v string) string {
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ";")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "session=") {
			parts[i] = "session=[REDACTED]"
		}
	}
	return strings.Join(parts, "; ")
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic", "err", rec, "stack", string(debug.Stack()))
					apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				} else if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type IPRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	history := l.requests[ip]
	var active []time.Time
	for _, t := range history {
		if t.After(cutoff) {
			active = append(active, t)
		}
	}
	if len(active) >= l.limit {
		l.requests[ip] = active
		return false
	}
	active = append(active, now)
	l.requests[ip] = active
	return true
}

func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	addr := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
