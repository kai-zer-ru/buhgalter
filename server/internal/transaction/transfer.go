package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const transferCategoryName = "Перевод"

type TransferInput struct {
	FromAccountID   string
	ToAccountID     string
	Amount          int64
	Description     *string
	TransactionDate time.Time
}

type Transfer struct {
	GroupID      string        `json:"group_id"`
	FromAccountID string       `json:"from_account_id"`
	ToAccountID   string       `json:"to_account_id"`
	Amount        int64         `json:"amount"`
	AmountDisplay string        `json:"amount_display"`
	Description   *string       `json:"description"`
	TransactionDate string      `json:"transaction_date"`
	Kind          string        `json:"kind"`
	Legs          []Transaction `json:"legs"`
}

func CreateTransfer(ctx context.Context, db *sql.DB, userID string, in TransferInput) (Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, ErrSameAccount
	}
	if in.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if err := validateActiveAccount(ctx, db, userID, in.FromAccountID); err != nil {
		return Transfer{}, err
	}
	if err := validateActiveAccount(ctx, db, userID, in.ToAccountID); err != nil {
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

	groupID := uuid.NewString()
	outNow := time.Now().UTC()
	inNow := outNow.Add(time.Millisecond)
	outCreated := outNow.Format(time.RFC3339Nano)
	inCreated := inNow.Format(time.RFC3339Nano)
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
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	return GetTransfer(ctx, db, userID, groupID)
}

func UpdateTransfer(ctx context.Context, db *sql.DB, userID, groupID string, in TransferInput) (Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, ErrSameAccount
	}
	if in.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	legs, err := queries(db).ListTransactionsByTransferGroup(ctx, sqlcdb.ListTransactionsByTransferGroupParams{
		TransferGroupID: &groupID, UserID: userID,
	})
	if err != nil {
		return Transfer{}, err
	}
	if len(legs) != 2 {
		return Transfer{}, ErrTransferNotFound
	}

	if err := validateActiveAccount(ctx, db, userID, in.FromAccountID); err != nil {
		return Transfer{}, err
	}
	if err := validateActiveAccount(ctx, db, userID, in.ToAccountID); err != nil {
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
	for i, leg := range legs {
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
	if err := dbTx.Commit(); err != nil {
		return Transfer{}, err
	}
	return GetTransfer(ctx, db, userID, groupID)
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

func DeleteTransfer(ctx context.Context, db *sql.DB, userID, groupID string) error {
	gid := groupID
	n, err := queries(db).DeleteTransactionsByGroup(ctx, sqlcdb.DeleteTransactionsByGroupParams{
		TransferGroupID: &gid, UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTransferNotFound
	}
	return nil
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

	legs := make([]Transaction, 0, len(rows))
	var fromID, toID string
	for i, row := range rows {
		legs = append(legs, txFromGroupRow(row))
		if i == 0 {
			fromID = row.AccountID
			if row.TransferAccountID != nil {
				toID = *row.TransferAccountID
			}
		}
	}

	first := rows[0]
	return Transfer{
		GroupID:         groupID,
		FromAccountID:   fromID,
		ToAccountID:     toID,
		Amount:          first.Amount,
		AmountDisplay:   legs[0].AmountDisplay,
		Description:     first.Description,
		TransactionDate: first.TransactionDate,
		Kind:            first.Kind,
		Legs:            legs,
	}, nil
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
