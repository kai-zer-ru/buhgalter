package recurring

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
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

type createRequest struct {
	Type          string  `json:"type"`
	Amount        string  `json:"amount"`
	Description   *string `json:"description"`
	AccountID     string  `json:"account_id"`
	CategoryID    string  `json:"category_id"`
	SubcategoryID *string `json:"subcategory_id"`
	Period        string  `json:"period"`
	Weekday       *int64  `json:"weekday"`
	DayOfMonth    *int64  `json:"day_of_month"`
	StartDate     string  `json:"start_date"`
	TimeLocal     string  `json:"time_local"`
	Active        *bool   `json:"active"`
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req createRequest
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
	if writeRecurringError(w, r, err) {
		return
	}
	_ = h.Audit.Log("recurring.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID})
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req createRequest
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
	if writeRecurringError(w, r, err) {
		return
	}
	_ = h.Audit.Log("recurring.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": item.ID})
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := Delete(r.Context(), h.Store.DB(), info.User.ID, id); errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	} else if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("recurring.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

// E2ERunNow forces a recurring operation to run immediately (BUHGALTER_E2E=1 only).
func (h *Handler) E2ERunNow(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("BUHGALTER_E2E") != "1" {
		http.NotFound(w, r)
		return
	}
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Hour))
	n, err := sqlcdb.New(h.Store.DB()).SetRecurringOperationNextRunAt(ctx, sqlcdb.SetRecurringOperationNextRunAtParams{
		NextRunAt: past,
		ID:        id,
		UserID:    info.User.ID,
	})
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if n == 0 {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	tz, err := userTimezone(ctx, h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	applied, err := ApplyDue(ctx, h.Store.DB(), info.User.ID, timeutil.NowUTC(), tz)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": applied})
}

func parseInput(req createRequest) (Input, error) {
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
		Type:          req.Type,
		Amount:        amount,
		Description:   req.Description,
		AccountID:     req.AccountID,
		CategoryID:    req.CategoryID,
		SubcategoryID: req.SubcategoryID,
		Period:        req.Period,
		Weekday:       req.Weekday,
		DayOfMonth:    req.DayOfMonth,
		StartDate:     startDate,
		TimeLocal:     timeLocal,
		Active:        active,
	}, nil
}

func writeRecurringError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrInvalidType):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_TYPE")
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_AMOUNT_POSITIVE")
	case errors.Is(err, ErrInvalidPeriod), errors.Is(err, ErrInvalidWeekday), errors.Is(err, ErrInvalidDay), errors.Is(err, ErrInvalidTime):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "VALIDATION_ERROR")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
	case errors.Is(err, ErrInvalidCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_NOT_FOUND")
	case errors.Is(err, ErrInvalidSub):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SUBCATEGORY_NOT_FOUND")
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
