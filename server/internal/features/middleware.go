package features

import (
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

// RequireFeature returns middleware that rejects requests when the flag is off.
// Disabled modules respond with 404 FEATURE_DISABLED.
func RequireFeature(store *db.Handle, key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enabled, err := IsEnabled(r.Context(), store.DB(), key)
			if err != nil {
				apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
				return
			}
			if !enabled {
				apperror.WriteR(w, r, http.StatusNotFound, apperror.FeatureDisabled)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
