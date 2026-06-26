package debt

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
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type createDebtorRequest struct {
	Name string `json:"name"`
}

type updateDebtorRequest struct {
	Name string `json:"name"`
}

type createDebtRequest struct {
	DebtorID       *string `json:"debtor_id"`
	DebtorName     *string `json:"debtor_name"`
	Direction      string  `json:"direction"`
	Amount         string  `json:"amount"`
	DebtDate       string  `json:"debt_date"`
	DueDate        string  `json:"due_date"`
	AffectsBalance bool    `json:"affects_balance"`
	Description    *string `json:"description"`
	AccountID      string  `json:"account_id"`
}

type settleDebtRequest struct {
	Amount         string `json:"amount"`
	SettledAt      string `json:"settled_at"`
	AffectsBalance bool   `json:"affects_balance"`
	AccountID      string `json:"account_id"`
}

func (h *Handler) GetDebtor(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	detail, err := GetDebtorDetail(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeDebtorError(w, r, err) {
		return
	}
	if detail.Debts == nil {
		detail.Debts = []Debt{}
	}
	if detail.Transactions == nil {
		detail.Transactions = []DebtTransaction{}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) ListDebtors(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	debtors, err := ListDebtors(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if debtors == nil {
		debtors = []Debtor{}
	}
	writeJSON(w, http.StatusOK, debtors)
}

func (h *Handler) CreateDebtor(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req createDebtorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	debtor, err := CreateDebtor(r.Context(), h.Store.DB(), info.User.ID, req.Name)
	if writeDebtorError(w, r, err) {
		return
	}
	_ = h.Audit.Log("debtor.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debtor_id": debtor.ID})
	writeJSON(w, http.StatusCreated, debtor)
}

func (h *Handler) UpdateDebtor(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateDebtorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	debtor, err := UpdateDebtor(r.Context(), h.Store.DB(), info.User.ID, id, req.Name)
	if writeDebtorError(w, r, err) {
		return
	}
	_ = h.Audit.Log("debtor.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debtor_id": id})
	writeJSON(w, http.StatusOK, debtor)
}

func (h *Handler) DeleteDebtor(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := DeleteDebtor(r.Context(), h.Store.DB(), info.User.ID, id); err != nil {
		if writeDebtorError(w, r, err) {
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("debtor.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debtor_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListDebts(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	debts, err := List(r.Context(), h.Store.DB(), info.User.ID, r.URL.Query().Get("settled"))
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if debts == nil {
		debts = []Debt{}
	}
	writeJSON(w, http.StatusOK, debts)
}

func (h *Handler) GetDebt(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	debt, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id)
	if writeDebtError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, debt)
}

func (h *Handler) CreateDebt(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req createDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parseCreateInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	debt, err := Create(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeDebtError(w, r, err) {
		return
	}
	_ = h.Audit.Log("debt.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debt_id": debt.ID})
	writeJSON(w, http.StatusCreated, debt)
}

func (h *Handler) SettleDebt(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req settleDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parseSettleInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	debt, err := Settle(r.Context(), h.Store.DB(), info.User.ID, id, in)
	if writeDebtError(w, r, err) {
		return
	}
	_ = h.Audit.Log("debt.settle", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debt_id": id})
	writeJSON(w, http.StatusOK, debt)
}

func (h *Handler) DeleteDebt(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := Delete(r.Context(), h.Store.DB(), info.User.ID, id); err != nil {
		if writeDebtError(w, r, err) {
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("debt.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"debt_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	summary, err := SummaryForUser(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func parseCreateInput(req createDebtRequest) (CreateInput, error) {
	amount, err := money.ParseRubles(req.Amount)
	if err != nil || amount <= 0 {
		return CreateInput{}, errors.New("некорректная сумма")
	}
	dueDate, err := timeutil.ParseUTC(req.DueDate)
	if err != nil {
		return CreateInput{}, errors.New("некорректная дата возврата")
	}
	debtDate, err := timeutil.ParseUTC(req.DebtDate)
	if err != nil {
		return CreateInput{}, errors.New("некорректная дата операции")
	}
	if req.Direction != "lent" && req.Direction != "borrowed" {
		return CreateInput{}, errors.New("направление: lent или borrowed")
	}
	return CreateInput{
		DebtorID:       req.DebtorID,
		DebtorName:     req.DebtorName,
		Direction:      req.Direction,
		Amount:         amount,
		DebtDate:       debtDate,
		DueDate:        dueDate,
		AffectsBalance: req.AffectsBalance,
		Description:    req.Description,
		AccountID:      req.AccountID,
	}, nil
}

func parseSettleInput(req settleDebtRequest) (SettleInput, error) {
	settledAt, err := timeutil.ParseUTC(req.SettledAt)
	if err != nil {
		return SettleInput{}, errors.New("некорректная дата закрытия")
	}
	var amount int64
	if strings.TrimSpace(req.Amount) != "" {
		amount, err = money.ParseRubles(req.Amount)
		if err != nil || amount <= 0 {
			return SettleInput{}, errors.New("некорректная сумма погашения")
		}
	}
	return SettleInput{
		Amount:         amount,
		SettledAt:      settledAt,
		AffectsBalance: req.AffectsBalance,
		AccountID:      req.AccountID,
	}, nil
}

func writeDebtorError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrDebtorNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrDebtorNameTaken):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_DEBTOR_NAME")
	case errors.Is(err, ErrDebtorHasActiveDebt):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_DEBTOR_ACTIVE_DEBTS")
	case errors.Is(err, ErrInvalidDebtorName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DEBTOR_NAME_REQUIRED")
	default:
		return false
	}
	return true
}

func writeDebtError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDebtorNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrInvalidDirection):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DEBT_DIRECTION")
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_AMOUNT")
	case errors.Is(err, ErrInvalidDueDate):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_DUE_DATE")
	case errors.Is(err, ErrInvalidDebtDate):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_DEBT_DATE")
	case errors.Is(err, ErrAccountRequired):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DEBT_ACCOUNT_REQUIRED")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
	case errors.Is(err, ErrAlreadySettled):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_DEBT_CLOSED")
	case errors.Is(err, ErrInvalidSettleAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SETTLE_AMOUNT")
	case errors.Is(err, ErrPlannedNotAllowed):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SYSTEM_CATEGORY_PLANNED")
	case errors.Is(err, ErrInvalidDebtorName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DEBTOR_REQUIRED")
	case errors.Is(err, ErrCannotBorrowFromDebtor):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_DEBT_CANNOT_BORROW")
	case errors.Is(err, ErrCannotLendToCreditor):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_DEBT_CANNOT_LEND")
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
