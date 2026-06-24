package auth

import (
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
)

// isAuthFailure reports whether err means the client is not authenticated
// (as opposed to a transient DB / infrastructure failure).
func isAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "session not found", "session expired", "empty token", "user not found", "api token not found", "api token expired":
		return true
	default:
		return false
	}
}

func writeUserLoadError(w http.ResponseWriter, r *http.Request, err error) {
	if isAuthFailure(err) {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized, "ERR_USER_NOT_FOUND")
		return
	}
	apperror.WriteR(w, r, http.StatusServiceUnavailable, apperror.ServiceUnavailable)
}
