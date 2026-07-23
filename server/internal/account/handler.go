package account

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

type createRequest struct {
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	BankID           *string `json:"bank_id"`
	InitialBalance   string  `json:"initial_balance"`
	CreditLimit      *string `json:"credit_limit"`
	PaymentAccountID *string `json:"payment_account_id"`
}

type updateRequest struct {
	Name                     string  `json:"name"`
	BankID                   *string `json:"bank_id"`
	InitialBalance           *string `json:"initial_balance"`
	CreditLimit              *string `json:"credit_limit"`
	PaymentAccountID         *string `json:"payment_account_id"`
	AutoTopupEnabled         *bool   `json:"auto_topup_enabled,omitempty"`
	AutoTopupThreshold       *string `json:"auto_topup_threshold,omitempty"`
	AutoTopupTarget          *string `json:"auto_topup_target,omitempty"`
	AutoTopupSourceAccountID *string `json:"auto_topup_source_account_id,omitempty"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	status := r.URL.Query().Get("status")
	accounts, err := ListByUser(r.Context(), h.Store.DB(), info.User.ID, status)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if accounts == nil {
		accounts = []Account{}
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	acc, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, acc)
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
	balance, err := money.ParseRubles(req.InitialBalance)
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_INVALID_BALANCE")
		return
	}
	creditLimit, err := parseOptionalRubles(req.CreditLimit)
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_INVALID_CREDIT_LIMIT")
		return
	}
	acc, err := Create(r.Context(), h.Store.DB(), info.User.ID, CreateInput{
		Name:             strings.TrimSpace(req.Name),
		Type:             req.Type,
		BankID:           req.BankID,
		InitialBalance:   balance,
		CreditLimit:      creditLimit,
		PaymentAccountID: req.PaymentAccountID,
	})
	if err := writeAccountError(w, r, err); err != nil {
		return
	}
	_ = h.Audit.Log("account.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": acc.ID})
	writeJSON(w, http.StatusCreated, acc)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	var balancePtr *int64
	if req.InitialBalance != nil {
		b, err := money.ParseRubles(*req.InitialBalance)
		if err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_INVALID_BALANCE")
			return
		}
		balancePtr = &b
	}
	creditLimit, err := parseOptionalRubles(req.CreditLimit)
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_INVALID_CREDIT_LIMIT")
		return
	}
	acc, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, UpdateInput{
		Name:             strings.TrimSpace(req.Name),
		BankID:           req.BankID,
		InitialBalance:   balancePtr,
		CreditLimit:      creditLimit,
		PaymentAccountID: req.PaymentAccountID,
		AutoTopup:        parseAutoTopupInput(req),
	})
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err := writeAccountError(w, r, err); err != nil {
		return
	}
	if req.AutoTopupEnabled != nil && *req.AutoTopupEnabled && AfterAutoTopupConfigured != nil {
		AfterAutoTopupConfigured(r.Context(), h.Store.DB(), info.User.ID, id)
	}
	_ = h.Audit.Log("account.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": id})
	writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "archived", "account.archive")
}

func (h *Handler) Unarchive(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "active", "account.unarchive")
}

func (h *Handler) SetPrimary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	acc, err := SetPrimary(r.Context(), h.Store.DB(), info.User.ID, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, ErrArchived) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_PRIMARY_ARCHIVED")
		return
	}
	if errors.Is(err, ErrCreditCardPrimary) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_PRIMARY_CREDIT_CARD")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("account.set_primary", info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": id})
	writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status, auditAction string) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	acc, err := SetStatus(r.Context(), h.Store.DB(), info.User.ID, id, status)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, ErrCreditCardArchiveNotFullyPaid) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log(auditAction, info.User.ID, info.User.Login, clientIP(r), map[string]any{"account_id": id})
	writeJSON(w, http.StatusOK, acc)
}

func writeAccountError(w http.ResponseWriter, r *http.Request, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidName):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NAME_LENGTH")
	case errors.Is(err, ErrNameTaken):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "CONFLICT_ACCOUNT_NAME")
	case errors.Is(err, ErrInvalidType):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_TYPE")
	case errors.Is(err, ErrBankRequired):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_BANK_REQUIRED")
	case errors.Is(err, ErrBankForbidden):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_BANK_FORBIDDEN")
	case errors.Is(err, ErrBankNotFound):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_BANK_NOT_FOUND")
	case errors.Is(err, ErrCreditLimitRequired), errors.Is(err, ErrInvalidCreditLimit):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_CREDIT_LIMIT_REQUIRED")
	case errors.Is(err, ErrCreditLimitForbidden):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_CREDIT_LIMIT_FORBIDDEN")
	case errors.Is(err, ErrInvalidPaymentAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_INVALID_PAYMENT_ACCOUNT")
	case errors.Is(err, ErrAutoTopupNotAllowed):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_AUTO_TOPUP_NOT_ALLOWED")
	case errors.Is(err, ErrInvalidAutoTopupThreshold):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_AUTO_TOPUP_THRESHOLD")
	case errors.Is(err, ErrInvalidAutoTopupTarget):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_AUTO_TOPUP_TARGET")
	case errors.Is(err, ErrInvalidAutoTopupSource):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_AUTO_TOPUP_SOURCE")
	case errors.Is(err, ErrArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED_EDIT")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return err
}

func parseOptionalRubles(s *string) (*int64, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	v, err := money.ParseRubles(*s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseAutoTopupInput(req updateRequest) *AutoTopupInput {
	if req.AutoTopupEnabled == nil {
		return nil
	}
	in := AutoTopupInput{Enabled: *req.AutoTopupEnabled}
	if req.AutoTopupThreshold != nil {
		if v, err := money.ParseRubles(*req.AutoTopupThreshold); err == nil {
			in.Threshold = v
		}
	}
	if req.AutoTopupTarget != nil {
		if v, err := money.ParseRubles(*req.AutoTopupTarget); err == nil {
			in.Target = v
		}
	}
	if req.AutoTopupSourceAccountID != nil {
		in.SourceAccountID = strings.TrimSpace(*req.AutoTopupSourceAccountID)
	}
	return &in
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
