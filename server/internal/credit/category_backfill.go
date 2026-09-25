package credit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// EnsureCreditPaymentCategories moves linked credit payment transactions into
// the system «Кредиты» category (expense/income by type) and clears subcategory.
// Idempotent: already-correct rows are skipped.
func EnsureCreditPaymentCategories(ctx context.Context, db *sql.DB) error {
	q := queries(db)
	userIDs, err := q.ListUsersWithLinkedCreditPayments(ctx)
	if err != nil {
		return fmt.Errorf("list users with linked credit payments: %w", err)
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)
	for _, userID := range userIDs {
		expenseCat, err := categoryseed.CreditCategoryID(ctx, db, userID)
		if err != nil {
			return fmt.Errorf("credit expense category for %s: %w", userID, err)
		}
		if _, err := q.RelinkCreditPaymentExpenseCategories(ctx, sqlcdb.RelinkCreditPaymentExpenseCategoriesParams{
			CategoryID:   &expenseCat,
			UpdatedAt:    nowStr,
			UserID:       userID,
			UserID_2:     userID,
			CategoryID_2: &expenseCat,
		}); err != nil {
			return fmt.Errorf("relink expense categories for %s: %w", userID, err)
		}
		incomeCat, err := categoryseed.CreditIncomeCategoryID(ctx, db, userID)
		if err != nil {
			return fmt.Errorf("credit income category for %s: %w", userID, err)
		}
		if _, err := q.RelinkCreditPaymentIncomeCategories(ctx, sqlcdb.RelinkCreditPaymentIncomeCategoriesParams{
			CategoryID:   &incomeCat,
			UpdatedAt:    nowStr,
			UserID:       userID,
			UserID_2:     userID,
			CategoryID_2: &incomeCat,
		}); err != nil {
			return fmt.Errorf("relink income categories for %s: %w", userID, err)
		}
	}
	return nil
}
