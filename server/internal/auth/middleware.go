package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
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
			session, sessionUser, err := LookupSessionWithUser(r.Context(), sqlDB, token)
			if err == nil {
				if RejectIfNotActive(w, r, sessionUser.Status) {
					return
				}
				info := AuthInfo{User: *sessionUser, SessionID: session.ID, Token: token}
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
			if RejectIfNotActive(w, r, user.Status) {
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
			row, err := queries(sqlDB).GetAPITokenByHash(r.Context(), hash)
			if err != nil {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_API_TOKEN_INVALID")
				return
			}

			if apiTokenExpired(row.ExpiresAt) {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_API_TOKEN_EXPIRED")
				return
			}
			_ = queries(sqlDB).TouchAPITokenByID(r.Context(), row.ID)

			user, err := LoadUser(r.Context(), sqlDB, row.UserID)
			if err != nil {
				apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_USER_NOT_FOUND")
				return
			}
			if RejectIfNotActive(w, r, user.Status) {
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
