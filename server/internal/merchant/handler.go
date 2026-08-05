package merchant

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type upsertRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	items, err := List(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if items == nil {
		items = []Merchant{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	m, err := Create(r.Context(), h.Store.DB(), info.User.ID, req.Name, req.Icon)
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("merchant.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"merchant_id": m.ID})
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	m, err := Get(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	m, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, req.Name, req.Icon)
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("merchant.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"merchant_id": id})
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := Delete(r.Context(), h.Store.DB(), info.User.ID, id); err != nil {
		if writeError(w, r, err) {
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("merchant.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"merchant_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrNameTaken):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_MERCHANT_NAME")
	case errors.Is(err, ErrInvalidName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_MERCHANT_NAME_REQUIRED")
	default:
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
