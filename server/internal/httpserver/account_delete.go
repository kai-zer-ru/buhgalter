package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type accountDeleteHandler struct {
	store *db.Handle
	audit *audit.Logger
}

func (h *accountDeleteHandler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req transferToRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
			return
		}
	}

	_, err := inactivateAccount(r.Context(), h.store.DB(), info.User.ID, id, "deleted", parseTransferToAccountID(r, req))
	if writeInactivateAccountError(w, r, err) {
		return
	}

	_ = h.audit.Log("account.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	return strings.TrimSpace(ip)
}
