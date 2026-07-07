package receipt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/store"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

var (
	ErrDuplicate       = errors.New("receipt already imported")
	ErrNotFound        = errors.New("receipt not found")
	ErrInvalidItem     = errors.New("invalid receipt item")
	ErrNoItemsSelected = errors.New("no items selected")
)

type Fetcher interface {
	FetchReceipt(ctx context.Context, qr string) (fns.ParsedReceipt, error)
}

type ImportItemInput struct {
	Index         int
	CategoryID    *string
	SubcategoryID *string
	Include       bool
}

type ImportInput struct {
	AccountID            string
	FiscalDriveNumber    string
	FiscalDocumentNumber string
	FiscalSign           string
	Receipt              fns.ParsedReceipt
	Items                []ImportItemInput
}

type ImportResult struct {
	ReceiptImportID string              `json:"receipt_import_id"`
	Transactions    []transaction.Transaction `json:"transactions"`
}

type Detail struct {
	ID                   string     `json:"id"`
	StoreName            string     `json:"store_name,omitempty"`
	FiscalDocumentNumber string     `json:"fiscal_document_number"`
	FiscalDriveNumber    string     `json:"fiscal_drive_number"`
	FiscalSign           string     `json:"fiscal_sign"`
	TransactionDate      string     `json:"transaction_date"`
	TotalSum             int64      `json:"total_sum"`
	TotalSumDisplay      string     `json:"total_sum_display"`
	Items                []fns.Item `json:"items"`
	FNSCheckURL          string     `json:"fns_check_url"`
	HighlightItemIndex   *int       `json:"highlight_item_index,omitempty"`
	UserINN              string     `json:"user_inn,omitempty"`
	RetailPlace          string     `json:"retail_place,omitempty"`
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func Parse(ctx context.Context, fetcher Fetcher, qr string) (fns.ParsedReceipt, error) {
	return fetcher.FetchReceipt(ctx, qr)
}

func GetByID(ctx context.Context, db *sql.DB, userID, id, transactionID string) (Detail, error) {
	row, err := queries(db).GetReceiptImportByID(ctx, sqlcdb.GetReceiptImportByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	payload, err := fns.PayloadFromJSON(row.PayloadJson)
	if err != nil {
		return Detail{}, err
	}
	detail := Detail{
		ID:                   row.ID,
		FiscalDocumentNumber: row.FiscalDocumentNumber,
		FiscalDriveNumber:    row.FiscalDriveNumber,
		FiscalSign:           row.FiscalSign,
		TransactionDate:      row.TransactionDate,
		TotalSum:             row.TotalSum,
		TotalSumDisplay:      money.FormatRubles(row.TotalSum),
		Items:                payload.Items,
		FNSCheckURL:          row.FnsCheckUrl,
		UserINN:              payload.UserINN,
		RetailPlace:          payload.RetailPlace,
	}
	if row.StoreName != nil && *row.StoreName != "" {
		detail.StoreName = *row.StoreName
	} else if payload.StoreName != "" {
		detail.StoreName = payload.StoreName
	}
	if transactionID != "" {
		receiptID := id
		idx, err := queries(db).GetTransactionReceiptItemIndex(ctx, sqlcdb.GetTransactionReceiptItemIndexParams{
			ID:              transactionID,
			UserID:          userID,
			ReceiptImportID: &receiptID,
		})
		if err == nil && idx != nil {
			v := int(*idx)
			detail.HighlightItemIndex = &v
		}
	}
	return detail, nil
}

func Import(ctx context.Context, db *sql.DB, userID string, in ImportInput) (ImportResult, error) {
	if len(in.Items) == 0 {
		return ImportResult{}, ErrNoItemsSelected
	}
	if err := transaction.ValidateActiveAccount(ctx, db, userID, in.AccountID); err != nil {
		return ImportResult{}, err
	}

	for _, itemIn := range in.Items {
		if !itemIn.Include {
			continue
		}
		if itemIn.CategoryID == nil || strings.TrimSpace(*itemIn.CategoryID) == "" {
			return ImportResult{}, fmt.Errorf("%w: category required", ErrInvalidItem)
		}
		if err := validateExpenseCategory(ctx, db, userID, itemIn.CategoryID); err != nil {
			return ImportResult{}, err
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	if _, err := q.GetReceiptImportByFiscalKey(ctx, sqlcdb.GetReceiptImportByFiscalKeyParams{
		UserID:               userID,
		FiscalDriveNumber:    in.FiscalDriveNumber,
		FiscalDocumentNumber: in.FiscalDocumentNumber,
		FiscalSign:           in.FiscalSign,
	}); err == nil {
		return ImportResult{}, ErrDuplicate
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ImportResult{}, err
	}

	storeID, err := store.GetOrCreate(ctx, tx, userID, in.Receipt.StoreName)
	if err != nil {
		return ImportResult{}, err
	}

	payload := fns.ToPayload(in.Receipt)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return ImportResult{}, err
	}

	receiptID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.Receipt.DateTime)
	if err := q.InsertReceiptImport(ctx, sqlcdb.InsertReceiptImportParams{
		ID:                   receiptID,
		UserID:               userID,
		FiscalDriveNumber:    in.FiscalDriveNumber,
		FiscalDocumentNumber: in.FiscalDocumentNumber,
		FiscalSign:           in.FiscalSign,
		TotalSum:             in.Receipt.TotalSum,
		StoreID:              storeID,
		TransactionDate:      txDate,
		PayloadJson:          string(payloadJSON),
		FnsCheckUrl:          in.Receipt.FNSCheckURL,
		CreatedAt:            now,
	}); err != nil {
		return ImportResult{}, err
	}

	totalDisplay := money.FormatRubles(in.Receipt.TotalSum)
	createdIDs := make([]string, 0)
	for _, itemIn := range in.Items {
		if !itemIn.Include {
			continue
		}
		if itemIn.Index < 0 || itemIn.Index >= len(in.Receipt.Items) {
			return ImportResult{}, ErrInvalidItem
		}

		item := in.Receipt.Items[itemIn.Index]
		amount := item.Sum
		if amount <= 0 {
			amount = item.Price
		}
		if amount <= 0 {
			return ImportResult{}, ErrInvalidItem
		}

		desc := fmt.Sprintf("%s. Чек №%s, сумма %s", item.Name, in.FiscalDocumentNumber, totalDisplay)
		idx := int64(itemIn.Index)
		txID, err := transaction.CreateWithReceipt(ctx, tx, userID, transaction.ReceiptCreateInput{
			AccountID:        in.AccountID,
			Amount:           amount,
			Description:      &desc,
			CategoryID:       itemIn.CategoryID,
			SubcategoryID:    itemIn.SubcategoryID,
			TransactionDate:  in.Receipt.DateTime,
			StoreID:          storeID,
			ReceiptImportID:  &receiptID,
			ReceiptItemIndex: &idx,
		})
		if err != nil {
			return ImportResult{}, err
		}
		createdIDs = append(createdIDs, txID)
	}
	if len(createdIDs) == 0 {
		return ImportResult{}, ErrNoItemsSelected
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	if err := transaction.RefreshAccountBalances(ctx, db, userID, in.AccountID); err != nil {
		return ImportResult{}, err
	}
	created := make([]transaction.Transaction, 0, len(createdIDs))
	for _, txID := range createdIDs {
		txRow, err := transaction.GetByID(ctx, db, userID, txID)
		if err != nil {
			return ImportResult{}, err
		}
		created = append(created, txRow)
	}
	return ImportResult{ReceiptImportID: receiptID, Transactions: created}, nil
}

func validateExpenseCategory(ctx context.Context, db *sql.DB, userID string, categoryID *string) error {
	if categoryID == nil || strings.TrimSpace(*categoryID) == "" {
		return ErrInvalidItem
	}
	cat, err := category.GetByID(ctx, db, userID, *categoryID)
	if errors.Is(err, category.ErrNotFound) {
		return ErrInvalidItem
	}
	if err != nil {
		return err
	}
	if cat.Type != "expense" {
		return ErrInvalidItem
	}
	return nil
}
