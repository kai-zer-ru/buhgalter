package transaction

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type createRequest struct {
	AccountID        string  `json:"account_id"`
	Type             string  `json:"type"`
	Amount           string  `json:"amount"`
	Description      *string `json:"description"`
	CategoryID       *string `json:"category_id"`
	SubcategoryID    *string `json:"subcategory_id"`
	SubcategoryName  *string `json:"subcategory_name"`
	TransactionDate  string  `json:"transaction_date"`
}

type transferRequest struct {
	FromAccountID   string  `json:"from_account_id"`
	ToAccountID     string  `json:"to_account_id"`
	Amount          string  `json:"amount"`
	Commission      *string `json:"commission,omitempty"`
	Description     *string `json:"description"`
	TransactionDate string  `json:"transaction_date"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	result, err := List(r.Context(), h.Store.DB(), info.User.ID, ListFilters{
		AccountID:  q.Get("account_id"),
		Type:       q.Get("type"),
		CategoryID: q.Get("category_id"),
		Kind:       q.Get("kind"),
		From:       q.Get("from"),
		To:         q.Get("to"),
		Search:     q.Get("search"),
		Sort:       q.Get("sort"),
		Page:       page,
		Limit:      limit,
	})
	if errors.Is(err, ErrInvalidDate) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	tx, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, tx)
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
	in, err := parseCreateInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	tx, err := Create(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeTxError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transaction.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"transaction_id": tx.ID})
	writeJSON(w, http.StatusCreated, tx)
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
	in, err := parseCreateInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	tx, err := Update(r.Context(), h.Store.DB(), info.User.ID, id, UpdateInput(in))
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if writeTxError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transaction.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"transaction_id": id})
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	err := Delete(r.Context(), h.Store.DB(), info.User.ID, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if errors.Is(err, debt.ErrLinkedTransactionProtected) {
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "ERR_LINKED_TX_DELETE")
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("transaction.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"transaction_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	tx, err := Activate(r.Context(), h.Store.DB(), info.User.ID, id)
	if errors.Is(err, ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parseTransferInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	tr, err := CreateTransfer(r.Context(), h.Store.DB(), info.User.ID, in)
	if writeTxError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transfer.create", info.User.ID, info.User.Login, clientIP(r), map[string]any{"group_id": tr.GroupID})
	writeJSON(w, http.StatusCreated, tr)
}

func (h *Handler) UpdateTransfer(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	groupID := chi.URLParam(r, "group_id")
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	in, err := parseTransferInput(req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	tr, err := UpdateTransfer(r.Context(), h.Store.DB(), info.User.ID, groupID, in)
	if errors.Is(err, ErrTransferNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if writeTxError(w, r, err) {
		return
	}
	_ = h.Audit.Log("transfer.update", info.User.ID, info.User.Login, clientIP(r), map[string]any{"group_id": groupID})
	writeJSON(w, http.StatusOK, tr)
}

func (h *Handler) DeleteTransfer(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	groupID := chi.URLParam(r, "group_id")
	err := DeleteTransfer(r.Context(), h.Store.DB(), info.User.ID, groupID)
	if errors.Is(err, ErrTransferNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	_ = h.Audit.Log("transfer.delete", info.User.ID, info.User.Login, clientIP(r), map[string]any{"group_id": groupID})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AccountBalance(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	row, err := queries(h.Store.DB()).GetAccountByID(r.Context(), sqlcdb.GetAccountByIDParams{ID: id, UserID: info.User.ID})
	if errors.Is(err, sql.ErrNoRows) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	ab, err := EnrichAccountBalance(r.Context(), h.Store.DB(), info.User.ID, row.ID, row.Name, row.Type, row.BankIcon, row.InitialBalance)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, ab)
}

func (h *Handler) AccountsSummary(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	summary, err := AccountsSummaryForUser(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	dash, err := DashboardForUser(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

func parseCreateInput(req createRequest) (CreateInput, error) {
	amount, err := money.ParseRubles(req.Amount)
	if err != nil || amount <= 0 {
		return CreateInput{}, errors.New("некорректная сумма")
	}
	txDate, err := timeutil.ParseUTC(req.TransactionDate)
	if err != nil {
		return CreateInput{}, errors.New("некорректная дата операции")
	}
	return CreateInput{
		AccountID:       req.AccountID,
		Type:            req.Type,
		Amount:          amount,
		Description:     req.Description,
		CategoryID:      req.CategoryID,
		SubcategoryID:   req.SubcategoryID,
		SubcategoryName: req.SubcategoryName,
		TransactionDate: txDate,
	}, nil
}

func parseTransferInput(req transferRequest) (TransferInput, error) {
	amount, err := money.ParseRubles(req.Amount)
	if err != nil || amount <= 0 {
		return TransferInput{}, errors.New("некорректная сумма")
	}
	var commission int64
	if req.Commission != nil && strings.TrimSpace(*req.Commission) != "" {
		commission, err = money.ParseRubles(*req.Commission)
		if err != nil || commission < 0 {
			return TransferInput{}, errors.New("некорректная комиссия")
		}
	}
	txDate, err := timeutil.ParseUTC(req.TransactionDate)
	if err != nil {
		return TransferInput{}, errors.New("некорректная дата операции")
	}
	return TransferInput{
		FromAccountID:   req.FromAccountID,
		ToAccountID:     req.ToAccountID,
		Amount:          amount,
		Commission:      commission,
		Description:     req.Description,
		TransactionDate: txDate,
	}, nil
}

func writeTxError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrInvalidType):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_TYPE")
	case errors.Is(err, ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_AMOUNT_POSITIVE")
	case errors.Is(err, ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
	case errors.Is(err, ErrInvalidCategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_NOT_FOUND")
	case errors.Is(err, ErrCategoryTypeMatch):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CATEGORY_TYPE_MISMATCH")
	case errors.Is(err, ErrSystemCategoryPlanned):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SYSTEM_CATEGORY_PLANNED")
	case errors.Is(err, credit.ErrCannotEditPayment):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CANNOT_EDIT_PAYMENT")
	case errors.Is(err, ErrInvalidSubcategory):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SUBCATEGORY_NOT_FOUND")
	case errors.Is(err, ErrSameAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TRANSFER_SAME_ACCOUNT")
	default:
		if strings.Contains(err.Error(), "transfer endpoint") {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_USE_TRANSFERS_ENDPOINT")
		} else {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		}
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
