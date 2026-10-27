package category

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
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type createCategoryRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	Scope     string `json:"scope"`
}

type updateCategoryRequest struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder *int   `json:"sort_order"`
}

type subcategoryRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type reorderCategoriesRequest struct {
	Type string   `json:"type"`
	IDs  []string `json:"ids"`
}

type reorderSubcategoriesRequest struct {
	IDs []string `json:"ids"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	catType := r.URL.Query().Get("type")
	cats, err := ListByScope(r.Context(), h.Store.DB(), info.Scope, r.URL.Query().Get("scope"), catType)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if cats == nil {
		cats = []Category{}
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	ownerID, err := scope.PrincipalForCreate(info.Scope, req.Scope)
	if writeScopeCreateError(w, r, err) {
		return
	}
	cat, err := Create(r.Context(), h.Store.DB(), ownerID, req.Name, req.Type, req.Icon, req.SortOrder)
	if writeCategoryError(w, r, err) {
		return
	}
	cat.Scope = scope.AccountScope(info.Scope, ownerID)
	_ = h.Audit.Log("category.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"category_id": cat.ID})
	writeJSON(w, http.StatusCreated, cat)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	cat, err := UpdateByScope(r.Context(), h.Store.DB(), info.Scope, id, req.Name, req.Icon, req.SortOrder)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, ErrSystemCategory) {
		apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_CATEGORY_SYSTEM_READONLY")
		return
	}
	if writeCategoryError(w, r, err) {
		return
	}
	_ = h.Audit.Log("category.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"category_id": id})
	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req reorderCategoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	if err := ReorderByScope(r.Context(), h.Store.DB(), info.Scope, req.Type, req.IDs); err != nil {
		if errors.Is(err, ErrInvalidReorder) {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_REORDER")
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	cats, err := ListByScope(r.Context(), h.Store.DB(), info.Scope, "", req.Type)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if cats == nil {
		cats = []Category{}
	}
	_ = h.Audit.Log("category.reorder", info.User.ID, info.User.Login, clientIP(r), map[string]any{"type": req.Type})
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) SetPrimary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	cat, err := SetPrimaryByScope(r.Context(), h.Store.DB(), info.Scope, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("category.set_primary", info.User.ID, info.User.Login, clientIP(r), map[string]any{"category_id": id})
	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	err := DeleteByScope(r.Context(), h.Store.DB(), info.Scope, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, ErrSystemCategory) {
		apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_CATEGORY_SYSTEM_DELETE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("category.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"category_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListSubcategories(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	categoryID := chi.URLParam(r, "id")
	parent, err := GetByScope(r.Context(), h.Store.DB(), info.Scope, categoryID)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	subs, err := ListSubcategories(r.Context(), h.Store.DB(), parent.OwnerUserID, categoryID)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if subs == nil {
		subs = []Subcategory{}
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	categoryID := chi.URLParam(r, "id")
	var req subcategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	parent, err := GetByScope(r.Context(), h.Store.DB(), info.Scope, categoryID)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	sub, err := CreateSubcategory(r.Context(), h.Store.DB(), parent.OwnerUserID, categoryID, req.Name, req.Icon)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if writeSubcategoryError(w, r, err) {
		return
	}
	_ = h.Audit.Log("subcategory.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{
		"category_id": categoryID, "subcategory_id": sub.ID,
	})
	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) ReorderSubcategories(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	categoryID := chi.URLParam(r, "id")
	var req reorderSubcategoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	parent, err := GetByScope(r.Context(), h.Store.DB(), info.Scope, categoryID)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	subs, err := ReorderSubcategories(r.Context(), h.Store.DB(), parent.OwnerUserID, categoryID, req.IDs)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, ErrInvalidReorder) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SUBCATEGORY_REORDER")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if subs == nil {
		subs = []Subcategory{}
	}
	_ = h.Audit.Log("subcategory.reorder", info.User.ID, info.User.Login, clientIP(r), map[string]any{"category_id": categoryID})
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) UpdateSubcategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req subcategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	sub, err := UpdateSubcategoryOwned(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), id, req.Name, req.Icon)
	if errors.Is(err, ErrSubNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if writeSubcategoryError(w, r, err) {
		return
	}
	_ = h.Audit.Log("subcategory.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"subcategory_id": id})
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) DeleteSubcategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	err := DeleteSubcategoryOwned(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), id)
	if errors.Is(err, ErrSubNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("subcategory.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"subcategory_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func writeScopeCreateError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, scope.ErrNoHousehold):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_NOT_MEMBER")
	case errors.Is(err, scope.ErrInvalidScope):
		apperror.WriteR(w, r, http.StatusBadRequest, "ERR_SCOPE_MISMATCH")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return true
}

func writeCategoryError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrInvalidName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_NAME_LENGTH")
	case errors.Is(err, ErrNameTaken):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_CATEGORY_NAME")
	case errors.Is(err, ErrSystemCategory):
		apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_CATEGORY_SYSTEM_READONLY")
	default:
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_INVALID")
	}
	return true
}

func writeSubcategoryError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrInvalidName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SUBCATEGORY_NAME_LENGTH")
	case errors.Is(err, ErrSubNameTaken):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_SUBCATEGORY_NAME")
	case errors.Is(err, ErrSystemCategory):
		apperror.WriteR(w, r, http.StatusForbidden, apperror.Forbidden, "ERR_SUBCATEGORY_SYSTEM")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return true
}

func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	return strings.TrimSpace(ip)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
