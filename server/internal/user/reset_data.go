package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// ResetUserData deletes the user's ledger (accounts, transactions, debts, credits,
// budgets, subscriptions, dictionaries) and reseeds default categories.
// Profile, password, sessions, API tokens and notification channel settings stay.
func ResetUserData(ctx context.Context, database *sql.DB, userID string) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	steps := []struct {
		name string
		fn   func() error
	}{
		{"credit payments", func() error { return q.DeleteUserCreditPayments(ctx, userID) }},
		{"debt transactions", func() error { return q.DeleteUserDebtTransactions(ctx, userID) }},
		{"credits", func() error { return q.DeleteUserCredits(ctx, userID) }},
		{"debts", func() error { return q.DeleteUserDebts(ctx, userID) }},
		{"debtors", func() error { return q.DeleteUserDebtors(ctx, userID) }},
		{"subscriptions", func() error { return q.DeleteUserSubscriptions(ctx, userID) }},
		{"recurring", func() error { return q.DeleteUserRecurringOperations(ctx, userID) }},
		{"budgets", func() error { return q.DeleteUserBudgets(ctx, userID) }},
		{"templates", func() error { return q.DeleteUserTransactionTemplates(ctx, userID) }},
		{"transactions", func() error { return q.DeleteUserTransactions(ctx, userID) }},
		{"merchants", func() error { return q.DeleteUserMerchants(ctx, userID) }},
		{"tags", func() error { return q.DeleteUserTags(ctx, userID) }},
		{"account refs", func() error { return q.ClearUserAccountSelfRefs(ctx, userID) }},
		{"accounts", func() error { return q.DeleteUserAccounts(ctx, userID) }},
		{"categories", func() error { return q.DeleteUserCategories(ctx, userID) }},
		{"import jobs", func() error { return q.DeleteUserImportJobs(ctx, userID) }},
		{"import keys", func() error { return q.DeleteUserImportIdempotency(ctx, userID) }},
		{"change events", func() error { return q.DeleteUserChangeEvents(ctx, userID) }},
		{"notification log", func() error { return q.DeleteUserNotificationLog(ctx, userID) }},
	}
	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	if err := categoryseed.SeedDefaults(ctx, tx, userID); err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
