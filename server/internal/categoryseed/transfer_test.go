package categoryseed_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestNewUserTransferCategories(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u-transfer", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	var transferSystem int
	var transferIcon string
	err = mgr.DB().QueryRow(`
		SELECT is_system, icon FROM categories
		WHERE user_id = ? AND name = ? AND type = 'expense'`,
		userID, categoryseed.TransferCategoryName,
	).Scan(&transferSystem, &transferIcon)
	if err != nil {
		t.Fatal(err)
	}
	if transferSystem != 1 || transferIcon != "transfer" {
		t.Fatalf("expected system transfer icon, got is_system=%d icon=%q", transferSystem, transferIcon)
	}

	var perevodySystem int
	err = mgr.DB().QueryRow(`
		SELECT is_system FROM categories
		WHERE user_id = ? AND name = ? AND type = 'expense'`,
		userID, categoryseed.TransfersCategoryName,
	).Scan(&perevodySystem)
	if err != nil {
		t.Fatal(err)
	}
	if perevodySystem != 0 {
		t.Fatalf("expected user Переводы, got is_system=%d", perevodySystem)
	}
}

func TestMigrateManualExpenseFromLegacyTransfer(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u-migrate", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := mgr.DB()

	transferID, err := categoryseed.TransferCategoryID(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	perevodyID, err := categoryseed.TransfersCategoryID(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := sqlDB.ExecContext(ctx,
		`UPDATE categories SET is_system = 0, icon = 'default' WHERE id = ? AND user_id = ?`,
		transferID, userID,
	); err != nil {
		t.Fatal(err)
	}

	accountID := uuid.NewString()
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		accountID, userID,
	); err != nil {
		t.Fatal(err)
	}

	txID := uuid.NewString()
	past := timeutil.NowUTC().Add(-time.Hour).Format(time.RFC3339)
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, account_id, type, kind, amount, category_id,
			transaction_date, affects_balance, created_at, updated_at
		) VALUES (?, ?, ?, 'expense', 'manual', 5000, ?, ?, 1, ?, ?)`,
		txID, userID, accountID, transferID, past, past, past,
	); err != nil {
		t.Fatal(err)
	}

	if err := categoryseed.EnsureTransferCategory(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}
	if err := categoryseed.EnsureSystemCategories(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}

	var catID string
	var isSystem int
	err = sqlDB.QueryRow(`SELECT category_id FROM transactions WHERE id = ?`, txID).Scan(&catID)
	if err != nil {
		t.Fatal(err)
	}
	if catID != perevodyID {
		t.Fatalf("expected tx in Переводы %q, got %q", perevodyID, catID)
	}
	err = sqlDB.QueryRow(`
		SELECT is_system FROM categories WHERE id = ?`, transferID,
	).Scan(&isSystem)
	if err != nil {
		t.Fatal(err)
	}
	if isSystem != 1 {
		t.Fatalf("expected Перевод system, got %d", isSystem)
	}
}

func TestMigrateTransferSubcategories(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u-subs", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := mgr.DB()

	transferID, err := categoryseed.TransferCategoryID(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	perevodyID, err := categoryseed.TransfersCategoryID(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`UPDATE categories SET is_system = 0 WHERE id = ? AND user_id = ?`,
		transferID, userID,
	); err != nil {
		t.Fatal(err)
	}

	sub, err := category.CreateSubcategory(ctx, sqlDB, userID, transferID, "Родственники", "default")
	if err != nil {
		t.Fatal(err)
	}

	accountID := uuid.NewString()
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		accountID, userID,
	); err != nil {
		t.Fatal(err)
	}

	txID := uuid.NewString()
	past := timeutil.NowUTC().Add(-time.Hour).Format(time.RFC3339)
	subID := sub.ID
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, account_id, type, kind, amount, category_id, subcategory_id,
			transaction_date, affects_balance, created_at, updated_at
		) VALUES (?, ?, ?, 'expense', 'manual', 5000, ?, ?, ?, 1, ?, ?)`,
		txID, userID, accountID, transferID, subID, past, past, past,
	); err != nil {
		t.Fatal(err)
	}

	if err := categoryseed.EnsureTransferCategory(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}

	var subCatID string
	err = sqlDB.QueryRow(`SELECT category_id FROM subcategories WHERE id = ?`, subID).Scan(&subCatID)
	if err != nil {
		t.Fatal(err)
	}
	if subCatID != perevodyID {
		t.Fatalf("expected subcategory under Переводы, got %q", subCatID)
	}

	var txCat, txSub sql.NullString
	err = sqlDB.QueryRow(`SELECT category_id, subcategory_id FROM transactions WHERE id = ?`, txID).
		Scan(&txCat, &txSub)
	if err != nil {
		t.Fatal(err)
	}
	if txCat.String != perevodyID || txSub.String != subID {
		t.Fatalf("expected tx in Переводы with sub %q, got cat=%q sub=%q", subID, txCat.String, txSub.String)
	}

	var subsUnderTransfer int
	_ = sqlDB.QueryRow(`SELECT COUNT(*) FROM subcategories WHERE category_id = ?`, transferID).Scan(&subsUnderTransfer)
	if subsUnderTransfer != 0 {
		t.Fatalf("expected no subcategories under Перевод, got %d", subsUnderTransfer)
	}
}

func TestMigrateSubcategoryNameCollisionMerge(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u-merge", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := mgr.DB()

	transferID, _ := categoryseed.TransferCategoryID(ctx, sqlDB, userID)
	perevodyID, _ := categoryseed.TransfersCategoryID(ctx, sqlDB, userID)
	_, _ = sqlDB.ExecContext(ctx, `UPDATE categories SET is_system = 0 WHERE id = ?`, transferID)

	existing, err := category.CreateSubcategory(ctx, sqlDB, userID, perevodyID, "Друзья", "default")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := category.CreateSubcategory(ctx, sqlDB, userID, transferID, "Друзья", "default")
	if err != nil {
		t.Fatal(err)
	}

	accountID := uuid.NewString()
	_, _ = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		accountID, userID,
	)
	txID := uuid.NewString()
	past := timeutil.NowUTC().Add(-time.Hour).Format(time.RFC3339)
	_, _ = sqlDB.ExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, account_id, type, kind, amount, category_id, subcategory_id,
			transaction_date, affects_balance, created_at, updated_at
		) VALUES (?, ?, ?, 'expense', 'manual', 1000, ?, ?, ?, 1, ?, ?)`,
		txID, userID, accountID, transferID, legacy.ID, past, past, past,
	)

	if err := categoryseed.EnsureTransferCategory(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}

	var txSub string
	if err := sqlDB.QueryRow(`SELECT subcategory_id FROM transactions WHERE id = ?`, txID).Scan(&txSub); err != nil {
		t.Fatal(err)
	}
	if txSub != existing.ID {
		t.Fatalf("expected merged sub %q, got %q", existing.ID, txSub)
	}

	var legacyExists int
	_ = sqlDB.QueryRow(`SELECT COUNT(*) FROM subcategories WHERE id = ?`, legacy.ID).Scan(&legacyExists)
	if legacyExists != 0 {
		t.Fatal("expected legacy subcategory deleted after merge")
	}
}
