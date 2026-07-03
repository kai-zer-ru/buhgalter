package categoryseed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const TransferCategoryName = "Перевод"
const TransfersCategoryName = "Переводы"

var transfersDefaultCategory = defaultCategory{
	Type: "expense", Name: TransfersCategoryName, Icon: "default", Sort: 6,
}

// EnsureTransfersCategory ensures the default user «Переводы» category exists.
func EnsureTransfersCategory(ctx context.Context, db sqlcdb.DBTX, userID string) error {
	q := sqlcdb.New(db)
	_, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: TransfersCategoryName, Type: "expense",
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return insertCategory(ctx, q, userID, transfersDefaultCategory, now)
}

// EnsureTransferCategory migrates legacy «Перевод» data before it becomes system.
func EnsureTransferCategory(ctx context.Context, db sqlcdb.DBTX, userID string) error {
	if err := EnsureTransfersCategory(ctx, db, userID); err != nil {
		return err
	}
	q := sqlcdb.New(db)
	perevodyID, err := transfersCategoryID(ctx, q, userID)
	if err != nil {
		return err
	}
	transferRow, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: TransferCategoryName, Type: "expense",
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if transferRow.ID == perevodyID {
		return nil
	}
	if err := migrateTransferSubcategories(ctx, q, userID, transferRow.ID, perevodyID); err != nil {
		return fmt.Errorf("migrate transfer subcategories: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	newCatID := perevodyID
	oldCatID := transferRow.ID
	if err := q.ReassignTransactionsCategoryByFilter(ctx, sqlcdb.ReassignTransactionsCategoryByFilterParams{
		CategoryID:   &newCatID,
		UpdatedAt:    now,
		UserID:       userID,
		CategoryID_2: &oldCatID,
		Type:         "transfer",
	}); err != nil {
		return fmt.Errorf("reassign transactions: %w", err)
	}
	if err := q.ReassignRecurringOperationsCategory(ctx, sqlcdb.ReassignRecurringOperationsCategoryParams{
		CategoryID:   perevodyID,
		UpdatedAt:    now,
		UserID:       userID,
		CategoryID_2: transferRow.ID,
	}); err != nil {
		return fmt.Errorf("reassign recurring: %w", err)
	}
	if err := q.ReassignBudgetsCategory(ctx, sqlcdb.ReassignBudgetsCategoryParams{
		CategoryID:   &newCatID,
		UpdatedAt:    now,
		UserID:       userID,
		CategoryID_2: &oldCatID,
	}); err != nil {
		return fmt.Errorf("reassign budgets: %w", err)
	}
	if err := q.ClearTransactionSubcategoriesByCategory(ctx, sqlcdb.ClearTransactionSubcategoriesByCategoryParams{
		UpdatedAt:  now,
		UserID:     userID,
		CategoryID: &oldCatID,
	}); err != nil {
		return fmt.Errorf("clear transfer subcategories: %w", err)
	}
	if transferRow.IsPrimary != 0 {
		if err := q.ClearCategoryPrimary(ctx, sqlcdb.ClearCategoryPrimaryParams{
			ID: transferRow.ID, UserID: userID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func migrateTransferSubcategories(ctx context.Context, q *sqlcdb.Queries, userID, fromCategoryID, toCategoryID string) error {
	subs, err := q.ListSubcategoriesByCategory(ctx, fromCategoryID)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}
	existing, err := q.ListSubcategoriesByCategory(ctx, toCategoryID)
	if err != nil {
		return err
	}
	byName := make(map[string]string, len(existing))
	for _, s := range existing {
		byName[s.Name] = s.ID
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, sub := range subs {
		if targetID, ok := byName[sub.Name]; ok {
			if err := mergeSubcategory(ctx, q, userID, sub.ID, targetID, now); err != nil {
				return err
			}
			continue
		}
		if err := q.MoveSubcategoryToCategory(ctx, sqlcdb.MoveSubcategoryToCategoryParams{
			CategoryID: toCategoryID,
			ID:         sub.ID,
		}); err != nil {
			return err
		}
		byName[sub.Name] = sub.ID
	}
	return nil
}

func mergeSubcategory(ctx context.Context, q *sqlcdb.Queries, userID, fromSubID, toSubID, now string) error {
	if err := q.ReassignTransactionSubcategory(ctx, sqlcdb.ReassignTransactionSubcategoryParams{
		SubcategoryID:   &toSubID,
		UpdatedAt:       now,
		UserID:          userID,
		SubcategoryID_2: &fromSubID,
	}); err != nil {
		return err
	}
	if err := q.ReassignRecurringSubcategory(ctx, sqlcdb.ReassignRecurringSubcategoryParams{
		SubcategoryID:   &toSubID,
		UpdatedAt:       now,
		UserID:          userID,
		SubcategoryID_2: &fromSubID,
	}); err != nil {
		return err
	}
	if err := q.ReassignBudgetSubcategory(ctx, sqlcdb.ReassignBudgetSubcategoryParams{
		SubcategoryID:   &toSubID,
		UpdatedAt:       now,
		UserID:          userID,
		SubcategoryID_2: &fromSubID,
	}); err != nil {
		return err
	}
	if _, err := q.DeleteSubcategory(ctx, fromSubID); err != nil {
		return err
	}
	return nil
}

// TransferCategoryID returns the system expense «Перевод» category id.
func TransferCategoryID(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	if err := EnsureSystemCategories(ctx, db, userID); err != nil {
		return "", err
	}
	row, err := sqlcdb.New(db).GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: TransferCategoryName, Type: "expense",
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// TransfersCategoryID returns the default user expense «Переводы» category id.
func TransfersCategoryID(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	if err := EnsureTransfersCategory(ctx, db, userID); err != nil {
		return "", err
	}
	return transfersCategoryID(ctx, sqlcdb.New(db), userID)
}

func transfersCategoryID(ctx context.Context, q *sqlcdb.Queries, userID string) (string, error) {
	row, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: TransfersCategoryName, Type: "expense",
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}
