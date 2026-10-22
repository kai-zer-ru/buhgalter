package budget

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
	"github.com/kai-zer-ru/buhgalter/internal/money"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type upsertRequest struct {
	Name           string  `json:"name"`
	Scope          string  `json:"scope"`
	CategoryID     *string `json:"category_id"`
	SubcategoryID  *string `json:"subcategory_id"`
	Amount         string  `json:"amount"`
	AccountID      *string `json:"account_id"`
	AlertAtPercent *int64  `json:"alert_at_percent"`
	IsActive       *bool   `json:"is_active"`
	CopyForward    *bool   `json:"copy_forward"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	month := r.URL.Query().Get("month")
	items, err := List(r.Context(), h.Store.DB(), info.User.ID, month)
	if writeBudgetError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	month := r.URL.Query().Get("month")
	result, err := Summary(r.Context(), h.Store.DB(), info.User.ID, month)
	if writeBudgetError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SpentPreview(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	result, err := PreviewSpent(r.Context(), h.Store.DB(), info.User.ID, SpentPreviewInput{
		Month:         q.Get("month"),
		Scope:         q.Get("scope"),
		CategoryID:    optionalQuery(q.Get("category_id")),
		SubcategoryID: optionalQuery(q.Get("subcategory_id")),
		AccountID:     optionalQuery(q.Get("account_id")),
	})
	if writeBudgetError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func optionalQuery(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
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
	in, err := parseInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	month, err := resolveMonth(r.Context(), h.Store.DB(), info.User.ID, r.URL.Query().Get("month"))
	if err != nil {
		writeBudgetError(w, r, err)
		return
	}
	in.Month = month
	item, err := Create(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeBudgetError(w, r, err) {
		return
	}
	_ = h.Audit.Log("budget.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID})
	writeJSON(w, http.StatusCreated, item)
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
	in, err := parseInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	month := r.URL.Query().Get("month")
	item, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, in, month)
	if writeBudgetError(w, r, err) {
		return
	}
	_ = h.Audit.Log("budget.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID})
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := Delete(r.Context(), h.Store.DB(), info.User.ID, id); writeBudgetError(w, r, err) {
		return
	}
	_ = h.Audit.Log("budget.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CopyNext(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	item, err := CopyToNextMonth(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeBudgetError(w, r, err) {
		return
	}
	_ = h.Audit.Log("budget.copy_next", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID, "from": id})
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) CopyFromPrevious(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	month := r.URL.Query().Get("month")
	items, err := CopyFromPreviousMonth(r.Context(), h.Store.DB(), info.User.ID, month)
	if writeBudgetError(w, r, err) {
		return
	}
	_ = h.Audit.Log("budget.copy_from_previous", info.User.ID, info.User.Login, clientIP(r), map[string]any{"month": month, "count": len(items)})
	writeJSON(w, http.StatusCreated, map[string]any{"items": items})
}

func parseInput(req upsertRequest) (Input, error) {
	amount, err := money.ParseRubles(req.Amount)
	if err != nil || amount <= 0 {
		return Input{}, errors.New("некорректная сумма")
	}
	alertAt := int64(90)
	if req.AlertAtPercent != nil {
		alertAt = *req.AlertAtPercent
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	copyForward := false
	if req.CopyForward != nil {
		copyForward = *req.CopyForward
	}
	name := strings.TrimSpace(req.Name)
	return Input{
		Name:           name,
		Scope:          req.Scope,
		CategoryID:     req.CategoryID,
		SubcategoryID:  req.SubcategoryID,
		Amount:         amount,
		AccountID:      req.AccountID,
		CopyForward:    copyForward,
		AlertAtPercent: alertAt,
		IsActive:       active,
	}, nil
}

func writeBudgetError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrDuplicateActive):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "ERR_BUDGET_DUPLICATE")
	case errors.Is(err, ErrCopyTargetExists):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "ERR_BUDGET_COPY_EXISTS")
	case errors.Is(err, ErrNothingToCopy):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_BUDGET_NOTHING_TO_COPY")
	case errors.Is(err, ErrInvalidScope):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_BUDGET_SCOPE")
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_AMOUNT_POSITIVE")
	case errors.Is(err, ErrInvalidMonth):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_BUDGET_MONTH")
	case errors.Is(err, ErrInvalidCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_NOT_FOUND")
	case errors.Is(err, ErrInvalidSub):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SUBCATEGORY_NOT_FOUND")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
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
