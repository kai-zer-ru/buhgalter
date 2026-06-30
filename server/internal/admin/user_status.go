package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

type updateUserStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	targetID := chi.URLParam(r, "id")
	if targetID == info.User.ID {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CANNOT_CHANGE_OWN_STATUS")
		return
	}

	var req updateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	to := auth.UserStatus(strings.TrimSpace(req.Status))
	if !to.Valid() || to == auth.UserStatusPending {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.UserStatusInvalid)
		return
	}

	var login string
	var fromStatus string
	err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT login, status FROM users WHERE id = ?`, targetID,
	).Scan(&login, &fromStatus)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	from := auth.UserStatus(fromStatus)
	if !auth.CanTransition(from, to) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.UserStatusTransition)
		return
	}

	_, err = h.Store.DB().ExecContext(r.Context(), `
		UPDATE users SET status = ?, updated_at = datetime('now') WHERE id = ?`,
		string(to), targetID,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if to == auth.UserStatusBanned {
		_ = auth.DeleteSessionsByUserID(r.Context(), h.Store.DB(), targetID)
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.user.status", info.User.ID, info.User.Login, ip, map[string]any{
		"target_id":    targetID,
		"target_login": login,
		"from":         fromStatus,
		"to":           string(to),
	})

	var u userItem
	var isAdmin int
	var createdAt string
	_ = h.Store.DB().QueryRowContext(r.Context(), `
		SELECT id, login, COALESCE(display_name, ''), is_admin, status, created_at
		FROM users WHERE id = ?`, targetID,
	).Scan(&u.ID, &u.Login, &u.DisplayName, &isAdmin, &u.Status, &createdAt)
	u.IsAdmin = isAdmin == 1
	u.CreatedAt = createdAt

	writeJSON(w, http.StatusOK, u)
}
