package merchant_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func setupOps(t *testing.T) (context.Context, *sql.DB, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "mops", "hash", "M", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, mgr.DB(), userID
}

func mustCash(t *testing.T, ctx context.Context, database *sql.DB, userID string) string {
	t.Helper()
	acc, err := account.Create(ctx, database, userID, account.CreateInput{
		Name: "Наличные", Type: "cash", InitialBalance: 10_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	return acc.ID
}

func hasAlias(aliases []string, name string) bool {
	for _, alias := range aliases {
		if strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

func TestJournalUsesPrimaryNameForAlias(t *testing.T) {
	ctx, database, userID := setupOps(t)
	accountID := mustCash(t, ctx, database, userID)

	m, err := merchant.CreateWithAliases(ctx, database, userID, "Пятёрочка", "shopping", []string{"Pyaterochka"})
	if err != nil {
		t.Fatal(err)
	}
	txAlias := "pyaterochka"
	tx, err := transaction.Create(ctx, database, userID, transaction.CreateInput{
		AccountID: accountID, Type: "expense", Amount: 1500,
		MerchantName: &txAlias, TransactionDate: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.MerchantID == nil || *tx.MerchantID != m.ID || tx.MerchantName == nil || *tx.MerchantName != "Пятёрочка" {
		t.Fatalf("tx by alias: id=%v name=%v", tx.MerchantID, tx.MerchantName)
	}
	listed, err := transaction.List(ctx, database, userID, transaction.ListFilters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Data) != 1 || listed.Data[0].MerchantName == nil || *listed.Data[0].MerchantName != "Пятёрочка" {
		t.Fatalf("journal=%+v", listed.Data)
	}
}

func TestMergeMovesOperationsAndKeepsTargetName(t *testing.T) {
	ctx, database, userID := setupOps(t)
	accountID := mustCash(t, ctx, database, userID)

	target, err := merchant.CreateWithAliases(ctx, database, userID, "Пятёрочка", "shopping", []string{"Pyaterochka"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := merchant.CreateWithAliases(ctx, database, userID, "Магнит", "default", []string{"Magnit"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO merchant_aliases (id, user_id, merchant_id, name, created_at)
		VALUES ('alias-primary-dup', ?, ?, ?, '2026-01-01T00:00:00Z')`,
		userID, source.ID, target.Name); err != nil {
		t.Fatal(err)
	}
	extra, err := merchant.Create(ctx, database, userID, "Лента", "default")
	if err != nil {
		t.Fatal(err)
	}
	sourceName := source.Name
	txSource, err := transaction.Create(ctx, database, userID, transaction.CreateInput{
		AccountID: accountID, Type: "expense", Amount: 900,
		MerchantName: &sourceName, TransactionDate: time.Now().UTC().Add(-2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO transaction_templates (id, user_id, name, type, merchant_id, created_at, updated_at)
		VALUES ('tpl-merge', ?, 'шаблон', 'expense', ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		userID, source.ID); err != nil {
		t.Fatal(err)
	}

	merged, err := merchant.Merge(ctx, database, userID, target.ID, []string{source.ID, source.ID, extra.ID})
	if err != nil {
		t.Fatal(err)
	}
	if merged.ID != target.ID || merged.Name != "Пятёрочка" || merged.Icon != "shopping" {
		t.Fatalf("target changed: %+v", merged)
	}
	if !hasAlias(merged.Aliases, "Магнит") || !hasAlias(merged.Aliases, "Magnit") || !hasAlias(merged.Aliases, "Лента") || !hasAlias(merged.Aliases, "Pyaterochka") {
		t.Fatalf("aliases=%v", merged.Aliases)
	}
	if hasAlias(merged.Aliases, "Пятёрочка") {
		t.Fatalf("primary name stored as alias: %v", merged.Aliases)
	}
	if _, err := merchant.Get(ctx, database, userID, source.ID); !errors.Is(err, merchant.ErrNotFound) {
		t.Fatalf("source still exists: %v", err)
	}
	if _, err := merchant.Get(ctx, database, userID, extra.ID); !errors.Is(err, merchant.ErrNotFound) {
		t.Fatalf("extra still exists: %v", err)
	}
	reloaded, err := transaction.GetByID(ctx, database, userID, txSource.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.MerchantName == nil || *reloaded.MerchantName != "Пятёрочка" || reloaded.MerchantID == nil || *reloaded.MerchantID != target.ID {
		t.Fatalf("journal after merge: %+v", reloaded)
	}
	var templateMerchant string
	if err := database.QueryRowContext(ctx, `SELECT merchant_id FROM transaction_templates WHERE id = 'tpl-merge'`).Scan(&templateMerchant); err != nil {
		t.Fatal(err)
	}
	if templateMerchant != target.ID {
		t.Fatalf("template merchant=%s", templateMerchant)
	}

	if _, err := merchant.Merge(ctx, database, userID, target.ID, nil); !errors.Is(err, merchant.ErrInvalidMerge) {
		t.Fatalf("empty merge: %v", err)
	}
	if _, err := merchant.Merge(ctx, database, userID, target.ID, []string{target.ID}); !errors.Is(err, merchant.ErrInvalidMerge) {
		t.Fatalf("self merge: %v", err)
	}
}

func TestMergeConflictRollsBack(t *testing.T) {
	ctx, database, userID := setupOps(t)
	accountID := mustCash(t, ctx, database, userID)

	target, err := merchant.Create(ctx, database, userID, "А", "default")
	if err != nil {
		t.Fatal(err)
	}
	source, err := merchant.Create(ctx, database, userID, "Б", "default")
	if err != nil {
		t.Fatal(err)
	}
	third, err := merchant.Create(ctx, database, userID, "В", "default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO merchant_aliases (id, user_id, merchant_id, name, created_at)
		VALUES ('alias-taken', ?, ?, ?, '2026-01-01T00:00:00Z')`,
		userID, third.ID, source.Name); err != nil {
		t.Fatal(err)
	}
	sourceName := source.Name
	tx, err := transaction.Create(ctx, database, userID, transaction.CreateInput{
		AccountID: accountID, Type: "expense", Amount: 400,
		MerchantName: &sourceName, TransactionDate: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = merchant.Merge(ctx, database, userID, target.ID, []string{source.ID})
	if !errors.Is(err, merchant.ErrNameTaken) {
		t.Fatalf("merge conflict: %v", err)
	}
	if _, err := merchant.Get(ctx, database, userID, source.ID); err != nil {
		t.Fatalf("source deleted on conflict: %v", err)
	}
	if _, err := merchant.Get(ctx, database, userID, target.ID); err != nil {
		t.Fatal(err)
	}
	reloaded, err := transaction.GetByID(ctx, database, userID, tx.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.MerchantID == nil || *reloaded.MerchantID != source.ID || reloaded.MerchantName == nil || *reloaded.MerchantName != source.Name {
		t.Fatalf("tx moved on rollback: %+v", reloaded)
	}
	targetGot, err := merchant.Get(ctx, database, userID, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetGot.Aliases) != 0 {
		t.Fatalf("target aliases changed: %+v", targetGot.Aliases)
	}
}
