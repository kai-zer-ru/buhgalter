package categoryrule

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
	Kind             string  `json:"kind"`
	Pattern          *string `json:"pattern"`
	CategoryID       string  `json:"category_id"`
	SubcategoryID    *string `json:"subcategory_id"`
	DefaultAccountID *string `json:"default_account_id"`
}

type orderRequest struct {
	IDs []string `json:"ids"`
}

type evaluateRequest struct {
	Description   *string `json:"description"`
	CategoryID    *string `json:"category_id"`
	SubcategoryID *string `json:"subcategory_id"`
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
		items = []Rule{}
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
	rule, err := Create(r.Context(), h.Store.DB(), info.User.ID, inputFromReq(req))
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("category_rule.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"rule_id": rule.ID})
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	rule, err := Get(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, rule)
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
	rule, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, inputFromReq(req))
	if writeError(w, r, err) {
		return
	}
	_ = h.Audit.Log("category_rule.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"rule_id": id})
	writeJSON(w, http.StatusOK, rule)
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
	_ = h.Audit.Log("category_rule.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"rule_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Evaluate(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	result, err := Evaluate(r.Context(), h.Store.DB(), info.User.ID, EvaluateInput(req))
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if result.TextMatches == nil {
		result.TextMatches = []TextMatchRef{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req orderRequest
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
		items = []Rule{}
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
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_LIMIT")
	case errors.Is(err, ErrInvalidKind):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_KIND")
	case errors.Is(err, ErrInvalidPattern):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_PATTERN")
	case errors.Is(err, ErrInvalidCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_CATEGORY")
	case errors.Is(err, ErrInvalidSubcategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_SUBCATEGORY")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_ACCOUNT")
	case errors.Is(err, ErrInvalidOrder):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_ORDER")
	case errors.Is(err, ErrSystemCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_RULE_SYSTEM_CATEGORY")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
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
