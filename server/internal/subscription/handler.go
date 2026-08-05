package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type upsertRequest struct {
	Name        string  `json:"name"`
	Amount      string  `json:"amount"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	WebsiteURL  *string `json:"website_url"`
	AccountID   string  `json:"account_id"`
	Period      string  `json:"period"`
	Weekday     *int64  `json:"weekday"`
	DayOfMonth  *int64  `json:"day_of_month"`
	StartDate   string  `json:"start_date"`
	TimeLocal   string  `json:"time_local"`
	Active      *bool   `json:"active"`
	AttachTxID  *string `json:"attach_transaction_id"`
}

type convertRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	WebsiteURL  *string `json:"website_url"`
}

type attachRequest struct {
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
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	days := 14
	if v := r.URL.Query().Get("upcoming_days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	sum, err := SummaryForUser(r.Context(), h.Store.DB(), info.User.ID, days)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, sum)
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
	item, err := Create(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeSubError(w, r, err) {
		return
	}
	if req.AttachTxID != nil && *req.AttachTxID != "" {
		_, _ = AttachTransactions(r.Context(), h.Store.DB(), info.User.ID, item.ID, []string{*req.AttachTxID})
		item, _ = getByID(r.Context(), h.Store.DB(), info.User.ID, item.ID)
	}
	_ = h.Audit.Log("subscription.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID})
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
	item, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, in)
	if writeSubError(w, r, err) {
		return
	}
	_ = h.Audit.Log("subscription.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": id})
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if writeSubError(w, r, Delete(r.Context(), h.Store.DB(), info.User.ID, id)) {
		return
	}
	_ = h.Audit.Log("subscription.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req attachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	res, err := AttachTransactions(r.Context(), h.Store.DB(), info.User.ID, id, req.IDs)
	if writeSubError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) Candidates(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	q := r.URL.Query()
	f := CandidateFilter{
		Mode:  q.Get("mode"),
		Query: q.Get("q"),
	}
	if v := q.Get("account_id"); v != "" {
		f.AccountID = &v
	}
	if v := q.Get("amount_min"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AmountMin = &n
		}
	}
	if v := q.Get("amount_max"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AmountMax = &n
		}
	}
	if v := q.Get("date_from"); v != "" {
		f.DateFrom = &v
	}
	if v := q.Get("date_to"); v != "" {
		f.DateTo = &v
	}
	if v := q.Get("min_score"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinScore = n
		}
	}
	if v := q.Get("ref_description"); v != "" {
		f.RefDescription = &v
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Offset = n
		}
	}
	res, err := ListCandidates(r.Context(), h.Store.DB(), info.User.ID, id, f)
	if writeSubError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) ConvertFromRecurring(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	recurringID := chi.URLParam(r, "recurring_id")
	var req convertRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	res, err := ConvertFromRecurring(r.Context(), h.Store.DB(), info.User.ID, recurringID, req.Name, req.Description, req.Icon, req.WebsiteURL)
	if writeSubError(w, r, err) {
		return
	}
	_ = h.Audit.Log("subscription.convert_recurring", info.User.ID, info.User.Login, clientIP(r), map[string]any{
		"subscription_id": res.Subscription.ID, "recurring_id": recurringID,
	})
	writeJSON(w, http.StatusCreated, res)
}

func (h *Handler) E2ERunNow(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("BUHGALTER_E2E") != "1" {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Minute))
	sub, err := getByID(ctx, h.Store.DB(), info.User.ID, id)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	_, _ = sqlcdb.New(h.Store.DB()).MarkSubscriptionRan(ctx, sqlcdb.MarkSubscriptionRanParams{
		NextRunAt: past, LastRunAt: sub.LastRunAt, SubcategoryID: sub.SubcategoryID,
		UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()), ID: id, UserID: info.User.ID,
	})
	tz, err := sqlcdb.New(h.Store.DB()).GetUserTimezone(ctx, info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if tz == "" {
		tz = "Europe/Moscow"
	}
	applied, err := ApplyDue(ctx, h.Store.DB(), info.User.ID, timeutil.NowUTC(), tz)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": applied})
}

func parseInput(req upsertRequest) (Input, error) {
	amount, err := money.ParseRubles(req.Amount)
	if err != nil || amount <= 0 {
		return Input{}, errors.New("некорректная сумма")
	}
	startDate, err := timeutil.ParseUTC(req.StartDate)
	if err != nil {
		return Input{}, errors.New("некорректная дата старта")
	}
	timeLocal := strings.TrimSpace(req.TimeLocal)
	if timeLocal == "" {
		timeLocal = "08:00"
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	return Input{
		Name: req.Name, Amount: amount, Description: req.Description, Icon: req.Icon,
		WebsiteURL: req.WebsiteURL, AccountID: req.AccountID, Period: req.Period,
		Weekday: req.Weekday, DayOfMonth: req.DayOfMonth, StartDate: startDate,
		TimeLocal: timeLocal, Active: active,
	}, nil
}

func writeSubError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrRecurringNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrInvalidPeriod),
		errors.Is(err, ErrInvalidWeekday), errors.Is(err, ErrInvalidDay), errors.Is(err, ErrInvalidTime),
		errors.Is(err, ErrInvalidURL), errors.Is(err, ErrInvalidType):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "VALIDATION_ERROR")
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
