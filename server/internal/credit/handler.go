package credit

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

type createCreditRequest struct {
	Name                      *string               `json:"name"`
	CreditKind                string                `json:"credit_kind"`
	PrincipalAmount           json.RawMessage       `json:"principal_amount"`
	PropertyPrice             json.RawMessage       `json:"property_price"`
	DownPayment               json.RawMessage       `json:"down_payment"`
	DownPaymentAffectsBalance *bool                 `json:"down_payment_affects_balance"`
	DownPaymentAccountID      *string               `json:"down_payment_account_id"`
	IssueDate                 string                `json:"issue_date"`
	TermMonths                int                   `json:"term_months"`
	InterestRate              float64               `json:"interest_rate"`
	PaymentInterval           string                `json:"payment_interval"`
	PaidAmount                json.RawMessage       `json:"paid_amount"`
	MonthlyPayment            json.RawMessage       `json:"monthly_payment"`
	DebitAccountID            string                `json:"debit_account_id"`
	DebitTimeLocal            *string               `json:"debit_time_local"`
	BankID                    *string               `json:"bank_id"`
	AddedRetroactively        *bool                 `json:"added_retroactively"`
	RetroactiveDebitCount     *int                  `json:"retroactive_debit_count"`
	CreateTransactions        *bool                 `json:"create_transactions"`
	ScheduleSeed              []scheduleSeedRequest `json:"schedule_seed"`
}

type scheduleSeedRequest struct {
	PaymentDate string          `json:"payment_date"`
	Amount      json.RawMessage `json:"amount"`
}

type updateCreditRequest struct {
	Name           *string         `json:"name"`
	MonthlyPayment json.RawMessage `json:"monthly_payment"`
	DebitAccountID *string         `json:"debit_account_id"`
	DebitTimeLocal *string         `json:"debit_time_local"`
	BankID         json.RawMessage `json:"bank_id"`
}

type payPaymentRequest struct {
	Amount      json.RawMessage `json:"amount"`
	PaymentDate string          `json:"payment_date"`
}

type completeCreditRequest struct {
	AffectsBalance *bool  `json:"affects_balance"`
	PaymentDate    string `json:"payment_date"`
}

type updateScheduleRequest struct {
	Payments []scheduleAmountUpdateRequest `json:"payments"`
}

type scheduleAmountUpdateRequest struct {
	ID     string          `json:"id"`
	Amount json.RawMessage `json:"amount"`
}

type previewScheduleRequest struct {
	Principal       json.RawMessage       `json:"principal"`
	CreditKind      string                `json:"credit_kind"`
	TermMonths      int                   `json:"term"`
	InterestRate    float64               `json:"interest_rate"`
	PaymentInterval string                `json:"payment_interval"`
	IssueDate       string                `json:"issue_date"`
	MonthlyPayment  json.RawMessage       `json:"monthly_payment"`
	SeedPayments    []scheduleSeedRequest `json:"seed_payments"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "active" && status != "closed" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_STATUS")
		return
	}
	credits, err := List(r.Context(), h.Store.DB(), info.User.ID, status)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if credits == nil {
		credits = []Credit{}
	}
	writeJSON(w, http.StatusOK, credits)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	c, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id, true)
	if writeCreditError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req createCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parseCreateInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	c, err := Create(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"credit_id": c.ID})
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in := UpdateInput{Name: req.Name, DebitAccountID: req.DebitAccountID, DebitTimeLocal: req.DebitTimeLocal}
	if len(req.MonthlyPayment) > 0 && string(req.MonthlyPayment) != "null" {
		amt, err := money.ParseAmount(req.MonthlyPayment)
		if err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_PAYMENT")
			return
		}
		in.MonthlyPayment = &amt
	}
	if len(req.BankID) > 0 {
		in.BankIDSet = true
		if string(req.BankID) != "null" {
			var bankID string
			if err := json.Unmarshal(req.BankID, &bankID); err != nil {
				apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
				return
			}
			in.BankID = &bankID
		}
	}
	c, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, in)
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"credit_id": id})
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) AddPayment(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req payPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	amount, err := money.ParseAmount(req.Amount)
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_AMOUNT")
		return
	}
	payDate, err := timeutil.ParseUTC(req.PaymentDate)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	c, err := PayNextScheduled(r.Context(), h.Store.DB(), info.User.ID, id, PayPaymentInput{
		Amount: amount, PaymentDate: payDate,
	})
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.pay", info.User.ID, info.User.Login, clientIP(r), map[string]any{"credit_id": id, "amount": amount})
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	creditID := chi.URLParam(r, "id")
	paymentID := chi.URLParam(r, "paymentId")
	c, err := RemovePayment(r.Context(), h.Store.DB(), info.User.ID, creditID, paymentID)
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.payment.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{
		"credit_id": creditID, "payment_id": paymentID,
	})
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req completeCreditRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
			return
		}
	}
	affectsBalance := true
	if req.AffectsBalance != nil {
		affectsBalance = *req.AffectsBalance
	}
	var payDate time.Time
	if req.PaymentDate != "" {
		parsed, err := timeutil.ParseUTC(req.PaymentDate)
		if err != nil {
			apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
			return
		}
		payDate = parsed
	}
	c, err := Complete(r.Context(), h.Store.DB(), info.User.ID, id, CompleteInput{
		AffectsBalance: affectsBalance,
		PaymentDate:    payDate,
	})
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.complete", info.User.ID, info.User.Login, clientIP(r), map[string]any{
		"credit_id": id, "affects_balance": affectsBalance,
	})
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	mode := r.URL.Query().Get("mode")
	if err := Delete(r.Context(), h.Store.DB(), info.User.ID, id, mode); writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"credit_id": id, "mode": mode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Schedule(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	schedule, err := loadSchedule(r.Context(), h.Store.DB(), id)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if _, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id, false); writeCreditError(w, r, err) {
		return
	}
	if schedule == nil {
		schedule = []CreditPayment{}
	}
	writeJSON(w, http.StatusOK, schedule)
}

func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	updates := make([]ScheduleAmountUpdate, 0, len(req.Payments))
	for _, p := range req.Payments {
		if p.ID == "" {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
			return
		}
		amount, err := money.ParseAmount(p.Amount)
		if err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_AMOUNT")
			return
		}
		updates = append(updates, ScheduleAmountUpdate{PaymentID: p.ID, Amount: amount})
	}
	c, err := UpdateScheduleAmounts(r.Context(), h.Store.DB(), info.User.ID, id, updates)
	if writeCreditError(w, r, err) {
		return
	}
	_ = h.Audit.Log("credit.schedule.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{
		"credit_id": id, "count": len(updates),
	})
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) PreviewSchedule(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req previewScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parsePreviewInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	entries, monthly, err := PreviewSchedule(in)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schedule_preview":                   entries,
		"calculated_monthly_payment":         monthly,
		"calculated_monthly_payment_display": money.FormatRubles(monthly),
	})
}

func parseCreateInput(req createCreditRequest) (CreateInput, error) {
	principal, err := money.ParseAmount(req.PrincipalAmount)
	if err != nil {
		return CreateInput{}, errors.New("некорректная сумма кредита")
	}
	paid := int64(0)
	if len(req.PaidAmount) > 0 && string(req.PaidAmount) != "null" {
		paid, err = money.ParseAmount(req.PaidAmount)
		if err != nil {
			return CreateInput{}, errors.New("некорректная уплаченная сумма")
		}
	}
	issueDate, err := timeutil.ParseUTC(req.IssueDate)
	if err != nil {
		return CreateInput{}, err
	}
	interval := PaymentInterval(req.PaymentInterval)
	if interval == "" {
		interval = IntervalMonth
	}
	if !interval.Valid() {
		return CreateInput{}, errors.New("некорректная периодичность платежей")
	}
	if req.TermMonths <= 0 {
		return CreateInput{}, errors.New("срок должен быть положительным")
	}
	if req.DebitAccountID == "" {
		return CreateInput{}, errors.New("укажите счёт списания")
	}

	createTx := true
	if req.CreateTransactions != nil {
		createTx = *req.CreateTransactions
	}

	in := CreateInput{
		Name: req.Name, CreditKind: req.CreditKind, PrincipalAmount: principal, IssueDate: issueDate,
		TermMonths: req.TermMonths, InterestRate: req.InterestRate,
		PaymentInterval: interval, PaidAmount: paid, DebitAccountID: req.DebitAccountID,
		DebitTimeLocal: req.DebitTimeLocal, BankID: req.BankID,
		AddedRetroactively: req.AddedRetroactively, CreateTransactions: createTx,
	}
	if len(req.PropertyPrice) > 0 && string(req.PropertyPrice) != "null" {
		propertyPrice, err := money.ParseAmount(req.PropertyPrice)
		if err != nil {
			return CreateInput{}, errors.New("некорректная цена объекта")
		}
		in.PropertyPrice = &propertyPrice
	}
	if len(req.DownPayment) > 0 && string(req.DownPayment) != "null" {
		downPayment, err := money.ParseAmount(req.DownPayment)
		if err != nil {
			return CreateInput{}, errors.New("некорректный первоначальный взнос")
		}
		in.DownPayment = downPayment
	}
	if req.DownPaymentAffectsBalance != nil {
		in.DownPaymentAffectsBalance = *req.DownPaymentAffectsBalance
	}
	if req.DownPaymentAccountID != nil {
		in.DownPaymentAccountID = req.DownPaymentAccountID
	}
	if req.RetroactiveDebitCount != nil {
		in.RetroactiveDebitCount = *req.RetroactiveDebitCount
	}
	if len(req.MonthlyPayment) > 0 && string(req.MonthlyPayment) != "null" {
		mp, err := money.ParseAmount(req.MonthlyPayment)
		if err != nil {
			return CreateInput{}, errors.New("некорректный ежемесячный платёж")
		}
		in.MonthlyPayment = &mp
	}
	for _, seed := range req.ScheduleSeed {
		d, err := timeutil.ParseUTC(seed.PaymentDate)
		if err != nil {
			return CreateInput{}, err
		}
		amt, err := money.ParseAmount(seed.Amount)
		if err != nil {
			return CreateInput{}, errors.New("некорректная сумма в графике")
		}
		in.ScheduleSeed = append(in.ScheduleSeed, ScheduleSeed{PaymentDate: d, Amount: amt})
	}
	if interval == IntervalManual && len(in.ScheduleSeed) != req.TermMonths {
		return CreateInput{}, errors.New("укажите даты и суммы для всех платежей")
	}
	return in, nil
}

func parsePreviewInput(req previewScheduleRequest) (PreviewInput, error) {
	principal, err := money.ParseAmount(req.Principal)
	if err != nil {
		return PreviewInput{}, errors.New("некорректная сумма")
	}
	issueDate, err := timeutil.ParseUTC(req.IssueDate)
	if err != nil {
		return PreviewInput{}, err
	}
	interval := PaymentInterval(req.PaymentInterval)
	if interval == "" {
		interval = IntervalMonth
	}
	term := req.TermMonths
	if term <= 0 {
		return PreviewInput{}, errors.New("укажите срок")
	}
	in := PreviewInput{
		Principal: principal, CreditKind: req.CreditKind, TermMonths: req.TermMonths, InterestRate: req.InterestRate,
		PaymentInterval: interval, IssueDate: issueDate,
	}
	if len(req.MonthlyPayment) > 0 && string(req.MonthlyPayment) != "null" {
		mp, err := money.ParseAmount(req.MonthlyPayment)
		if err != nil {
			return PreviewInput{}, errors.New("некорректный платёж")
		}
		in.MonthlyPayment = &mp
	}
	for _, seed := range req.SeedPayments {
		d, err := timeutil.ParseUTC(seed.PaymentDate)
		if err != nil {
			return PreviewInput{}, err
		}
		amt, err := money.ParseAmount(seed.Amount)
		if err != nil {
			return PreviewInput{}, errors.New("некорректная сумма seed")
		}
		in.SeedPayments = append(in.SeedPayments, ScheduleSeed{PaymentDate: d, Amount: amt})
	}
	if interval == IntervalManual && len(in.SeedPayments) != term {
		return PreviewInput{}, errors.New("укажите даты и суммы для всех платежей")
	}
	return in, nil
}

func writeCreditError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_AMOUNT")
	case errors.Is(err, ErrInvalidTerm):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_TERM")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
	case errors.Is(err, ErrAlreadyClosed):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CLOSED")
	case errors.Is(err, ErrInvalidInterval):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_INTERVAL")
	case errors.Is(err, ErrInvalidDebitTime):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_DEBIT_TIME")
	case errors.Is(err, ErrInvalidBank):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_BANK_NOT_FOUND")
	case errors.Is(err, ErrPlannedNotAllowed):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SYSTEM_CATEGORY_PLANNED")
	case errors.Is(err, ErrInvalidCreditKind):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_KIND")
	case errors.Is(err, ErrInvalidMortgageFields):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_MORTGAGE")
	case errors.Is(err, ErrCreditBankLocked):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_CREDIT_BANK_LOCKED")
	case errors.Is(err, ErrNoPendingPayment):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_NO_PENDING_PAYMENT")
	case errors.Is(err, ErrCannotRemoveRetroactive):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CANNOT_REMOVE_RETRO")
	case errors.Is(err, ErrOnlyLatestPaymentDelete):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_ONLY_LATEST_DELETE")
	case errors.Is(err, ErrInvalidRetroactiveDebit):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_INVALID_RETRO_DEBIT")
	case errors.Is(err, ErrCannotEditPayment):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CANNOT_EDIT_PAYMENT")
	case errors.Is(err, ErrCompleteParamsRequired):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_COMPLETE_DATE")
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
	return r.RemoteAddr
}
