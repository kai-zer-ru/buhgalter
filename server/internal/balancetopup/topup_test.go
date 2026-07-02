package balancetopup_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/balancehooks"
	"github.com/kai-zer-ru/buhgalter/internal/balancetopup"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func seedAutoTopupEnv(t *testing.T) (context.Context, *sql.DB, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	sqlDB := mgr.DB()

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, sqlDB, "topupuser", hash, "Topup User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	if err := bank.SeedIfEmpty(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	banks, err := bank.ListAll(ctx, sqlDB)
	if err != nil || len(banks) == 0 {
		t.Fatal("expected seeded bank")
	}
	bankIDStr := banks[0].ID

	targetID := "acc-target"
	sourceID := "acc-source"
	for _, row := range []struct {
		id, name string
		bal      int64
	}{
		{targetID, "Яндекс", 250000},
		{sourceID, "Сбер", 500000},
	} {
		_, err := sqlDB.ExecContext(ctx, `
			INSERT INTO accounts (id, user_id, name, type, bank_id, initial_balance, current_balance, status, created_at, updated_at)
			VALUES (?, ?, ?, 'bank', ?, ?, ?, 'active', datetime('now'), datetime('now'))`,
			row.id, userID, row.name, bankIDStr, row.bal, row.bal)
		if err != nil {
			t.Fatal(err)
		}
	}

	if _, err := category.ListByUser(ctx, sqlDB, userID, ""); err != nil {
		t.Fatal(err)
	}
	if err := accountbalance.Refresh(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}

	_, err = account.Update(ctx, sqlDB, userID, targetID, account.UpdateInput{
		Name:   "Яндекс",
		BankID: &bankIDStr,
		AutoTopup: &account.AutoTopupInput{
			Enabled:         true,
			Threshold:       300000,
			Target:          500000,
			SourceAccountID: sourceID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, sqlDB, userID, targetID, sourceID
}

func TestApplyIfNeededCreatesTransfer(t *testing.T) {
	ctx, sqlDB, userID, targetID, sourceID := seedAutoTopupEnv(t)

	applied, err := balancetopup.ApplyIfNeeded(ctx, sqlDB, userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected auto topup transfer")
	}

	rows, err := sqlDB.QueryContext(ctx, `
		SELECT description FROM transactions
		WHERE user_id = ? AND type = 'transfer' AND account_id = ?`,
		userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected transfer leg on target account")
	}
	var desc string
	if err := rows.Scan(&desc); err != nil {
		t.Fatal(err)
	}
	if desc != balancetopup.Description {
		t.Fatalf("description %q, want %q", desc, balancetopup.Description)
	}

	target, err := account.GetByID(ctx, sqlDB, userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if target.Balance != 500000 {
		t.Fatalf("target balance %d, want 500000", target.Balance)
	}
	source, err := account.GetByID(ctx, sqlDB, userID, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Balance != 250000 {
		t.Fatalf("source balance %d, want 250000", source.Balance)
	}
}

func TestApplyIfNeededDisablesWhenSourceInsufficient(t *testing.T) {
	ctx, sqlDB, userID, targetID, sourceID := seedAutoTopupEnv(t)

	_, err := sqlDB.ExecContext(ctx, `UPDATE accounts SET current_balance = 100000 WHERE id = ?`, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `UPDATE accounts SET current_balance = 100000 WHERE id = ?`, targetID)
	if err != nil {
		t.Fatal(err)
	}

	applied, err := balancetopup.ApplyIfNeeded(ctx, sqlDB, userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected no transfer")
	}
	acc, err := account.GetByID(ctx, sqlDB, userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if acc.AutoTopupEnabled {
		t.Fatal("expected auto topup disabled")
	}
}

func TestApplyIfNeededNoOpForCashAccount(t *testing.T) {
	ctx, sqlDB, userID, _, _ := seedAutoTopupEnv(t)
	cashID := "acc-cash"
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 0, 'active', datetime('now'), datetime('now'))`,
		cashID, userID)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := balancetopup.ApplyIfNeeded(ctx, sqlDB, userID, cashID)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("cash account must not auto top up")
	}
}

func TestCheckAfterRefreshViaTransactionHook(t *testing.T) {
	ctx, sqlDB, userID, targetID, _ := seedAutoTopupEnv(t)
	transaction.AfterBalanceRefresh = nil
	balancehooks.AfterRefresh = balancetopup.CheckAfterRefresh
	t.Cleanup(func() { balancehooks.AfterRefresh = nil })

	cats, err := category.ListByUser(ctx, sqlDB, userID, "")
	if err != nil {
		t.Fatal(err)
	}
	var expenseID string
	for _, c := range cats {
		if c.Type == "expense" {
			expenseID = c.ID
			break
		}
	}
	if expenseID == "" {
		t.Fatal("missing expense category")
	}

	_, err = transaction.Create(ctx, sqlDB, userID, transaction.CreateInput{
		AccountID:       targetID,
		Type:            "expense",
		Amount:          200000,
		CategoryID:      &expenseID,
		TransactionDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := account.GetByID(ctx, sqlDB, userID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Balance != 500000 {
		t.Fatalf("balance after hook %d, want 500000", acc.Balance)
	}
}

func TestUpdateRejectsAutoTopupForCash(t *testing.T) {
	ctx, sqlDB, userID, _, _ := seedAutoTopupEnv(t)
	cashID := "acc-cash2"
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 0, 'active', datetime('now'), datetime('now'))`,
		cashID, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = account.Update(ctx, sqlDB, userID, cashID, account.UpdateInput{
		Name: "Cash",
		AutoTopup: &account.AutoTopupInput{
			Enabled:         true,
			Threshold:       100000,
			Target:          200000,
			SourceAccountID: "acc-source",
		},
	})
	if err == nil {
		t.Fatal("expected error for cash auto topup")
	}
	if err != account.ErrAutoTopupNotAllowed {
		t.Fatalf("got %v, want ErrAutoTopupNotAllowed", err)
	}
}
