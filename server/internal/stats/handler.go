package stats

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
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
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	out, err := svc.Summary(r.Context(), info.User.ID, f, true)
	if writeStatsError(w, r, err) {
		return
	}
	out.Currency = currency
	out.AvailableCurrencies = available
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ByCategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	out, err := svc.ByCategory(r.Context(), info.User.ID, f)
	if writeStatsError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": currency, "available_currencies": available})
}

func (h *Handler) BySubcategory(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	out, err := svc.BySubcategory(r.Context(), info.User.ID, f)
	if writeStatsError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": currency, "available_currencies": available})
}

func (h *Handler) ByStore(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	out, err := svc.ByStore(r.Context(), info.User.ID, f)
	if writeStatsError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": currency, "available_currencies": available})
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
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	out, err := svc.ByPeriod(r.Context(), info.User.ID, groupBy, f)
	if writeStatsError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": currency, "available_currencies": available})
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
	currency, available, ok := h.resolveCurrency(w, r, info)
	if !ok {
		return
	}
	result, err := transaction.ListOwned(r.Context(), h.Store.DB(), scope.FilterOwnerIDs(info.Scope, q.Get("scope")), info.User.ID, transaction.ListFilters{
		AccountID:  q.Get("account_id"),
		Type:       q.Get("type"),
		CategoryID: q.Get("category_id"),
		StoreID:    q.Get("store_id"),
		MerchantID: q.Get("merchant_id"),
		Kind:       q.Get("kind"),
		From:       q.Get("from"),
		To:         q.Get("to"),
		Search:     search,
		Sort:       q.Get("sort"),
		Page:       page,
		Limit:      limit,
		Currency:   currency,
	})
	if errors.Is(err, transaction.ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	result.Currency = currency
	result.AvailableCurrencies = available
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) NetWorth(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	groupBy := q.Get("group_by")
	if groupBy == "" {
		groupBy = "month"
	}
	currency, available, ok := h.resolveCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())
	ownerIDs := scope.FilterOwnerIDs(info.Scope, q.Get("scope"))
	out, err := svc.NetWorthOwned(r.Context(), info.User.ID, ownerIDs, q.Get("from"), q.Get("to"), groupBy, currency)
	if writeStatsError(w, r, err) {
		return
	}
	out.Currency = currency
	out.AvailableCurrencies = available
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Context(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	f, currency, available, ok := h.filtersWithCurrency(w, r, info)
	if !ok {
		return
	}
	svc := New(h.Store.DB())

	var (
		out ContextSummary
		err error
	)
	switch {
	case q.Get("account_id") != "":
		if acc, aerr := sqlcdb.New(h.Store.DB()).GetAccountByID(r.Context(), sqlcdb.GetAccountByIDParams{ID: q.Get("account_id"), OwnerIds: f.OwnerIDs}); aerr == nil {
			currency = money.ProfileOrDefault(acc.Currency)
			f.Currency = currency
		}
		out, err = svc.ContextAccount(r.Context(), info.User.ID, q.Get("account_id"), f)
	case q.Get("debtor_id") != "":
		out, err = svc.ContextDebtor(r.Context(), info.User.ID, q.Get("debtor_id"), f)
	case q.Get("credit_id") != "":
		if cr, cerr := sqlcdb.New(h.Store.DB()).GetCreditByID(r.Context(), sqlcdb.GetCreditByIDParams{
			ID: q.Get("credit_id"), OwnerIds: info.Scope.OwnerIDs(),
		}); cerr == nil {
			if acc, aerr := sqlcdb.New(h.Store.DB()).GetAccountByID(r.Context(), sqlcdb.GetAccountByIDParams{ID: cr.DebitAccountID, OwnerIds: info.Scope.OwnerIDs()}); aerr == nil {
				currency = money.ProfileOrDefault(acc.Currency)
				f.Currency = currency
			}
		}
		out, err = svc.ContextCredit(r.Context(), info.User.ID, q.Get("credit_id"), f)
	case q.Get("debts") == "1":
		out, err = svc.ContextDebts(r.Context(), info.User.ID, f)
	default:
		out, err = svc.ContextDefault(r.Context(), info.User.ID, f)
	}
	if writeStatsError(w, r, err) {
		return
	}
	out.Currency = currency
	out.AvailableCurrencies = available
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
		StoreID:       q.Get("store_id"),
		MerchantID:    q.Get("merchant_id"),
		Kind:          q.Get("kind"),
		Search:        q.Get("search"),
		IncludeFuture: includeFuture,
	}
}

func (h *Handler) resolveCurrency(w http.ResponseWriter, r *http.Request, info auth.AuthInfo) (currency string, available []string, ok bool) {
	ownerIDs := scope.FilterOwnerIDs(info.Scope, r.URL.Query().Get("scope"))
	available, err := account.ActiveCurrenciesOwned(r.Context(), h.Store.DB(), info.User.ID, ownerIDs)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return "", nil, false
	}
	profile, err := account.ProfileCurrency(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return "", nil, false
	}
	currency, err = money.ParseQueryCurrency(r.URL.Query().Get("currency"), profile)
	if errors.Is(err, money.ErrInvalidCurrency) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CURRENCY")
		return "", nil, false
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return "", nil, false
	}
	return currency, available, true
}

func (h *Handler) filtersWithCurrency(w http.ResponseWriter, r *http.Request, info auth.AuthInfo) (Filters, string, []string, bool) {
	f := filtersFromQuery(r)
	f.OwnerIDs = scope.FilterOwnerIDs(info.Scope, r.URL.Query().Get("scope"))
	currency, available, ok := h.resolveCurrency(w, r, info)
	if !ok {
		return Filters{}, "", nil, false
	}
	f.Currency = currency
	return f, currency, available, true
}

func writeStatsError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return true
	}
	if errors.Is(err, ErrInvalidGroupBy) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_GROUP_BY")
		return true
	}
	apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
