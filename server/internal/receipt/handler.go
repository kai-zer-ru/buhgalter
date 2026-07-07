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
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
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

func (h *Handler) fetcher() Fetcher {
	if h.Fetcher != nil {
		return h.Fetcher
	}
	return fns.NewClient()
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
	out, err := Parse(r.Context(), h.fetcher(), qr)
	if writeReceiptError(w, r, err) {
		return
	}
	_ = info
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
		items = append(items, ImportItemInput{
			Index:         it.Index,
			CategoryID:    it.CategoryID,
			SubcategoryID: it.SubcategoryID,
			Include:       it.Include,
		})
	}
	result, err := Import(r.Context(), h.Store.DB(), info.User.ID, ImportInput{
		AccountID:            strings.TrimSpace(req.AccountID),
		FiscalDriveNumber:    strings.TrimSpace(req.FiscalDriveNumber),
		FiscalDocumentNumber: strings.TrimSpace(req.FiscalDocumentNumber),
		FiscalSign:           strings.TrimSpace(req.FiscalSign),
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

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	txID := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	out, err := GetByID(r.Context(), h.Store.DB(), info.User.ID, id, txID)
	if writeReceiptError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func writeReceiptError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
