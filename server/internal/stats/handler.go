package stats

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

type Handler struct {
	Store *db.Handle
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f := filtersFromQuery(r)
	svc := New(h.Store.DB())
	out, err := svc.Summary(r.Context(), info.User.ID, f, true)
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ByCategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f := filtersFromQuery(r)
	svc := New(h.Store.DB())
	out, err := svc.ByCategory(r.Context(), info.User.ID, f)
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) BySubcategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f := filtersFromQuery(r)
	svc := New(h.Store.DB())
	out, err := svc.BySubcategory(r.Context(), info.User.ID, f)
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) ByPeriod(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "day"
	}
	if groupBy != "day" && groupBy != "week" && groupBy != "month" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_GROUP_BY")
		return
	}
	f := filtersFromQuery(r)
	svc := New(h.Store.DB())
	out, err := svc.ByPeriod(r.Context(), info.User.ID, groupBy, f)
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	search := q.Get("q")
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	result, err := transaction.List(r.Context(), h.Store.DB(), info.User.ID, transaction.ListFilters{
		AccountID:  q.Get("account_id"),
		Type:       q.Get("type"),
		CategoryID: q.Get("category_id"),
		Kind:       q.Get("kind"),
		From:       q.Get("from"),
		To:         q.Get("to"),
		Search:     search,
		Sort:       q.Get("sort"),
		Page:       page,
		Limit:      limit,
	})
	if errors.Is(err, transaction.ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Context(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	f := filtersFromQuery(r)
	svc := New(h.Store.DB())

	var (
		out ContextSummary
		err error
	)
	switch {
	case q.Get("account_id") != "":
		out, err = svc.ContextAccount(r.Context(), info.User.ID, q.Get("account_id"), f)
	case q.Get("debtor_id") != "":
		out, err = svc.ContextDebtor(r.Context(), info.User.ID, q.Get("debtor_id"), f)
	case q.Get("credit_id") != "":
		out, err = svc.ContextCredit(r.Context(), info.User.ID, q.Get("credit_id"), f)
	case q.Get("debts") == "1":
		out, err = svc.ContextDebts(r.Context(), info.User.ID, f)
	default:
		out, err = svc.ContextDefault(r.Context(), info.User.ID, f)
	}
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func filtersFromQuery(r *http.Request) Filters {
	q := r.URL.Query()
	includeFuture := q.Get("include_future") == "true" || q.Get("include_future") == "1"
	return Filters{
		From:          q.Get("from"),
		To:            q.Get("to"),
		Type:          q.Get("type"),
		AccountID:     q.Get("account_id"),
		CategoryID:    q.Get("category_id"),
		Kind:          q.Get("kind"),
		Search:        q.Get("search"),
		IncludeFuture: includeFuture,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
