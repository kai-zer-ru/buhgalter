package transaction

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type seedEnv struct {
	userID    string
	accountID string
	account2  string
	expenseID string
	incomeID  string
}

func seedEnvFull(t *testing.T) (*db.Handle, seedEnv) {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "txuser", hash, "Tx User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	accountID := "acc-1"
	account2 := "acc-2"
	for _, row := range []struct{ id, name string }{
		{accountID, "Кошелёк"},
		{account2, "Банк"},
	} {
		_, err := sqlDB.ExecContext(ctx, `
			INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
			VALUES (?, ?, ?, 'cash', 100000, 'active', datetime('now'), datetime('now'))`,
			row.id, userID, row.name)
		if err != nil {
			t.Fatal(err)
		}
	}

	cats, err := category.ListByUser(ctx, sqlDB, userID, "")
	if err != nil {
		t.Fatal(err)
	}
	var expenseID, incomeID string
	for _, c := range cats {
		if c.Type == "expense" && expenseID == "" {
			expenseID = c.ID
		}
		if c.Type == "income" && incomeID == "" {
			incomeID = c.ID
		}
	}
	if expenseID == "" || incomeID == "" {
		t.Fatal("expected seeded categories")
	}
	if err := accountbalance.Refresh(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}

	return db.NewHandle(mgr), seedEnv{userID: userID, accountID: accountID, account2: account2, expenseID: expenseID, incomeID: incomeID}
}

func TestCreateUpdateDeleteExpense(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-24 * time.Hour)

	tx, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          2500,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Type != "expense" || tx.Amount != 2500 {
		t.Fatalf("unexpected tx: %+v", tx)
	}

	got, err := GetByID(ctx, database, env.userID, tx.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != tx.ID {
		t.Fatalf("GetByID: %+v", got)
	}

	updated, err := Update(ctx, database, env.userID, tx.ID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          3000,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 3000 {
		t.Fatalf("expected 3000, got %d", updated.Amount)
	}

	_, err = Update(ctx, database, env.userID, tx.ID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "income",
		Amount:          3000,
		CategoryID:      &env.incomeID,
		TransactionDate: past,
	})
	if err != ErrTypeChange {
		t.Fatalf("expected ErrTypeChange, got %v", err)
	}

	future := timeutil.NowUTC().Add(72 * time.Hour)
	updatedFuture, err := Update(ctx, database, env.userID, tx.ID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          3000,
		CategoryID:      &env.expenseID,
		TransactionDate: future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedFuture.Kind != "future" {
		t.Fatalf("expected future kind, got %s", updatedFuture.Kind)
	}

	if err := Delete(ctx, database, env.userID, tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetByID(ctx, database, env.userID, tx.ID); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCreateIncomeAndList(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-48 * time.Hour)

	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "income",
		Amount:          5000,
		CategoryID:      &env.incomeID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := List(ctx, database, env.userID, ListFilters{Type: "income", Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.Total < 1 || len(res.Data) < 1 {
		t.Fatalf("expected income rows, got %+v", res)
	}
}

func TestCreateValidationErrors(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC()

	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "transfer", Amount: 100, TransactionDate: past,
	})
	if err != ErrInvalidType {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}

	_, err = Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 0, TransactionDate: past,
	})
	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferCRUD(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-time.Hour)

	tr, err := CreateTransfer(ctx, database, env.userID, TransferInput{
		FromAccountID: env.accountID, ToAccountID: env.account2,
		Amount: 10000, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Legs) != 2 {
		t.Fatalf("expected 2 legs, got %d", len(tr.Legs))
	}

	got, err := GetTransfer(ctx, database, env.userID, tr.GroupID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 10000 {
		t.Fatalf("amount %d", got.Amount)
	}

	updated, err := UpdateTransfer(ctx, database, env.userID, tr.GroupID, TransferInput{
		FromAccountID: env.accountID, ToAccountID: env.account2,
		Amount: 15000, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 15000 {
		t.Fatalf("updated amount %d", updated.Amount)
	}

	if err := DeleteTransfer(ctx, database, env.userID, tr.GroupID); err != nil {
		t.Fatal(err)
	}
	_, err = GetTransfer(ctx, database, env.userID, tr.GroupID)
	if err != ErrTransferNotFound {
		t.Fatalf("expected transfer not found, got %v", err)
	}
}

func TestTransferWithCommission(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-time.Hour)

	tr, err := CreateTransfer(ctx, database, env.userID, TransferInput{
		FromAccountID: env.accountID, ToAccountID: env.account2,
		Amount: 10000, Commission: 500, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Commission != 500 {
		t.Fatalf("commission %d", tr.Commission)
	}
	if len(tr.Legs) != 2 {
		t.Fatalf("expected 2 transfer legs, got %d", len(tr.Legs))
	}

	bal, err := Balance(ctx, database, env.userID, env.accountID, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if bal != 89500 {
		t.Fatalf("from balance %d, want 89500", bal)
	}

	updated, err := UpdateTransfer(ctx, database, env.userID, tr.GroupID, TransferInput{
		FromAccountID: env.accountID, ToAccountID: env.account2,
		Amount: 10000, Commission: 0, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Commission != 0 {
		t.Fatalf("updated commission %d", updated.Commission)
	}
	bal, err = Balance(ctx, database, env.userID, env.accountID, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if bal != 90000 {
		t.Fatalf("from balance after commission removal %d, want 90000", bal)
	}

	if err := DeleteTransfer(ctx, database, env.userID, tr.GroupID); err != nil {
		t.Fatal(err)
	}
}

func TestActivateFutureTransaction(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	future := timeutil.NowUTC().Add(72 * time.Hour)

	tx, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          1000,
		CategoryID:      &env.expenseID,
		TransactionDate: future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Kind != "future" {
		t.Fatalf("expected future, got %s", tx.Kind)
	}

	activated, err := Activate(ctx, database, env.userID, tx.ID)
	if err != nil {
		t.Fatal(err)
	}
	if activated.Kind != "manual" {
		t.Fatalf("expected manual after activate, got %s", activated.Kind)
	}
}

func TestCreateFutureForSystemCategoryRejected(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	future := timeutil.NowUTC().Add(72 * time.Hour)

	cats, err := category.ListByUser(ctx, database, env.userID, "expense")
	if err != nil {
		t.Fatal(err)
	}
	var systemExpenseID string
	for _, c := range cats {
		if c.IsSystem {
			systemExpenseID = c.ID
			break
		}
	}
	if systemExpenseID == "" {
		t.Fatal("expected system expense category")
	}

	_, err = Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          1000,
		CategoryID:      &systemExpenseID,
		TransactionDate: future,
	})
	if !errors.Is(err, ErrSystemCategoryPlanned) {
		t.Fatalf("expected ErrSystemCategoryPlanned, got %v", err)
	}
}

func TestAccountsSummaryAndDashboard(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-time.Hour)

	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 5000,
		CategoryID: &env.expenseID, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := AccountsSummaryForUser(ctx, database, env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalBalance != 195000 {
		t.Fatalf("total balance %d", summary.TotalBalance)
	}

	dash, err := DashboardForUser(ctx, database, env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if dash.TotalBalance != 195000 {
		t.Fatalf("dashboard balance %d", dash.TotalBalance)
	}
	if len(dash.RecentTransactions) < 1 {
		t.Fatal("expected recent transactions")
	}
}

func TestListRecent(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-time.Hour)

	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "income", Amount: 100,
		CategoryID: &env.incomeID, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	recent, err := ListRecent(ctx, database, env.userID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) < 1 {
		t.Fatal("expected recent list")
	}
}

func TestListWithFilters(t *testing.T) {
	handle, env := seedEnvFull(t)
	ctx := context.Background()
	database := handle.DB()
	past := timeutil.NowUTC().Add(-time.Hour)

	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 999,
		CategoryID: &env.expenseID, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := List(ctx, database, env.userID, ListFilters{
		AccountID: env.accountID, Type: "expense", Kind: "manual",
		From: "2020-01-01 00:00:00", To: "2030-12-31 23:59:59", Sort: "date_desc",
		Page: 1, Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.Total < 1 {
		t.Fatalf("meta %+v", res.Meta)
	}

	resAsc, err := List(ctx, database, env.userID, ListFilters{Sort: "date_asc", Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resAsc.Data) < 1 {
		t.Fatal("expected asc list")
	}

	desc := "уникальный-поиск-xyz"
	_, err = Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 100,
		CategoryID: &env.expenseID, TransactionDate: past, Description: &desc,
	})
	if err != nil {
		t.Fatal(err)
	}
	resSearch, err := List(ctx, database, env.userID, ListFilters{
		Search: "уникальный-поиск", Page: 1, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resSearch.Meta.Total < 1 {
		t.Fatalf("search meta %+v", resSearch.Meta)
	}
}

func TestCreateInvalidCategory(t *testing.T) {
	handle, env := seedEnvFull(t)
	ctx := context.Background()
	database := handle.DB()
	badCat := "bad-cat"
	_, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 100,
		CategoryID: &badCat, TransactionDate: timeutil.NowUTC(),
	})
	if err == nil {
		t.Fatal("expected category error")
	}
}

func TestCreateWithSubcategoryName(t *testing.T) {
	handle, env := seedEnvFull(t)
	ctx := context.Background()
	database := handle.DB()
	subName := "Автобус"
	tx, err := Create(ctx, database, env.userID, CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: 500,
		CategoryID: &env.expenseID, SubcategoryName: &subName,
		TransactionDate: timeutil.NowUTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.SubcategoryID == nil || *tx.SubcategoryID == "" {
		t.Fatal("expected subcategory id")
	}
}
