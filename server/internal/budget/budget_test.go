package budget_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/budget"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/stats"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func seedBudgetEnv(t *testing.T) (context.Context, *sql.DB, string, string, string) {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "budgetuser", hash, "Budget", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	accountID := "acc-budget"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Кошелёк', 'cash', 1000000, 1000000, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
	if err != nil {
		t.Fatal(err)
	}

	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) == 0 {
		t.Fatal("expected expense categories")
	}
	return ctx, sqlDB, userID, accountID, cats[0].ID
}

func TestBudgetCRUDAndSummary(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	b, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name:           "Продукты",
		Scope:          budget.ScopeCategory,
		CategoryID:     &cat,
		Amount:         30_000,
		AlertAtPercent: 80,
		IsActive:       true,
		Month:          month,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Amount != 30_000 {
		t.Fatalf("expected amount 30000, got %d", b.Amount)
	}

	dupCat := categoryID
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name:       "Дубликат",
		Scope:      budget.ScopeCategory,
		CategoryID: &dupCat,
		Amount:     10_000,
		IsActive:   true,
		Month:      month,
	})
	if err != budget.ErrDuplicateActive {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	now := timeutil.NowUTC()
	txDate := timeutil.FormatUTC(now)
	txID := "tx-budget-1"
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: txID, UserID: userID, AccountID: accountID, Type: "expense", Kind: "manual",
		Amount: 5_000, CategoryID: &cat, TransactionDate: txDate, AffectsBalance: 1,
		CreatedAt: txDate, UpdatedAt: txDate,
	}); err != nil {
		t.Fatal(err)
	}

	items, err := budget.Summary(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	if len(items.Items) != 1 {
		t.Fatalf("expected 1 summary item, got %d", len(items.Items))
	}
	if items.Items[0].Spent != 5_000 {
		t.Fatalf("expected spent 5000, got %d", items.Items[0].Spent)
	}
	if items.Items[0].Status != budget.StatusOK {
		t.Fatalf("expected status ok, got %s", items.Items[0].Status)
	}

	periodStart, periodEnd, err := budget.MonthBounds(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	st, err := stats.New(sqlDB).Summary(ctx, userID, stats.Filters{
		From: periodStart, To: periodEnd, Type: "expense", CategoryID: categoryID,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if st.ExpenseTotal != items.Items[0].Spent {
		t.Fatalf("stats expense %d != budget spent %d", st.ExpenseTotal, items.Items[0].Spent)
	}

	b.Amount = 40_000
	updated, err := budget.Update(ctx, sqlDB, userID, b.ID, budget.Input{
		Name: b.Name, Scope: b.Scope, CategoryID: &cat, Amount: 40_000,
		AlertAtPercent: 80, IsActive: true, Month: month,
	}, month)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 40_000 {
		t.Fatalf("expected updated amount 40000")
	}

	if err := budget.Delete(ctx, sqlDB, userID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Get(ctx, sqlDB, userID, b.ID); err != budget.ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestComputeStatus(t *testing.T) {
	p, s := budget.ComputeStatus(50, 100, 80)
	if p != 50 || s != budget.StatusOK {
		t.Fatalf("got %d %s", p, s)
	}
	p, s = budget.ComputeStatus(80, 100, 80)
	if p != 80 || s != budget.StatusWarning {
		t.Fatalf("got %d %s", p, s)
	}
	p, s = budget.ComputeStatus(120, 100, 80)
	if p != 120 || s != budget.StatusExceeded {
		t.Fatalf("got %d %s", p, s)
	}
}

func TestAllExpenseBudget(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Все расходы", Scope: budget.ScopeAllExpense, Amount: 100_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	txDate := timeutil.FormatUTC(timeutil.NowUTC())
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: "tx-all", UserID: userID, AccountID: "acc-budget", Type: "expense", Kind: "manual",
		Amount: 2_000, CategoryID: &cat, TransactionDate: txDate, AffectsBalance: 1,
		CreatedAt: txDate, UpdatedAt: txDate,
	}); err != nil {
		t.Fatal(err)
	}
	items, err := budget.Summary(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	var all *budget.SummaryItem
	for i := range items.Items {
		if items.Items[i].Scope == budget.ScopeAllExpense {
			all = &items.Items[i]
			break
		}
	}
	if all == nil || all.Spent != 2_000 {
		t.Fatalf("expected all_expense spent 2000, got %+v", all)
	}
}

func TestBudgetExcludesTransferCommission(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Все расходы", Scope: budget.ScopeAllExpense, Amount: 100_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	txDate := timeutil.FormatUTC(timeutil.NowUTC())
	groupID := "tg-budget-test"
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: "tx-regular", UserID: userID, AccountID: accountID, Type: "expense", Kind: "manual",
		Amount: 5_000, CategoryID: &cat, TransactionDate: txDate, AffectsBalance: 1,
		CreatedAt: txDate, UpdatedAt: txDate,
	}); err != nil {
		t.Fatal(err)
	}
	commissionCat := cat // any category; transfer_group_id is what matters
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: "tx-commission", UserID: userID, AccountID: accountID, Type: "expense", Kind: "manual",
		Amount: 500, CategoryID: &commissionCat, TransferGroupID: &groupID,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: txDate, UpdatedAt: txDate,
	}); err != nil {
		t.Fatal(err)
	}
	items, err := budget.Summary(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	var all *budget.SummaryItem
	for i := range items.Items {
		if items.Items[i].Scope == budget.ScopeAllExpense {
			all = &items.Items[i]
			break
		}
	}
	if all == nil || all.Spent != 5_000 {
		t.Fatalf("expected all_expense spent 5000 (no transfer commission), got %+v", all)
	}
}

func TestSummaryAllExpenseFirstWithChildrenTotals(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Категория", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 5_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Общий", Scope: budget.ScopeAllExpense,
		Amount: 20_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := budget.Summary(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if res.Items[0].Scope != budget.ScopeAllExpense {
		t.Fatalf("expected all_expense first, got %s", res.Items[0].Scope)
	}
	if res.Items[0].ChildrenPlanned != 5_000 {
		t.Fatalf("children planned %d", res.Items[0].ChildrenPlanned)
	}
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Дубликат общий", Scope: budget.ScopeAllExpense,
		Amount: 10_000, IsActive: true, Month: month,
	})
	if err != budget.ErrDuplicateActive {
		t.Fatalf("expected duplicate all_expense, got %v", err)
	}
}

func TestPreviewSpentMatchesSummary(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	acc := accountID
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Продукты", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 30_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	txDate := timeutil.FormatUTC(timeutil.NowUTC())
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: "tx-preview-1", UserID: userID, AccountID: accountID, Type: "expense", Kind: "manual",
		Amount: 7_500, CategoryID: &cat, TransactionDate: txDate, AffectsBalance: 1,
		CreatedAt: txDate, UpdatedAt: txDate,
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := budget.Summary(ctx, sqlDB, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Items) != 1 || summary.Items[0].Spent != 7_500 {
		t.Fatalf("summary spent unexpected: %+v", summary.Items)
	}

	preview, err := budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: budget.ScopeCategory, CategoryID: &cat,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Spent != summary.Items[0].Spent {
		t.Fatalf("preview spent %d != summary %d", preview.Spent, summary.Items[0].Spent)
	}
	if preview.SpentDisplay != summary.Items[0].SpentDisplay {
		t.Fatalf("preview display %q != summary %q", preview.SpentDisplay, summary.Items[0].SpentDisplay)
	}

	previewAcc, err := budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: budget.ScopeCategory, CategoryID: &cat, AccountID: &acc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if previewAcc.Spent != 7_500 {
		t.Fatalf("expected account filter spent 7500, got %d", previewAcc.Spent)
	}

	otherAcc := "acc-other"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Другой', 'cash', 0, 0, 'active', datetime('now'), datetime('now'))`,
		otherAcc, userID)
	if err != nil {
		t.Fatal(err)
	}
	previewOther, err := budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: budget.ScopeCategory, CategoryID: &cat, AccountID: &otherAcc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if previewOther.Spent != 0 {
		t.Fatalf("expected other account spent 0, got %d", previewOther.Spent)
	}

	allPreview, err := budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: budget.ScopeAllExpense,
	})
	if err != nil {
		t.Fatal(err)
	}
	if allPreview.Spent != 7_500 {
		t.Fatalf("expected all_expense preview 7500, got %d", allPreview.Spent)
	}

	_, err = budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: "nope",
	})
	if err != budget.ErrInvalidScope {
		t.Fatalf("expected invalid scope, got %v", err)
	}

	_, err = budget.PreviewSpent(ctx, sqlDB, userID, budget.SpentPreviewInput{
		Month: month, Scope: budget.ScopeCategory,
	})
	if err != budget.ErrInvalidCategory {
		t.Fatalf("expected missing category, got %v", err)
	}
}

func TestBudgetCopyForwardOnlyNextMonth(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, err := budget.CurrentMonthQuery(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	next, err := budget.AddMonths(month, 1)
	if err != nil {
		t.Fatal(err)
	}
	next2, err := budget.AddMonths(month, 2)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	_, err = budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Копируемый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sumNext, err := budget.Summary(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(sumNext.Items) != 1 {
		t.Fatalf("expected auto-copy to next month, got %d items", len(sumNext.Items))
	}
	if sumNext.Items[0].CopyForward {
		t.Fatal("copied budget should have copy_forward=false")
	}
	sumNext2, err := budget.Summary(ctx, sqlDB, userID, next2)
	if err != nil {
		t.Fatal(err)
	}
	if len(sumNext2.Items) != 0 {
		t.Fatalf("expected no budget two months ahead, got %d", len(sumNext2.Items))
	}
}
