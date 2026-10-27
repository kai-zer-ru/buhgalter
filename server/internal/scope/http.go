package scope

import (
	"errors"
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
)

func WriteCreateError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNoHousehold):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_NOT_MEMBER")
	case errors.Is(err, ErrInvalidScope):
		apperror.WriteR(w, r, http.StatusBadRequest, "ERR_SCOPE_MISMATCH")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return true
}
