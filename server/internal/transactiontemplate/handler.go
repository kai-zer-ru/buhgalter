package transactiontemplate

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
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	AccountID     *string  `json:"account_id"`
	ToAccountID   *string  `json:"to_account_id"`
	CategoryID    *string  `json:"category_id"`
	SubcategoryID *string  `json:"subcategory_id"`
	Amount        *int64   `json:"amount"`
	Description   *string  `json:"description"`
	MerchantID    *string  `json:"merchant_id"`
	Icon          *string  `json:"icon"`
	TagIDs        []string `json:"tag_ids"`
}

type reorderRequest struct {
	IDs []string `json:"ids"`
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
		items = []Template{}
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
	t, err := Create(r.Context(), h.Store.DB(), info.User.ID, inputFromReq(req))
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transaction_template.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"template_id": t.ID})
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	t, err := Get(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, t)
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
	t, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, inputFromReq(req))
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transaction_template.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"template_id": id})
	writeJSON(w, http.StatusOK, t)
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
	_ = h.Audit.Log("transaction_template.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"template_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req reorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	if err := Reorder(r.Context(), h.Store.DB(), info.User.ID, req.IDs); err != nil {
		if writeError(w, r, err) {
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	items, err := List(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if items == nil {
		items = []Template{}
	}
	writeJSON(w, http.StatusOK, items)
}

func inputFromReq(req upsertRequest) Input {
	return Input(req)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrLimitReached):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_LIMIT")
	case errors.Is(err, ErrInvalidName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_NAME")
	case errors.Is(err, ErrInvalidType):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_TYPE")
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_AMOUNT")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_ACCOUNT")
	case errors.Is(err, ErrInvalidCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_CATEGORY")
	case errors.Is(err, ErrInvalidMerchant):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_MERCHANT")
	case errors.Is(err, ErrInvalidTag):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_TAG")
	case errors.Is(err, ErrInvalidReorder):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TEMPLATE_REORDER")
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
