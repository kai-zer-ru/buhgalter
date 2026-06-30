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

	q := sqlcdb.New(h.Store.DB())
	current, err := q.GetUserLoginAndStatus(r.Context(), targetID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	from := auth.UserStatus(current.Status)
	if !auth.CanTransition(from, to) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.UserStatusTransition)
		return
	}

	if err := q.UpdateUserStatus(r.Context(), sqlcdb.UpdateUserStatusParams{
		Status: string(to),
		ID:     targetID,
	}); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if to == auth.UserStatusBanned {
		_ = auth.DeleteSessionsByUserID(r.Context(), h.Store.DB(), targetID)
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.user.status", info.User.ID, info.User.Login, ip, map[string]any{
		"target_id":    targetID,
		"target_login": current.Login,
		"from":         current.Status,
		"to":           string(to),
	})

	row, _ := q.GetUserAdminItem(r.Context(), targetID)
	writeJSON(w, http.StatusOK, userItem{
		ID:          row.ID,
		Login:       row.Login,
		DisplayName: row.DisplayName,
		IsAdmin:     row.IsAdmin == 1,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
	})
}
