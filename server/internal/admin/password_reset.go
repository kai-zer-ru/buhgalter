package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type resetUserPasswordRequest struct {
	NewPassword        string `json:"new_password"`
	NewPasswordConfirm string `json:"new_password_confirm"`
}

func (h *Handler) ListPasswordResetRequests(w http.ResponseWriter, r *http.Request) {
	items, err := auth.ListPendingPasswordResetRequests(r.Context(), h.Store.DB())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if items == nil {
		items = []auth.PasswordResetRequest{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AckPasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	requestID := chi.URLParam(r, "id")
	if err := auth.DismissPasswordResetRequest(r.Context(), h.Store.DB(), requestID); err != nil {
		if err.Error() == "request not found" {
			apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.password_reset.ack", info.User.ID, info.User.Login, ip, map[string]any{
		"request_id": requestID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	targetID := chi.URLParam(r, "id")
	var req resetUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	targetLogin, err := sqlcdb.New(h.Store.DB()).GetUserLogin(r.Context(), targetID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	if err := auth.ValidatePassword(req.NewPassword, targetLogin); err != nil {
		msgKey := apperror.PasswordTooWeak
		fallback := "пароль должен содержать минимум одну букву и одну цифру и не совпадать с логином"
		if err == auth.ErrPasswordTooShort {
			msgKey = apperror.PasswordTooShort
			fallback = "пароль должен быть не короче 8 символов"
		}
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, msgKey, fallback)
		return
	}
	if req.NewPassword != req.NewPasswordConfirm {
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.PasswordsMismatch, "пароли не совпадают")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ctx := r.Context()
	if err := auth.SetUserPassword(ctx, h.Store.DB(), targetID, hash); err != nil {
		if err.Error() == "user not found" {
			apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = auth.DeleteSessionsByUserID(ctx, h.Store.DB(), targetID)
	_ = auth.DismissPasswordResetRequestsForUser(ctx, h.Store.DB(), targetID)

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.user.password.reset", info.User.ID, info.User.Login, ip, map[string]any{
		"target_login": strings.TrimSpace(targetLogin),
	})

	w.WriteHeader(http.StatusNoContent)
}
