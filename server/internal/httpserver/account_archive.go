package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type accountArchiveHandler struct {
	store *db.Handle
	audit *audit.Logger
}

func (h *accountArchiveHandler) archiveAccount(w http.ResponseWriter, r *http.Request) {
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

	updated, err := inactivateAccount(r.Context(), h.store.DB(), info.User.ID, id, "archived", parseTransferToAccountID(r, req))
	if writeInactivateAccountError(w, r, err) {
		return
	}

	_ = h.audit.Log("account.archive", info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": id})
	writeJSON(w, http.StatusOK, updated)
}
