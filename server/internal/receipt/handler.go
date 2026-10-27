package receipt

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/locale"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

type Handler struct {
	Store   *db.Handle
	Audit   *audit.Logger
	Fetcher Fetcher
	Logger  *slog.Logger
}

type parseRequest struct {
	QR string `json:"qr"`
}

type importItemRequest struct {
	Index         int     `json:"index"`
	CategoryID    *string `json:"category_id"`
	SubcategoryID *string `json:"subcategory_id"`
	Include       bool    `json:"include"`
}

type importRequest struct {
	AccountID            string              `json:"account_id"`
	FiscalDriveNumber    string              `json:"fiscal_drive_number"`
	FiscalDocumentNumber string              `json:"fiscal_document_number"`
	FiscalSign           string              `json:"fiscal_sign"`
	Receipt              fns.ParsedReceipt   `json:"receipt"`
	Items                []importItemRequest `json:"items"`
}

type attachRequest struct {
	TransactionID          string              `json:"transaction_id"`
	FiscalDriveNumber      string              `json:"fiscal_drive_number"`
	FiscalDocumentNumber   string              `json:"fiscal_document_number"`
	FiscalSign             string              `json:"fiscal_sign"`
	Receipt                fns.ParsedReceipt   `json:"receipt"`
	Items                  []importItemRequest `json:"items"`
	AcknowledgeSumMismatch bool                `json:"acknowledge_sum_mismatch"`
}

func (h *Handler) fetcher() Fetcher {
	if h.Fetcher != nil {
		return h.Fetcher
	}
	return TokenFetcher{DB: h.Store.DB()}
}

func (h *Handler) Parse(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req parseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	qr := strings.TrimSpace(req.QR)
	if qr == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_QR")
		return
	}
	out, err := ParseForUser(r.Context(), h.Store.DB(), info.User.ID, qr, h.fetcher())
	if writeReceiptError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	items := make([]ImportItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, ImportItemInput(it))
	}
	fn, fd, fp := NormalizeFiscalKey(req.FiscalDriveNumber, req.FiscalDocumentNumber, req.FiscalSign, req.Receipt)
	accountID := strings.TrimSpace(req.AccountID)
	ownerID, err := scope.PrincipalForAccount(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), accountID)
	if err != nil {
		if errors.Is(err, scope.ErrNotFound) {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	result, err := Import(r.Context(), h.Store.DB(), ownerID, ImportInput{
		AccountID:            accountID,
		FiscalDriveNumber:    fn,
		FiscalDocumentNumber: fd,
		FiscalSign:           fp,
		Receipt:              req.Receipt,
		Items:                items,
	})
	if writeReceiptError(w, r, err) {
		return
	}
	_ = h.Audit.Log("receipt.import", result.ReceiptImportID, "", auth.ClientIP(r), map[string]any{
		"transactions": len(result.Transactions),
	})
	writeJSON(w, http.StatusOK, result)
}

type detachRequest struct {
	TransactionID string `json:"transaction_id"`
}

func (h *Handler) Detach(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req detachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	txID := strings.TrimSpace(req.TransactionID)
	if txID == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_DETACH_TRANSACTION")
		return
	}
	txRow, err := transaction.GetOwned(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), info.User.ID, txID)
	if err != nil {
		writeReceiptError(w, r, err)
		return
	}
	result, err := Detach(r.Context(), h.Store.DB(), txRow.OwnerUserID, txID)
	if writeReceiptError(w, r, err) {
		return
	}
	_ = h.Audit.Log("receipt.detach", txID, info.User.Login, auth.ClientIP(r), map[string]any{
		"transaction_id": txID,
	})
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req attachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	txID := strings.TrimSpace(req.TransactionID)
	if txID == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_ATTACH_TRANSACTION")
		return
	}
	items := make([]ImportItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, ImportItemInput(it))
	}
	txRow, err := transaction.GetOwned(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), info.User.ID, txID)
	if err != nil {
		writeReceiptError(w, r, err)
		return
	}
	result, err := Attach(r.Context(), h.Store.DB(), txRow.OwnerUserID, AttachInput{
		TransactionID:          txID,
		FiscalDriveNumber:      firstNonEmpty(req.FiscalDriveNumber, req.Receipt.FiscalDriveNumber),
		FiscalDocumentNumber:   firstNonEmpty(req.FiscalDocumentNumber, req.Receipt.FiscalDocumentNumber),
		FiscalSign:             firstNonEmpty(req.FiscalSign, req.Receipt.FiscalSign),
		Receipt:                req.Receipt,
		Items:                  items,
		AcknowledgeSumMismatch: req.AcknowledgeSumMismatch,
	})
	if writeReceiptError(w, r, err) {
		return
	}
	_ = h.Audit.Log("receipt.attach", result.ReceiptImportID, info.User.Login, auth.ClientIP(r), map[string]any{
		"transaction_id": txID,
	})
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	txID := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	out, err := GetByIDForOwners(r.Context(), h.Store.DB(), info.Scope.OwnerIDs(), id, txID)
	if writeReceiptError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func writeReceiptError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var mismatch *SumMismatchError
	switch {
	case errors.Is(err, ErrAPINotConfigured):
		writeAPINotConfigured(w, r)
	case errors.As(err, &mismatch):
		writeSumMismatch(w, r, mismatch)
	case errors.Is(err, ErrAttachNotExpense):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_ATTACH_NOT_EXPENSE")
	case errors.Is(err, ErrAlreadyAttached):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "ERR_RECEIPT_ALREADY_ATTACHED")
	case errors.Is(err, ErrNotAttached):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound, "ERR_RECEIPT_NOT_ATTACHED")
	case errors.Is(err, ErrInvalidFiscalKey):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_QR")
	case errors.Is(err, fns.ErrInvalidQR):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_QR")
	case errors.Is(err, fns.ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound, "ERR_RECEIPT_NOT_FOUND")
	case errors.Is(err, fns.ErrUnavailable), errors.Is(err, fns.ErrFetchFailed):
		apperror.WriteR(w, r, http.StatusBadGateway, apperror.ServiceUnavailable, "ERR_FNS_UNAVAILABLE")
	case errors.Is(err, ErrDuplicate):
		apperror.WriteR(w, r, http.StatusConflict, apperror.Conflict, "ERR_RECEIPT_DUPLICATE")
	case errors.Is(err, ErrNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound, "ERR_RECEIPT_NOT_FOUND")
	case errors.Is(err, ErrNoItemsSelected), errors.Is(err, ErrInvalidItem):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_ITEMS")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return true
}

type sumMismatchBody struct {
	Error struct {
		Code              string `json:"code"`
		Message           string `json:"message"`
		TransactionAmount int64  `json:"transaction_amount"`
		ReceiptTotalSum   int64  `json:"receipt_total_sum"`
	} `json:"error"`
}

func writeSumMismatch(w http.ResponseWriter, r *http.Request, e *SumMismatchError) {
	const code = "ERR_RECEIPT_SUM_MISMATCH"
	var body sumMismatchBody
	body.Error.Code = code
	body.Error.Message = locale.T(r, code, code)
	body.Error.TransactionAmount = e.TransactionAmount
	body.Error.ReceiptTotalSum = e.ReceiptTotalSum
	writeJSON(w, http.StatusConflict, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
