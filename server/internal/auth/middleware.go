package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type ctxKey string

const AuthContextKey ctxKey = "auth"

type AuthInfo struct {
	User      User
	SessionID string
	Token     string
	APIToken  bool
}

func extractToken(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func RequireAuth(store *db.Handle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
				return
			}

			sqlDB := store.DB()
			session, err := LookupSession(r.Context(), sqlDB, token)
			if err == nil {
				user, err := LoadUser(r.Context(), sqlDB, session.UserID)
				if err != nil {
					writeUserLoadError(w, r, err)
					return
				}
				info := AuthInfo{User: *user, SessionID: session.ID, Token: token}
				ctx := context.WithValue(r.Context(), AuthContextKey, info)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			userID, err := LookupAPIToken(r.Context(), sqlDB, token)
			if err != nil {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
				return
			}
			user, err := LoadUser(r.Context(), sqlDB, userID)
			if err != nil {
				writeUserLoadError(w, r, err)
				return
			}
			info := AuthInfo{User: *user, Token: token, APIToken: true}
			ctx := context.WithValue(r.Context(), AuthContextKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, ok := FromContext(r.Context())
		if !ok {
			apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
			return
		}
		if !info.User.IsAdmin {
			apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAPIToken(store *db.Handle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_API_TOKEN_REQUIRED")
				return
			}

			sqlDB := store.DB()
			hash := HashToken(token)
			var userID, tokenID string
			var expiresAt sql.NullString
			err := sqlDB.QueryRowContext(r.Context(), `
				SELECT id, user_id, expires_at FROM api_tokens WHERE token_hash = ?`, hash,
			).Scan(&tokenID, &userID, &expiresAt)
			if err != nil {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_API_TOKEN_INVALID")
				return
			}

			if expiresAt.Valid && expiresAt.String != "" {
				if !verifyAPIToken(r.Context(), sqlDB, token) {
					apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_API_TOKEN_EXPIRED")
					return
				}
			} else {
				_, _ = sqlDB.ExecContext(r.Context(), `UPDATE api_tokens SET last_used_at = datetime('now') WHERE id = ?`, tokenID)
			}

			user, err := LoadUser(r.Context(), sqlDB, userID)
			if err != nil {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_USER_NOT_FOUND")
				return
			}

			info := AuthInfo{User: *user, Token: token, APIToken: true}
			ctx := context.WithValue(r.Context(), AuthContextKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FromContext(ctx context.Context) (AuthInfo, bool) {
	v, ok := ctx.Value(AuthContextKey).(AuthInfo)
	return v, ok
}

func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(IdleTimeout.Seconds()),
	})
}

func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func ClientIP(r *http.Request) string {
	return appmw.ClientIP(r)
}
