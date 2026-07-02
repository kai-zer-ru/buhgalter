package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const transferCategoryName = "Перевод"

type TransferInput struct {
	FromAccountID   string
	ToAccountID     string
	Amount          int64
	Commission      int64
	Description     *string
	TransactionDate time.Time
}

type Transfer struct {
	GroupID           string        `json:"group_id"`
	FromAccountID     string        `json:"from_account_id"`
	ToAccountID       string        `json:"to_account_id"`
	Amount            int64         `json:"amount"`
	AmountDisplay     string        `json:"amount_display"`
	Commission        int64         `json:"commission"`
	CommissionDisplay string        `json:"commission_display"`
	Description       *string       `json:"description"`
	TransactionDate   string        `json:"transaction_date"`
	Kind              string        `json:"kind"`
	Legs              []Transaction `json:"legs"`
}

func CreateTransfer(ctx context.Context, db *sql.DB, userID string, in TransferInput) (Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, ErrSameAccount
	}
	if in.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if in.Commission < 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if err := validateActiveAccount(ctx, db, userID, in.FromAccountID); err != nil {
		return Transfer{}, err
	}
	if err := validateActiveAccount(ctx, db, userID, in.ToAccountID); err != nil {
		return Transfer{}, err
	}
	if err := validateCreditCardTransferAmount(ctx, db, userID, in.ToAccountID, in.Amount); err != nil {
		return Transfer{}, err
	}

	kind, err := resolveKind(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return Transfer{}, err
	}

	catID, err := ensureTransferCategory(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	commissionCatID, err := categoryseed.CommissionCategoryID(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	fromType, toType, err := accountTypes(ctx, db, userID, in.FromAccountID, in.ToAccountID)
	if err != nil {
		return Transfer{}, err
	}
	commissionDesc := transferCommissionDescription(fromType, toType)

	groupID := uuid.NewString()
	outNow := time.Now().UTC()
	inNow := outNow.Add(time.Millisecond)
	commissionNow := inNow.Add(time.Millisecond)
	outCreated := outNow.Format(time.RFC3339Nano)
	inCreated := inNow.Format(time.RFC3339Nano)
	commissionCreated := commissionNow.Format(time.RFC3339Nano)
	updated := outCreated
	txDate := timeutil.FormatUTC(in.TransactionDate)
	outID := uuid.NewString()
	inID := uuid.NewString()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: outID, UserID: userID, AccountID: in.FromAccountID,
		Type: "transfer", Kind: kind, Amount: in.Amount, Description: in.Description,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: &in.ToAccountID,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: outCreated, UpdatedAt: updated,
	}); err != nil {
		return Transfer{}, fmt.Errorf("insert transfer out: %w", err)
	}
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: inID, UserID: userID, AccountID: in.ToAccountID,
		Type: "transfer", Kind: kind, Amount: in.Amount, Description: in.Description,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: &in.FromAccountID,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: inCreated, UpdatedAt: updated,
	}); err != nil {
		return Transfer{}, fmt.Errorf("insert transfer in: %w", err)
	}
	if err := insertTransferCommission(ctx, q, userID, groupID, in.FromAccountID, commissionCatID, in.Commission, commissionDesc, kind, txDate, commissionCreated, updated); err != nil {
		return Transfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	if err := refreshAccountBalances(ctx, db, userID, in.FromAccountID, in.ToAccountID); err != nil {
		return Transfer{}, err
	}
	return GetTransfer(ctx, db, userID, groupID)
}

// CreateTransferForAccountDelete moves the full balance when deleting cash/bank accounts.
// The source account may be active or archived; the target must be active.
func CreateTransferForAccountDelete(ctx context.Context, db *sql.DB, userID string, in TransferInput) (Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, ErrSameAccount
	}
	if in.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if in.Commission != 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if err := validateAccountForTransfer(ctx, db, userID, in.FromAccountID, false); err != nil {
		return Transfer{}, err
	}
	if err := validateAccountForTransfer(ctx, db, userID, in.ToAccountID, true); err != nil {
		return Transfer{}, err
	}
	if err := validateCreditCardTransferAmount(ctx, db, userID, in.ToAccountID, in.Amount); err != nil {
		return Transfer{}, err
	}

	kind, err := resolveKind(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return Transfer{}, err
	}

	catID, err := ensureTransferCategory(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	commissionCatID, err := categoryseed.CommissionCategoryID(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	fromType, toType, err := accountTypes(ctx, db, userID, in.FromAccountID, in.ToAccountID)
	if err != nil {
		return Transfer{}, err
	}
	commissionDesc := transferCommissionDescription(fromType, toType)

	groupID := uuid.NewString()
	outNow := time.Now().UTC()
	inNow := outNow.Add(time.Millisecond)
	commissionNow := inNow.Add(time.Millisecond)
	outCreated := outNow.Format(time.RFC3339Nano)
	inCreated := inNow.Format(time.RFC3339Nano)
	commissionCreated := commissionNow.Format(time.RFC3339Nano)
	updated := outCreated
	txDate := timeutil.FormatUTC(in.TransactionDate)
	outID := uuid.NewString()
	inID := uuid.NewString()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: outID, UserID: userID, AccountID: in.FromAccountID,
		Type: "transfer", Kind: kind, Amount: in.Amount, Description: in.Description,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: &in.ToAccountID,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: outCreated, UpdatedAt: updated,
	}); err != nil {
		return Transfer{}, fmt.Errorf("insert transfer out: %w", err)
	}
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: inID, UserID: userID, AccountID: in.ToAccountID,
		Type: "transfer", Kind: kind, Amount: in.Amount, Description: in.Description,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: &in.FromAccountID,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: inCreated, UpdatedAt: updated,
	}); err != nil {
		return Transfer{}, fmt.Errorf("insert transfer in: %w", err)
	}
	if err := insertTransferCommission(ctx, q, userID, groupID, in.FromAccountID, commissionCatID, 0, commissionDesc, kind, txDate, commissionCreated, updated); err != nil {
		return Transfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	if err := refreshAccountBalances(ctx, db, userID, in.FromAccountID, in.ToAccountID); err != nil {
		return Transfer{}, err
	}
	return Transfer{GroupID: groupID}, nil
}

func UpdateTransfer(ctx context.Context, db *sql.DB, userID, groupID string, in TransferInput) (Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, ErrSameAccount
	}
	if in.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if in.Commission < 0 {
		return Transfer{}, ErrInvalidAmount
	}

	rows, err := queries(db).ListTransactionsByTransferGroup(ctx, sqlcdb.ListTransactionsByTransferGroupParams{
		TransferGroupID: &groupID, UserID: userID,
	})
	if err != nil {
		return Transfer{}, err
	}
	commissionCatID, err := categoryseed.CommissionCategoryID(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	transferLegs := filterTransferLegs(rows)
	commissionLeg := findCommissionLeg(rows, commissionCatID)
	if len(transferLegs) != 2 {
		return Transfer{}, ErrTransferNotFound
	}

	if err := validateAccountForTransfer(ctx, db, userID, in.FromAccountID, true); err != nil {
		return Transfer{}, err
	}
	if err := validateAccountForTransfer(ctx, db, userID, in.ToAccountID, true); err != nil {
		return Transfer{}, err
	}
	if err := validateCreditCardTransferAmount(ctx, db, userID, in.ToAccountID, in.Amount); err != nil {
		return Transfer{}, err
	}

	kind, err := resolveKind(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return Transfer{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.TransactionDate)

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	outCreated := time.Now().UTC().Format(time.RFC3339Nano)
	inCreated := time.Now().UTC().Add(time.Millisecond).Format(time.RFC3339Nano)
	for i, leg := range transferLegs {
		var accountID string
		var transferAccountID string
		var legCreated string
		if i == 0 {
			accountID = in.FromAccountID
			transferAccountID = in.ToAccountID
			legCreated = outCreated
		} else {
			accountID = in.ToAccountID
			transferAccountID = in.FromAccountID
			legCreated = inCreated
		}
		if err := q.UpdateTransferLeg(ctx, sqlcdb.UpdateTransferLegParams{
			AccountID: accountID, Amount: in.Amount, Description: in.Description,
			TransactionDate: txDate, Kind: kind, UpdatedAt: now,
			ID: leg.ID, UserID: userID,
		}); err != nil {
			return Transfer{}, err
		}
		if err := updateTransferAccountID(ctx, q, leg.ID, userID, transferAccountID, now); err != nil {
			return Transfer{}, err
		}
		if err := updateTransferCreatedAt(ctx, q, leg.ID, userID, legCreated); err != nil {
			return Transfer{}, err
		}
	}
	commissionCreated := time.Now().UTC().Add(2 * time.Millisecond).Format(time.RFC3339Nano)
	fromType, toType, err := accountTypes(ctx, db, userID, in.FromAccountID, in.ToAccountID)
	if err != nil {
		return Transfer{}, err
	}
	commissionDesc := transferCommissionDescription(fromType, toType)
	if err := syncTransferCommission(ctx, q, userID, groupID, in.FromAccountID, commissionCatID, in.Commission, commissionDesc, kind, txDate, now, commissionCreated, commissionLeg); err != nil {
		return Transfer{}, err
	}
	if err := dbTx.Commit(); err != nil {
		return Transfer{}, err
	}
	accountIDs := uniqueAccountIDs(in.FromAccountID, in.ToAccountID)
	for _, leg := range transferLegs {
		accountIDs = append(accountIDs, leg.AccountID)
	}
	if err := refreshAccountBalances(ctx, db, userID, uniqueAccountIDs(accountIDs...)...); err != nil {
		return Transfer{}, err
	}
	return GetTransfer(ctx, db, userID, groupID)
}

func DeleteTransfer(ctx context.Context, db *sql.DB, userID, groupID string) error {
	gid := groupID
	rows, err := queries(db).ListTransactionsByTransferGroup(ctx, sqlcdb.ListTransactionsByTransferGroupParams{
		TransferGroupID: &gid, UserID: userID,
	})
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrTransferNotFound
	}
	accountIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		accountIDs = append(accountIDs, row.AccountID)
	}
	n, err := queries(db).DeleteTransactionsByGroup(ctx, sqlcdb.DeleteTransactionsByGroupParams{
		TransferGroupID: &gid, UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTransferNotFound
	}
	return refreshAccountBalances(ctx, db, userID, uniqueAccountIDs(accountIDs...)...)
}

func GetTransfer(ctx context.Context, db *sql.DB, userID, groupID string) (Transfer, error) {
	gid := groupID
	rows, err := queries(db).ListTransactionsByTransferGroup(ctx, sqlcdb.ListTransactionsByTransferGroupParams{
		TransferGroupID: &gid, UserID: userID,
	})
	if err != nil {
		return Transfer{}, err
	}
	if len(rows) == 0 {
		return Transfer{}, ErrTransferNotFound
	}

	commissionCatID, err := categoryseed.CommissionCategoryID(ctx, db, userID)
	if err != nil {
		return Transfer{}, err
	}
	transferLegs := filterTransferLegs(rows)
	commissionLeg := findCommissionLeg(rows, commissionCatID)
	if len(transferLegs) == 0 {
		return Transfer{}, ErrTransferNotFound
	}

	legs := make([]Transaction, 0, len(transferLegs))
	var fromID, toID string
	for i, row := range transferLegs {
		legs = append(legs, txFromGroupRow(row))
		if i == 0 {
			fromID = row.AccountID
			if row.TransferAccountID != nil {
				toID = *row.TransferAccountID
			}
		}
	}

	first := transferLegs[0]
	var commission int64
	if commissionLeg != nil {
		commission = commissionLeg.Amount
	}
	return Transfer{
		GroupID:           groupID,
		FromAccountID:     fromID,
		ToAccountID:       toID,
		Amount:            first.Amount,
		AmountDisplay:     money.FormatRubles(first.Amount),
		Commission:        commission,
		CommissionDisplay: money.FormatRubles(commission),
		Description:       first.Description,
		TransactionDate:   first.TransactionDate,
		Kind:              first.Kind,
		Legs:              legs,
	}, nil
}

func insertTransferCommission(
	ctx context.Context,
	q *sqlcdb.Queries,
	userID, groupID, fromAccountID, commissionCatID string,
	commission int64,
	description *string,
	kind, txDate, createdAt, updatedAt string,
) error {
	if commission <= 0 {
		return nil
	}
	id := uuid.NewString()
	return q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: id, UserID: userID, AccountID: fromAccountID,
		Type: "expense", Kind: kind, Amount: commission, Description: description,
		CategoryID: &commissionCatID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: nil,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func syncTransferCommission(
	ctx context.Context,
	q *sqlcdb.Queries,
	userID, groupID, fromAccountID, commissionCatID string,
	commission int64,
	description *string,
	kind, txDate, updatedAt, createdAt string,
	existing *sqlcdb.ListTransactionsByTransferGroupRow,
) error {
	if commission <= 0 {
		if existing == nil {
			return nil
		}
		n, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{ID: existing.ID, UserID: userID})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	}
	if existing != nil {
		return q.UpdateTransaction(ctx, sqlcdb.UpdateTransactionParams{
			AccountID: fromAccountID, Type: "expense", Kind: kind, Amount: commission, Description: description,
			CategoryID: &commissionCatID, SubcategoryID: nil, TransactionDate: txDate, UpdatedAt: updatedAt,
			ID: existing.ID, UserID: userID,
		})
	}
	return q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: uuid.NewString(), UserID: userID, AccountID: fromAccountID,
		Type: "expense", Kind: kind, Amount: commission, Description: description,
		CategoryID: &commissionCatID, SubcategoryID: nil,
		TransferGroupID: &groupID, TransferAccountID: nil,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func filterTransferLegs(rows []sqlcdb.ListTransactionsByTransferGroupRow) []sqlcdb.ListTransactionsByTransferGroupRow {
	out := make([]sqlcdb.ListTransactionsByTransferGroupRow, 0, 2)
	for _, row := range rows {
		if row.Type == "transfer" {
			out = append(out, row)
		}
	}
	return out
}

func findCommissionLeg(rows []sqlcdb.ListTransactionsByTransferGroupRow, commissionCatID string) *sqlcdb.ListTransactionsByTransferGroupRow {
	for i := range rows {
		row := &rows[i]
		if row.Type != "expense" || row.CategoryID == nil || *row.CategoryID != commissionCatID {
			continue
		}
		return row
	}
	return nil
}

func updateTransferCreatedAt(ctx context.Context, q *sqlcdb.Queries, id, userID, createdAt string) error {
	return q.UpdateTransferCreatedAt(ctx, sqlcdb.UpdateTransferCreatedAtParams{
		CreatedAt: createdAt,
		ID:        id,
		UserID:    userID,
	})
}

func updateTransferAccountID(ctx context.Context, q *sqlcdb.Queries, id, userID, transferAccountID, updatedAt string) error {
	return q.UpdateTransferAccountID(ctx, sqlcdb.UpdateTransferAccountIDParams{
		TransferAccountID: &transferAccountID,
		UpdatedAt:         updatedAt,
		ID:                id,
		UserID:            userID,
	})
}

func ensureTransferCategory(ctx context.Context, db *sql.DB, userID string) (string, error) {
	row, err := queries(db).GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: transferCategoryName, Type: "expense",
	})
	if err == nil {
		return row.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertCategory(ctx, sqlcdb.InsertCategoryParams{
		ID: id, UserID: userID, Name: transferCategoryName, Type: "expense",
		Icon: "default", SortOrder: 9999, IsPrimary: 0, IsSystem: 0, CreatedAt: now,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func accountTypes(ctx context.Context, db *sql.DB, userID, fromID, toID string) (string, string, error) {
	fromRow, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: fromID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrInvalidAccount
	}
	if err != nil {
		return "", "", err
	}
	toRow, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: toID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrInvalidAccount
	}
	if err != nil {
		return "", "", err
	}
	return fromRow.Type, toRow.Type, nil
}

func transferCommissionDescription(fromType, toType string) *string {
	if fromType == "credit_card" {
		s := "Комиссия за перевод"
		return &s
	}
	if toType == "credit_card" {
		s := "Комиссия за использование карты"
		return &s
	}
	return nil
}

func validateCreditCardTransferAmount(ctx context.Context, db *sql.DB, userID, toAccountID string, amount int64) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: toAccountID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if !account.IsCreditCard(row.Type) {
		return nil
	}
	if row.CreditLimit == nil {
		return ErrCreditCardPaymentExceedsLimit
	}
	maxAmount := *row.CreditLimit - row.CurrentBalance
	if amount > maxAmount {
		return ErrCreditCardPaymentExceedsLimit
	}
	return nil
}
