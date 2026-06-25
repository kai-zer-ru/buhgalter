package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Transaction struct {
	ID                string  `json:"id"`
	AccountID         string  `json:"account_id"`
	AccountName           string  `json:"account_name,omitempty"`
	TransferAccountName   string  `json:"transfer_account_name,omitempty"`
	Type              string  `json:"type"`
	Kind              string  `json:"kind"`
	Amount            int64   `json:"amount"`
	AmountDisplay     string  `json:"amount_display"`
	Description       *string `json:"description"`
	CategoryID        *string `json:"category_id"`
	CategoryName      *string `json:"category_name,omitempty"`
	CategoryIcon      *string `json:"category_icon,omitempty"`
	SubcategoryID     *string `json:"subcategory_id"`
	SubcategoryName   *string `json:"subcategory_name,omitempty"`
	TransferGroupID   *string `json:"transfer_group_id,omitempty"`
	TransferAccountID   *string `json:"transfer_account_id,omitempty"`
	TransferIsOut         bool `json:"transfer_is_out,omitempty"`
	CreditPaymentLinked   bool `json:"credit_payment_linked,omitempty"`
	TransactionDate       string  `json:"transaction_date"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type ListFilters struct {
	AccountID  string
	Type       string
	CategoryID string
	Kind       string
	From       string
	To         string
	Search     string
	Sort       string
	Page       int
	Limit      int
}

type ListMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type ListResult struct {
	Data []Transaction `json:"data"`
	Meta ListMeta      `json:"meta"`
}

var (
	ErrNotFound          = errors.New("transaction not found")
	ErrTransferNotFound  = errors.New("transfer not found")
	ErrInvalidType       = errors.New("invalid transaction type")
	ErrInvalidAccount    = errors.New("invalid account")
	ErrAccountArchived   = errors.New("account is archived")
	ErrInvalidCategory   = errors.New("invalid category")
	ErrCategoryTypeMatch = errors.New("category type mismatch")
	ErrInvalidSubcategory = errors.New("invalid subcategory")
	ErrInvalidDate       = errors.New("invalid transaction date")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrSameAccount       = errors.New("same account for transfer")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func txFromGetRow(row sqlcdb.GetTransactionByIDRow) Transaction {
	return txFromFields(
		row.ID, row.AccountID, row.Type, row.Kind, row.Amount, row.Description,
		row.CategoryID, row.SubcategoryID, row.TransferGroupID, row.TransferAccountID,
		row.TransactionDate, row.CreatedAt, row.UpdatedAt,
		row.CategoryName, row.CategoryIcon, row.SubcategoryName, row.AccountName, row.TransferAccountName,
		row.TransferIsOut, row.CreditPaymentLinked,
	)
}

func txFromListDesc(row sqlcdb.ListTransactionsFilteredDateDescRow) Transaction {
	return txFromFields(
		row.ID, row.AccountID, row.Type, row.Kind, row.Amount, row.Description,
		row.CategoryID, row.SubcategoryID, row.TransferGroupID, row.TransferAccountID,
		row.TransactionDate, row.CreatedAt, row.UpdatedAt,
		row.CategoryName, row.CategoryIcon, row.SubcategoryName, row.AccountName, row.TransferAccountName,
		row.TransferIsOut, row.CreditPaymentLinked,
	)
}

func txFromListAsc(row sqlcdb.ListTransactionsFilteredDateAscRow) Transaction {
	return txFromFields(
		row.ID, row.AccountID, row.Type, row.Kind, row.Amount, row.Description,
		row.CategoryID, row.SubcategoryID, row.TransferGroupID, row.TransferAccountID,
		row.TransactionDate, row.CreatedAt, row.UpdatedAt,
		row.CategoryName, row.CategoryIcon, row.SubcategoryName, row.AccountName, row.TransferAccountName,
		row.TransferIsOut, row.CreditPaymentLinked,
	)
}

func txFromRecent(row sqlcdb.ListRecentTransactionsRow) Transaction {
	return txFromFields(
		row.ID, row.AccountID, row.Type, row.Kind, row.Amount, row.Description,
		row.CategoryID, row.SubcategoryID, row.TransferGroupID, row.TransferAccountID,
		row.TransactionDate, row.CreatedAt, row.UpdatedAt,
		row.CategoryName, row.CategoryIcon, row.SubcategoryName, row.AccountName, row.TransferAccountName,
		row.TransferIsOut, row.CreditPaymentLinked,
	)
}

func txFromGroupRow(row sqlcdb.ListTransactionsByTransferGroupRow) Transaction {
	return txFromFields(
		row.ID, row.AccountID, row.Type, row.Kind, row.Amount, row.Description,
		row.CategoryID, row.SubcategoryID, row.TransferGroupID, row.TransferAccountID,
		row.TransactionDate, row.CreatedAt, row.UpdatedAt,
		row.CategoryName, row.CategoryIcon, row.SubcategoryName, row.AccountName, row.TransferAccountName,
		row.TransferIsOut, 0,
	)
}

func txFromFields(
	id, accountID, txType, kind string, amount int64, description *string,
	categoryID, subcategoryID, transferGroupID, transferAccountID *string,
	transactionDate, createdAt, updatedAt string,
	categoryName, categoryIcon, subcategoryName, accountName, transferAccountName *string,
	transferIsOut, creditPaymentLinked int64,
) Transaction {
	t := Transaction{
		ID:                id,
		AccountID:         accountID,
		Type:              txType,
		Kind:              kind,
		Amount:            amount,
		AmountDisplay:     money.FormatRubles(amount),
		Description:       description,
		CategoryID:        categoryID,
		SubcategoryID:     subcategoryID,
		TransferGroupID:   transferGroupID,
		TransferAccountID: transferAccountID,
		TransactionDate:   transactionDate,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		CategoryName:      categoryName,
		CategoryIcon:      categoryIcon,
		SubcategoryName:   subcategoryName,
	}
	if accountName != nil {
		t.AccountName = *accountName
	}
	if transferAccountName != nil {
		t.TransferAccountName = *transferAccountName
	}
	if txType == "transfer" {
		t.TransferIsOut = transferIsOut == 1
	}
	if creditPaymentLinked == 1 {
		t.CreditPaymentLinked = true
	}
	return t
}

func userTimezone(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func resolveKind(ctx context.Context, db *sql.DB, userID string, txDate time.Time) (string, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return "", err
	}
	future, err := timeutil.IsFutureInTZ(txDate, timeutil.NowUTC(), tz)
	if err != nil {
		return "", err
	}
	if future {
		return "future", nil
	}
	return "manual", nil
}

type CreateInput struct {
	AccountID        string
	Type             string
	Amount           int64
	Description      *string
	CategoryID       *string
	SubcategoryID    *string
	SubcategoryName  *string
	TransactionDate  time.Time
}

func Create(ctx context.Context, db *sql.DB, userID string, in CreateInput) (Transaction, error) {
	if in.Type != "income" && in.Type != "expense" {
		return Transaction{}, ErrInvalidType
	}
	if in.Amount <= 0 {
		return Transaction{}, ErrInvalidAmount
	}
	if err := validateActiveAccount(ctx, db, userID, in.AccountID); err != nil {
		return Transaction{}, err
	}
	subID, err := resolveSubcategory(ctx, db, userID, in.CategoryID, in.SubcategoryID, in.SubcategoryName)
	if err != nil {
		return Transaction{}, err
	}
	if err := validateCategoryForType(ctx, db, userID, in.CategoryID, in.Type); err != nil {
		return Transaction{}, err
	}

	kind, err := resolveKind(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return Transaction{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.TransactionDate)
	if err := queries(db).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID:                id,
		UserID:            userID,
		AccountID:         in.AccountID,
		Type:              in.Type,
		Kind:              kind,
		Amount:            in.Amount,
		Description:       in.Description,
		CategoryID:        in.CategoryID,
		SubcategoryID:     subID,
		TransferGroupID:   nil,
		TransferAccountID: nil,
		TransactionDate:   txDate,
		AffectsBalance:    1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	return GetByID(ctx, db, userID, id)
}

type UpdateInput struct {
	AccountID       string
	Type            string
	Amount          int64
	Description     *string
	CategoryID      *string
	SubcategoryID   *string
	SubcategoryName *string
	TransactionDate time.Time
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in UpdateInput) (Transaction, error) {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Transaction{}, err
	}
	if err := credit.GuardTransactionUpdate(ctx, db, userID, id); err != nil {
		return Transaction{}, err
	}
	if existing.TransferGroupID != nil && *existing.TransferGroupID != "" {
		return Transaction{}, fmt.Errorf("use transfer endpoint to update transfers")
	}
	if in.Type != "income" && in.Type != "expense" {
		return Transaction{}, ErrInvalidType
	}
	if in.Amount <= 0 {
		return Transaction{}, ErrInvalidAmount
	}
	if err := validateActiveAccount(ctx, db, userID, in.AccountID); err != nil {
		return Transaction{}, err
	}
	subID, err := resolveSubcategory(ctx, db, userID, in.CategoryID, in.SubcategoryID, in.SubcategoryName)
	if err != nil {
		return Transaction{}, err
	}
	if err := validateCategoryForType(ctx, db, userID, in.CategoryID, in.Type); err != nil {
		return Transaction{}, err
	}
	kind, err := resolveKind(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return Transaction{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).UpdateTransaction(ctx, sqlcdb.UpdateTransactionParams{
		AccountID:       in.AccountID,
		Type:            in.Type,
		Kind:            kind,
		Amount:          in.Amount,
		Description:     in.Description,
		CategoryID:      in.CategoryID,
		SubcategoryID:   subID,
		TransactionDate: timeutil.FormatUTC(in.TransactionDate),
		UpdatedAt:       now,
		ID:              id,
		UserID:          userID,
	}); err != nil {
		return Transaction{}, err
	}
	return GetByID(ctx, db, userID, id)
}

func GetByID(ctx context.Context, db *sql.DB, userID, id string) (Transaction, error) {
	row, err := queries(db).GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	if err != nil {
		return Transaction{}, err
	}
	return txFromGetRow(row), nil
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return err
	}
	if existing.TransferGroupID != nil && *existing.TransferGroupID != "" {
		return DeleteTransfer(ctx, db, userID, *existing.TransferGroupID)
	}
	if err := debt.GuardTransactionDelete(ctx, db, userID, id); err != nil {
		return err
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	if err := q.ClearDebtTransactionLink(ctx, sqlcdb.ClearDebtTransactionLinkParams{
		TransactionID: &id, UserID: userID,
	}); err != nil {
		return err
	}
	if err := q.DeleteDebtTransactionLink(ctx, id); err != nil {
		return err
	}
	if err := credit.OnTransactionDelete(ctx, q, userID, id); err != nil {
		return err
	}
	n, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return dbTx.Commit()
}

func Activate(ctx context.Context, db *sql.DB, userID, id string) (Transaction, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).ActivateTransaction(ctx, sqlcdb.ActivateTransactionParams{
		UpdatedAt: now,
		ID:        id,
		UserID:    userID,
	})
	if err != nil {
		return Transaction{}, err
	}
	if n == 0 {
		return Transaction{}, ErrNotFound
	}
	return GetByID(ctx, db, userID, id)
}

// ActivateDueFutureTransactions promotes past-due planned operations to manual (UTC cutoff = now).
func ActivateDueFutureTransactions(ctx context.Context, db *sql.DB, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	cutoff := timeutil.FormatUTC(timeutil.NowUTC())
	q := queries(db)
	if _, err := q.ActivateAppliedCreditFutureTransactions(ctx, sqlcdb.ActivateAppliedCreditFutureTransactionsParams{
		UpdatedAt: now,
		UserID:    userID,
	}); err != nil {
		return err
	}
	_, err := q.ActivateFutureTransactionsBefore(ctx, sqlcdb.ActivateFutureTransactionsBeforeParams{
		UpdatedAt:       now,
		UserID:          userID,
		TransactionDate: cutoff,
	})
	return err
}

func List(ctx context.Context, db *sql.DB, userID string, f ListFilters) (ListResult, error) {
	if err := ActivateDueFutureTransactions(ctx, db, userID); err != nil {
		return ListResult{}, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 50
	}
	if err := validateDateFilter(f.From); err != nil {
		return ListResult{}, err
	}
	if err := validateDateFilter(f.To); err != nil {
		return ListResult{}, err
	}

	fp := buildFilterParams(userID, f)
	q := queries(db)
	total, err := q.CountTransactionsFiltered(ctx, fp.count())
	if err != nil {
		return ListResult{}, err
	}

	offset := int64((f.Page - 1) * f.Limit)
	limit := int64(f.Limit)
	var items []Transaction
	if f.Sort == "date_asc" {
		rows, err := q.ListTransactionsFilteredDateAsc(ctx, fp.listAsc(limit, offset))
		if err != nil {
			return ListResult{}, err
		}
		items = make([]Transaction, 0, len(rows))
		for _, row := range rows {
			items = append(items, txFromListAsc(row))
		}
	} else {
		rows, err := q.ListTransactionsFilteredDateDesc(ctx, fp.listDesc(limit, offset))
		if err != nil {
			return ListResult{}, err
		}
		items = make([]Transaction, 0, len(rows))
		for _, row := range rows {
			items = append(items, txFromListDesc(row))
		}
	}
	if items == nil {
		items = []Transaction{}
	}
	return ListResult{
		Data: items,
		Meta: ListMeta{Page: f.Page, Limit: f.Limit, Total: total},
	}, nil
}

func ListRecent(ctx context.Context, db *sql.DB, userID string, limit int) ([]Transaction, error) {
	if limit < 1 {
		limit = 10
	}
	rows, err := queries(db).ListRecentTransactions(ctx, sqlcdb.ListRecentTransactionsParams{
		UserID: userID,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		out = append(out, txFromRecent(row))
	}
	return out, nil
}

type filterParams struct {
	userID     string
	accountID  string
	txType     string
	categoryID string
	kind       string
	from       string
	to         string
	search     string
}

func buildFilterParams(userID string, f ListFilters) filterParams {
	return filterParams{
		userID:     userID,
		accountID:  f.AccountID,
		txType:     f.Type,
		categoryID: f.CategoryID,
		kind:       f.Kind,
		from:       f.From,
		to:         f.To,
		search:     f.Search,
	}
}

func (fp filterParams) count() sqlcdb.CountTransactionsFilteredParams {
	searchPtr := strPtr(fp.search)
	catPtr := strPtr(fp.categoryID)
	return sqlcdb.CountTransactionsFilteredParams{
		UserID:            fp.userID,
		Column2:           fp.accountID,
		AccountID:         fp.accountID,
		Column4:           fp.txType,
		Type:              fp.txType,
		Column6:           fp.categoryID,
		CategoryID:        catPtr,
		Column8:           fp.kind,
		Kind:              fp.kind,
		Column10:          fp.from,
		TransactionDate:   fp.from,
		Column12:          fp.to,
		TransactionDate_2: fp.to,
		Column14:          fp.search,
		Column15:          searchPtr,
	}
}

func (fp filterParams) listDesc(limit, offset int64) sqlcdb.ListTransactionsFilteredDateDescParams {
	p := fp.count()
	return sqlcdb.ListTransactionsFilteredDateDescParams{
		UserID:            p.UserID,
		Column2:           p.Column2,
		AccountID:         p.AccountID,
		Column4:           p.Column4,
		Type:              p.Type,
		Column6:           p.Column6,
		CategoryID:        p.CategoryID,
		Column8:           p.Column8,
		Kind:              p.Kind,
		Column10:          p.Column10,
		TransactionDate:   p.TransactionDate,
		Column12:          p.Column12,
		TransactionDate_2: p.TransactionDate_2,
		Column14:          p.Column14,
		Column15:          p.Column15,
		Limit:             limit,
		Offset:            offset,
	}
}

func (fp filterParams) listAsc(limit, offset int64) sqlcdb.ListTransactionsFilteredDateAscParams {
	p := fp.count()
	return sqlcdb.ListTransactionsFilteredDateAscParams{
		UserID:            p.UserID,
		Column2:           p.Column2,
		AccountID:         p.AccountID,
		Column4:           p.Column4,
		Type:              p.Type,
		Column6:           p.Column6,
		CategoryID:        p.CategoryID,
		Column8:           p.Column8,
		Kind:              p.Kind,
		Column10:          p.Column10,
		TransactionDate:   p.TransactionDate,
		Column12:          p.Column12,
		TransactionDate_2: p.TransactionDate_2,
		Column14:          p.Column14,
		Column15:          p.Column15,
		Limit:             limit,
		Offset:            offset,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func validateDateFilter(s string) error {
	if s == "" {
		return nil
	}
	_, err := timeutil.ParseUTC(s)
	if err != nil {
		return ErrInvalidDate
	}
	return nil
}

func validateActiveAccount(ctx context.Context, db *sql.DB, userID, accountID string) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if row.Status != "active" {
		return ErrAccountArchived
	}
	return nil
}

func validateCategoryForType(ctx context.Context, db *sql.DB, userID string, categoryID *string, txType string) error {
	if categoryID == nil || *categoryID == "" {
		return nil
	}
	cat, err := category.GetByID(ctx, db, userID, *categoryID)
	if errors.Is(err, category.ErrNotFound) {
		return ErrInvalidCategory
	}
	if err != nil {
		return err
	}
	if cat.Type != txType {
		return ErrCategoryTypeMatch
	}
	return nil
}

func resolveSubcategory(ctx context.Context, db *sql.DB, userID string, categoryID, subcategoryID, subcategoryName *string) (*string, error) {
	if subcategoryID != nil && *subcategoryID != "" {
		sub, err := category.GetSubcategory(ctx, db, userID, *subcategoryID)
		if errors.Is(err, category.ErrSubNotFound) {
			return nil, ErrInvalidSubcategory
		}
		if err != nil {
			return nil, err
		}
		if categoryID != nil && *categoryID != "" && sub.CategoryID != *categoryID {
			return nil, ErrInvalidSubcategory
		}
		return subcategoryID, nil
	}
	if subcategoryName != nil && strings.TrimSpace(*subcategoryName) != "" {
		if categoryID == nil || *categoryID == "" {
			return nil, ErrInvalidCategory
		}
		sub, err := category.CreateSubcategory(ctx, db, userID, *categoryID, *subcategoryName, "")
		if err != nil {
			return nil, err
		}
		return &sub.ID, nil
	}
	return nil, nil
}

// ResolveKindForDate is exported for tests.
func ResolveKindForDate(ctx context.Context, db *sql.DB, userID string, txDate time.Time) (string, error) {
	return resolveKind(ctx, db, userID, txDate)
}
