package receipt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/store"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

// SumMismatchError is returned when receipt total differs from the transaction amount.
type SumMismatchError struct {
	TransactionAmount int64
	ReceiptTotalSum   int64
}

func (e SumMismatchError) Error() string {
	return "receipt sum mismatch"
}

type AttachInput struct {
	TransactionID          string
	FiscalDriveNumber      string
	FiscalDocumentNumber   string
	FiscalSign             string
	Receipt                fns.ParsedReceipt
	Items                  []ImportItemInput
	AcknowledgeSumMismatch bool
}

type AttachResult struct {
	ReceiptImportID string                  `json:"receipt_import_id"`
	Transaction     transaction.Transaction `json:"transaction"`
}

func Attach(ctx context.Context, db *sql.DB, userID string, in AttachInput) (AttachResult, error) {
	txID := strings.TrimSpace(in.TransactionID)
	if txID == "" {
		return AttachResult{}, transaction.ErrNotFound
	}
	txRow, err := transaction.GetByID(ctx, db, userID, txID)
	if errors.Is(err, transaction.ErrNotFound) {
		return AttachResult{}, ErrNotFound
	}
	if err != nil {
		return AttachResult{}, err
	}
	if txRow.Type != "expense" {
		return AttachResult{}, ErrAttachNotExpense
	}
	if txRow.TransferGroupID != nil && strings.TrimSpace(*txRow.TransferGroupID) != "" {
		return AttachResult{}, ErrAttachNotExpense
	}
	if txRow.ReceiptImportID != nil && strings.TrimSpace(*txRow.ReceiptImportID) != "" {
		return AttachResult{}, ErrAlreadyAttached
	}
	if txRow.CategoryIsSystem {
		return AttachResult{}, ErrAttachNotExpense
	}

	if err := validateAttachItems(ctx, db, userID, in.Receipt, in.Items); err != nil {
		return AttachResult{}, err
	}

	receiptTotal := in.Receipt.TotalSum
	if txRow.Amount != receiptTotal && !in.AcknowledgeSumMismatch {
		return AttachResult{}, &SumMismatchError{
			TransactionAmount: txRow.Amount,
			ReceiptTotalSum:   receiptTotal,
		}
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return AttachResult{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	receiptID, storeID, err := ensureReceiptImport(ctx, dbTx, q, userID, in.FiscalDriveNumber, in.FiscalDocumentNumber, in.FiscalSign, in.Receipt)
	if err != nil {
		return AttachResult{}, err
	}

	var merchantID *string
	if txRow.MerchantID == nil || strings.TrimSpace(*txRow.MerchantID) == "" {
		merchantID, err = resolveReceiptMerchant(ctx, dbTx, userID, in.Receipt.StoreName)
		if err != nil {
			return AttachResult{}, err
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := q.AttachReceiptToTransaction(ctx, sqlcdb.AttachReceiptToTransactionParams{
		ReceiptImportID: &receiptID,
		StoreID:         storeID,
		MerchantID:      merchantID,
		UpdatedAt:       now,
		ID:              txID,
		UserID:          userID,
	})
	if err != nil {
		return AttachResult{}, err
	}
	if rows == 0 {
		return AttachResult{}, ErrNotFound
	}

	if err := saveReceiptItemCategories(ctx, dbTx, q, userID, receiptID, in.Receipt, in.Items); err != nil {
		return AttachResult{}, err
	}

	if err := dbTx.Commit(); err != nil {
		return AttachResult{}, err
	}

	updated, err := transaction.GetByID(ctx, db, userID, txID)
	if err != nil {
		return AttachResult{}, err
	}
	return AttachResult{ReceiptImportID: receiptID, Transaction: updated}, nil
}

func ensureReceiptImport(
	ctx context.Context,
	dbTx *sql.Tx,
	q *sqlcdb.Queries,
	userID, fn, fd, fp string,
	receipt fns.ParsedReceipt,
) (receiptID string, storeID *string, err error) {
	fn = strings.TrimSpace(fn)
	fd = strings.TrimSpace(fd)
	fp = strings.TrimSpace(fp)
	if fn == "" || fd == "" || fp == "" {
		return "", nil, ErrInvalidFiscalKey
	}

	existingID, err := q.GetReceiptImportByFiscalKey(ctx, sqlcdb.GetReceiptImportByFiscalKeyParams{
		UserID:               userID,
		FiscalDriveNumber:    fn,
		FiscalDocumentNumber: fd,
		FiscalSign:           fp,
	})
	if err == nil {
		rid := existingID
		count, err := q.CountReceiptLinkedTransactions(ctx, sqlcdb.CountReceiptLinkedTransactionsParams{
			UserID:          userID,
			ReceiptImportID: &rid,
		})
		if err != nil {
			return "", nil, err
		}
		if count > 0 {
			return "", nil, ErrDuplicate
		}
		row, err := q.GetReceiptImportByID(ctx, sqlcdb.GetReceiptImportByIDParams{ID: existingID, UserID: userID})
		if err != nil {
			return "", nil, err
		}
		return existingID, row.StoreID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", nil, err
	}

	sid, err := store.GetOrCreate(ctx, dbTx, userID, receipt.StoreName)
	if err != nil {
		return "", nil, err
	}

	payload := fns.ToPayload(receipt)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	receiptID = uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(receipt.DateTime)
	if err := q.InsertReceiptImport(ctx, sqlcdb.InsertReceiptImportParams{
		ID:                   receiptID,
		UserID:               userID,
		FiscalDriveNumber:    fn,
		FiscalDocumentNumber: fd,
		FiscalSign:           fp,
		TotalSum:             receipt.TotalSum,
		StoreID:              sid,
		TransactionDate:      txDate,
		PayloadJson:          string(payloadJSON),
		FnsCheckUrl:          receipt.FNSCheckURL,
		CreatedAt:            now,
	}); err != nil {
		return "", nil, err
	}
	return receiptID, sid, nil
}
