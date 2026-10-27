package budget_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/budget"
	"github.com/kai-zer-ru/buhgalter/internal/budgetnotify"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/realtime"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestBudgetCopyForwardEmptyNextMonth(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	month, next, next2 := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	created, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Октябрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := budget.Update(ctx, sqlDB, userID, created.ID, budget.Input{
		Name: "Октябрь новый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 20_000, AccountID: &accountID, AlertAtPercent: 70,
		IsActive: true, CopyForward: true, Month: month,
	}, month)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CopyForward {
		t.Fatal("source copy_forward should stay on")
	}
	nextItems := mustList(t, ctx, sqlDB, userID, next)
	if len(nextItems) != 1 {
		t.Fatalf("expected 1 next-month budget, got %d", len(nextItems))
	}
	got := nextItems[0]
	if got.Name != "Октябрь новый" || got.Amount != 20_000 || got.AlertAtPercent != 70 || got.CopyForward || !got.IsActive {
		t.Fatalf("next month fields: %+v", got)
	}
	if got.AccountID == nil || *got.AccountID != accountID {
		t.Fatalf("next month account: %+v", got.AccountID)
	}
	if got.Period == nil || got.Period.PlannedAmount != 20_000 {
		t.Fatalf("next month planned: %+v", got.Period)
	}
	if len(mustList(t, ctx, sqlDB, userID, next2)) != 0 {
		t.Fatal("month after next should stay empty")
	}
}

func TestBudgetCopyForwardUpdatesExisting(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	month, next, next2 := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Октябрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	nextRow, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Ноябрь старый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 5_000, IsActive: true, Month: next, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Декабрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 7_777, IsActive: true, Month: next2, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	start, _, err := budget.MonthBounds(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sqlDB.ExecContext(ctx, `
		UPDATE budget_periods SET planned_amount = ?, rollover_amount = ?
		WHERE budget_id = ? AND period_start = ?`, 1_111, 2_222, nextRow.ID, start)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("expected to rewrite next period, updated %d", n)
	}
	now := timeutil.FormatUTC(timeutil.NowUTC())
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO budget_alert_sent (budget_id, period_start, threshold_percent, sent_at)
		VALUES (?, ?, 90, ?)`, nextRow.ID, start, now); err != nil {
		t.Fatal(err)
	}
	if err := sqlcdb.New(sqlDB).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: "tx-copy-forward", UserID: userID, AccountID: accountID, Type: "expense", Kind: "manual",
		Amount: 3_000, CategoryID: &cat, TransactionDate: start, AffectsBalance: 1,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	decBefore, err := budget.Get(ctx, sqlDB, userID, dec.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
		Name: "Октябрь новый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 90_000, AccountID: &accountID, AlertAtPercent: 70,
		IsActive: true, CopyForward: true,
	}, month); err != nil {
		t.Fatal(err)
	}

	got := mustOne(t, mustList(t, ctx, sqlDB, userID, next), "Октябрь новый")
	if got.ID != nextRow.ID || got.CopyForward || got.Amount != 90_000 || got.AlertAtPercent != 70 {
		t.Fatalf("updated next month: %+v", got)
	}
	if got.AccountID == nil || *got.AccountID != accountID {
		t.Fatalf("account not copied: %+v", got.AccountID)
	}
	if got.Period == nil || got.Period.PlannedAmount != 90_000 || got.Period.RolloverAmount != 2_222 {
		t.Fatalf("planned/rollover: %+v", got.Period)
	}
	sum, err := budget.Summary(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Items) != 1 || sum.Items[0].Spent != 3_000 {
		t.Fatalf("spent changed: %+v", sum.Items)
	}
	var alerts int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM budget_alert_sent WHERE budget_id = ?`, nextRow.ID).Scan(&alerts); err != nil {
		t.Fatal(err)
	}
	if alerts != 1 {
		t.Fatalf("alert rows = %d", alerts)
	}
	var txAmount int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT amount FROM transactions WHERE id = ?`, "tx-copy-forward").Scan(&txAmount); err != nil {
		t.Fatal(err)
	}
	if txAmount != 3_000 {
		t.Fatalf("expense amount %d", txAmount)
	}
	decAfter, err := budget.Get(ctx, sqlDB, userID, dec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if decAfter.Amount != decBefore.Amount || decAfter.Name != decBefore.Name || decAfter.CopyForward != decBefore.CopyForward || decAfter.UpdatedAt != decBefore.UpdatedAt {
		t.Fatalf("december changed: before %+v after %+v", decBefore, decAfter)
	}
	srcAfter, err := budget.Get(ctx, sqlDB, userID, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !srcAfter.CopyForward {
		t.Fatal("source flag should stay on")
	}
}

func TestBudgetCopyForwardOnlyImmediateNext(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, next2 := adjacentMonths(t, ctx, sqlDB, userID)
	next3, err := budget.AddMonths(month, 3)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Октябрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, spec := range []struct {
		month, name string
		amount      int64
	}{
		{next, "Ноябрь", 2_000},
		{next2, "Декабрь", 3_000},
		{next3, "Январь", 4_000},
	} {
		row, err := budget.Create(ctx, sqlDB, userID, budget.Input{
			Name: spec.name, Scope: budget.ScopeCategory, CategoryID: &cat,
			Amount: spec.amount, IsActive: true, Month: spec.month, CopyForward: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[spec.month] = row.ID
	}
	before := map[string]budget.Budget{}
	for m, id := range ids {
		row, err := budget.Get(ctx, sqlDB, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		before[m] = row
	}
	if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
		Name: "Октябрь новый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 9_000, AlertAtPercent: 90, IsActive: true, CopyForward: true,
	}, month); err != nil {
		t.Fatal(err)
	}
	nov := mustOne(t, mustList(t, ctx, sqlDB, userID, next), "Октябрь новый")
	if nov.Amount != 9_000 || nov.CopyForward {
		t.Fatalf("november: %+v", nov)
	}
	for _, m := range []string{next2, next3} {
		after, err := budget.Get(ctx, sqlDB, userID, ids[m])
		if err != nil {
			t.Fatal(err)
		}
		was := before[m]
		if after.Amount != was.Amount || after.Name != was.Name || after.UpdatedAt != was.UpdatedAt || after.CopyForward != was.CopyForward {
			t.Fatalf("%s changed: %+v", m, after)
		}
	}
	var total int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM budgets WHERE user_id = ?`, userID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("expected 4 budgets, got %d", total)
	}
}

func TestBudgetCopyForwardOffLeavesNext(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Октябрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	nextRow, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Ноябрь", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 5_000, IsActive: true, Month: next, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := budget.Get(ctx, sqlDB, userID, nextRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
		Name: "Октябрь другой", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 8_000, AlertAtPercent: 90, IsActive: true, CopyForward: false,
	}, month); err != nil {
		t.Fatal(err)
	}
	after, err := budget.Get(ctx, sqlDB, userID, nextRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Amount != before.Amount || after.Name != before.Name || after.CopyForward != before.CopyForward || after.UpdatedAt != before.UpdatedAt {
		t.Fatalf("next month changed: %+v", after)
	}
}

func TestBudgetCopyForwardCreate(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, next2 := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	created, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Текущий", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.CopyForward {
		t.Fatal("source copy_forward should stay on")
	}
	nextItems := mustList(t, ctx, sqlDB, userID, next)
	if len(nextItems) != 1 || nextItems[0].CopyForward || nextItems[0].Amount != 10_000 || nextItems[0].Name != "Текущий" {
		t.Fatalf("next month: %+v", nextItems)
	}
	if len(mustList(t, ctx, sqlDB, userID, next2)) != 0 {
		t.Fatal("month after next should stay empty")
	}
	if err := budget.Delete(ctx, sqlDB, userID, nextItems[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("list recreated the deleted next month")
	}
	sum, err := budget.Summary(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Items) != 0 {
		t.Fatal("summary recreated the deleted next month")
	}
	src, err := budget.Get(ctx, sqlDB, userID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !src.CopyForward {
		t.Fatal("source flag should stay on after delete")
	}
}

func TestBudgetCopyForwardTargetErrorRollsBack(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	called := false
	budget.SetCopyForwardHookForTest(func() error {
		called = true
		return errors.New("target failed")
	})
	t.Cleanup(func() { budget.SetCopyForwardHookForTest(nil) })
	_, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Откат", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month, CopyForward: true,
	})
	if err == nil {
		t.Fatal("expected target error")
	}
	if !called {
		t.Fatal("target hook was not called")
	}
	if len(mustList(t, ctx, sqlDB, userID, month)) != 0 {
		t.Fatal("source budget was committed")
	}
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("next month was committed")
	}
}

func TestBudgetCopyForwardInactive(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	if _, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Скрытый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 4_000, IsActive: false, Month: month, CopyForward: true,
	}); err != nil {
		t.Fatal(err)
	}
	items := mustList(t, ctx, sqlDB, userID, next)
	if len(items) != 1 || items[0].IsActive || items[0].CopyForward || items[0].Amount != 4_000 {
		t.Fatalf("inactive copy: %+v", items)
	}
	sum, err := budget.Summary(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Items) != 0 {
		t.Fatalf("summary showed inactive budget: %+v", sum.Items)
	}
}

func TestBudgetCopyForwardPicksActiveOrLatestInactive(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	other, err := category.Create(ctx, sqlDB, userID, "Копия теста", "expense", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("latest inactive", func(t *testing.T) {
		src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
			Name: "Источник", Scope: budget.ScopeCategory, CategoryID: &cat,
			Amount: 1_000, IsActive: false, Month: month,
		})
		if err != nil {
			t.Fatal(err)
		}
		insertRawBudget(t, ctx, sqlDB, "old-inactive", userID, "Старый", budget.ScopeCategory, &cat, nil, 1_500, next, 1, 0, "2026-01-01 00:00:00")
		insertRawBudget(t, ctx, sqlDB, "new-inactive", userID, "Поздний", budget.ScopeCategory, &cat, nil, 1_600, next, 1, 0, "2026-06-01 00:00:00")
		if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
			Name: "Обновлённый", Scope: budget.ScopeCategory, CategoryID: &cat,
			Amount: 8_000, AlertAtPercent: 90, IsActive: false, CopyForward: true,
		}, month); err != nil {
			t.Fatal(err)
		}
		items := mustList(t, ctx, sqlDB, userID, next)
		if len(items) != 2 {
			t.Fatalf("expected 2 inactive rows, got %d", len(items))
		}
		late := mustOne(t, items, "Обновлённый")
		if late.ID != "new-inactive" || late.CopyForward || late.IsActive || late.Amount != 8_000 {
			t.Fatalf("latest inactive: %+v", late)
		}
		old := mustOne(t, items, "Старый")
		if old.Amount != 1_500 || !old.CopyForward {
			t.Fatalf("older inactive changed: %+v", old)
		}
	})

	t.Run("active wins", func(t *testing.T) {
		otherID := other.ID
		src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
			Name: "Активный источник", Scope: budget.ScopeCategory, CategoryID: &otherID,
			Amount: 1_000, IsActive: false, Month: month,
		})
		if err != nil {
			t.Fatal(err)
		}
		insertRawBudget(t, ctx, sqlDB, "pref-active", userID, "Активная цель", budget.ScopeCategory, &otherID, nil, 2_000, next, 1, 1, "2026-01-01 00:00:00")
		insertRawBudget(t, ctx, sqlDB, "pref-inactive", userID, "Новее неактивная", budget.ScopeCategory, &otherID, nil, 2_100, next, 1, 0, "2026-08-01 00:00:00")
		if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
			Name: "С источника", Scope: budget.ScopeCategory, CategoryID: &otherID,
			Amount: 6_000, AlertAtPercent: 90, IsActive: false, CopyForward: true,
		}, month); err != nil {
			t.Fatal(err)
		}
		items := mustList(t, ctx, sqlDB, userID, next)
		active := mustOne(t, items, "С источника")
		if active.ID != "pref-active" || active.IsActive || active.CopyForward || active.Amount != 6_000 {
			t.Fatalf("active target: %+v", active)
		}
		kept := mustOne(t, items, "Новее неактивная")
		if kept.Amount != 2_100 {
			t.Fatalf("newer inactive changed: %+v", kept)
		}
		sum, err := budget.Summary(ctx, sqlDB, userID, next)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range sum.Items {
			if item.ID == "pref-active" {
				t.Fatal("deactivated target is still in summary")
			}
		}
	})
}

func TestBudgetCopyForwardReadDoesNotWrite(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "flagged-source", userID, "Уже включён", budget.ScopeCategory, &cat, nil, 10_000, month, 1, 1, "2026-10-01 00:00:00")
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("list created a next month")
	}
	sum, err := budget.Summary(ctx, sqlDB, userID, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Items) != 0 {
		t.Fatal("summary created a next month")
	}
	if err := budgetnotify.CheckThresholdsForUser(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("threshold check created a next month")
	}
	var n int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM budgets WHERE user_id = ? AND month = ?`, userID, next).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("next month rows = %d", n)
	}
}

func TestBudgetCopyForwardConcurrentSave(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Гонка", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(amount int64) {
			defer wg.Done()
			_, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
				Name: "Гонка", Scope: budget.ScopeCategory, CategoryID: &cat,
				Amount: amount, AlertAtPercent: 90, IsActive: true, CopyForward: true,
			}, month)
			if err != nil && !errors.Is(err, budget.ErrDuplicateActive) {
				errs <- err
			}
		}(int64(10_000 + i*100))
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	cnt, err := sqlcdb.New(sqlDB).CountActiveBudgetsByUserMonth(ctx, sqlcdb.CountActiveBudgetsByUserMonthParams{
		UserID: userID, Month: next,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 active next-month budget, got %d", cnt)
	}
}

func TestBudgetCopyForwardSkipDoesNotWriteNext(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	created, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Импорт", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 10_000, IsActive: true, Month: month, CopyForward: true,
		SkipCopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.CopyForward {
		t.Fatal("flag should stay on")
	}
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("skip wrote the next month")
	}
}

func TestBudgetCopyForwardPastMonthUpdatesOnlyNext(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	prev, err := budget.AddMonths(month, -1)
	if err != nil {
		t.Fatal(err)
	}
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Прошлый", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: prev,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Текущий", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 3_000, IsActive: true, Month: month,
	}); err != nil {
		t.Fatal(err)
	}
	later, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Дальше", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 4_000, IsActive: true, Month: next,
	})
	if err != nil {
		t.Fatal(err)
	}
	laterBefore, err := budget.Get(ctx, sqlDB, userID, later.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
		Name: "Из прошлого", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 9_000, AlertAtPercent: 90, IsActive: true, CopyForward: true,
	}, prev); err != nil {
		t.Fatal(err)
	}
	got := mustOne(t, mustList(t, ctx, sqlDB, userID, month), "Из прошлого")
	if got.Amount != 9_000 || got.CopyForward {
		t.Fatalf("immediate next: %+v", got)
	}
	laterAfter, err := budget.Get(ctx, sqlDB, userID, later.ID)
	if err != nil {
		t.Fatal(err)
	}
	if laterAfter.Amount != laterBefore.Amount || laterAfter.Name != laterBefore.Name || laterAfter.UpdatedAt != laterBefore.UpdatedAt {
		t.Fatalf("month after next changed: %+v", laterAfter)
	}
	srcAfter, err := budget.Get(ctx, sqlDB, userID, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !srcAfter.CopyForward {
		t.Fatal("past month flag should stay on")
	}
}

func TestBudgetCopyForwardUpdateHookRollsBack(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	created, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Было", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	budget.SetCopyForwardHookForTest(func() error {
		return errors.New("target failed")
	})
	t.Cleanup(func() { budget.SetCopyForwardHookForTest(nil) })
	_, err = budget.Update(ctx, sqlDB, userID, created.ID, budget.Input{
		Name: "Стало", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 9_000, AlertAtPercent: 90, IsActive: true, CopyForward: true,
	}, month)
	if err == nil {
		t.Fatal("expected target error")
	}
	got, err := budget.Get(ctx, sqlDB, userID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Было" || got.Amount != 1_000 || got.CopyForward {
		t.Fatalf("source was committed: %+v", got)
	}
	if len(mustList(t, ctx, sqlDB, userID, next)) != 0 {
		t.Fatal("next month was committed")
	}
}

func TestBudgetCopyForwardSubcategoriesStayDistinct(t *testing.T) {
	ctx, sqlDB, userID, _, _ := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	parent, err := category.Create(ctx, sqlDB, userID, "Еда копии", "expense", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	subA, err := category.CreateSubcategory(ctx, sqlDB, userID, parent.ID, "Овощи", "")
	if err != nil {
		t.Fatal(err)
	}
	subB, err := category.CreateSubcategory(ctx, sqlDB, userID, parent.ID, "Фрукты", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Овощи", Scope: budget.ScopeSubcategory, SubcategoryID: &subA.ID,
		Amount: 1_000, IsActive: true, Month: month, CopyForward: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Фрукты", Scope: budget.ScopeSubcategory, SubcategoryID: &subB.ID,
		Amount: 2_000, IsActive: true, Month: month, CopyForward: true,
	}); err != nil {
		t.Fatal(err)
	}
	items := mustList(t, ctx, sqlDB, userID, next)
	if len(items) != 2 {
		t.Fatalf("expected 2 next-month rows, got %d", len(items))
	}
	a := mustOne(t, items, "Овощи")
	b := mustOne(t, items, "Фрукты")
	if a.Amount != 1_000 || b.Amount != 2_000 || a.CopyForward || b.CopyForward {
		t.Fatalf("subcategories overwritten: %+v %+v", a, b)
	}
	if a.SubcategoryID == nil || b.SubcategoryID == nil || *a.SubcategoryID == *b.SubcategoryID {
		t.Fatalf("subcategory ids: %+v %+v", a.SubcategoryID, b.SubcategoryID)
	}
}

func TestBudgetCopyForwardFirstOfMonthSourceRules(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	cat := categoryID
	other, err := category.Create(ctx, sqlDB, userID, "Выбор источника", "expense", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	otherID := other.ID
	insertRawBudget(t, ctx, sqlDB, "off-src", userID, "Выключен", budget.ScopeAllExpense, nil, nil, 1_000, "2026-11", 0, 1, "2026-11-01 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "inactive-src", userID, "Неактивный", budget.ScopeCategory, &cat, nil, 2_000, "2026-11", 1, 0, "2026-11-01 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "active-src", userID, "Активный", budget.ScopeCategory, &otherID, nil, 111, "2026-11", 1, 1, "2026-01-01 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "newer-inactive", userID, "Новее", budget.ScopeCategory, &otherID, nil, 999, "2026-11", 1, 0, "2026-08-01 00:00:00")

	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("created %d", n)
	}
	items := mustList(t, ctx, sqlDB, userID, "2026-12")
	if len(items) != 2 {
		t.Fatalf("december rows: %+v", items)
	}
	for _, item := range items {
		if item.Name == "Выключен" || item.Scope == budget.ScopeAllExpense {
			t.Fatalf("copy_forward off was copied: %+v", item)
		}
	}
	inactive := mustOne(t, items, "Неактивный")
	if inactive.IsActive || inactive.CopyForward || inactive.Amount != 2_000 {
		t.Fatalf("inactive copy: %+v", inactive)
	}
	active := mustOne(t, items, "Активный")
	if !active.IsActive || active.CopyForward || active.Amount != 111 {
		t.Fatalf("same-scope source: %+v", active)
	}
}

func TestBudgetCopyForwardFirstOfMonthContinuesAfterRowError(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "first-row", userID, "Первый", budget.ScopeCategory, &cat, nil, 1_000, "2026-11", 1, 1, "2026-11-02 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "second-row", userID, "Второй", budget.ScopeAllExpense, nil, nil, 4_000, "2026-11", 1, 1, "2026-11-01 00:00:00")
	calls := 0
	budget.SetCopyForwardHookForTest(func() error {
		calls++
		if calls == 1 {
			return errors.New("row failed")
		}
		return nil
	})
	t.Cleanup(func() { budget.SetCopyForwardHookForTest(nil) })

	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected the first row to fail")
	}
	if n != 1 {
		t.Fatalf("created %d", n)
	}
	items := mustList(t, ctx, sqlDB, userID, "2026-12")
	if len(items) != 1 || items[0].Name != "Второй" || items[0].Amount != 4_000 {
		t.Fatalf("later row was skipped: %+v", items)
	}
}

func TestBudgetCopyForwardFutureMonth(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	_, next, next2 := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	created, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Будущий", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 12_000, IsActive: true, Month: next, CopyForward: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.CopyForward {
		t.Fatal("future month flag should stay on")
	}
	if len(mustList(t, ctx, sqlDB, userID, next2)) != 0 {
		t.Fatal("saving a future month created the month after it")
	}
}

func TestBudgetCopyForwardScopeChangeKeepsOld(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	month, next, _ := adjacentMonths(t, ctx, sqlDB, userID)
	cat := categoryID
	src, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Категория", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_000, IsActive: true, Month: month,
	})
	if err != nil {
		t.Fatal(err)
	}
	oldNext, err := budget.Create(ctx, sqlDB, userID, budget.Input{
		Name: "Старый scope", Scope: budget.ScopeCategory, CategoryID: &cat,
		Amount: 1_111, IsActive: true, Month: next,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Update(ctx, sqlDB, userID, src.ID, budget.Input{
		Name: "Общий", Scope: budget.ScopeAllExpense,
		Amount: 2_222, AlertAtPercent: 90, IsActive: true, CopyForward: true,
	}, month); err != nil {
		t.Fatal(err)
	}
	items := mustList(t, ctx, sqlDB, userID, next)
	kept := mustOne(t, items, "Старый scope")
	if kept.ID != oldNext.ID || kept.Amount != 1_111 || kept.Scope != budget.ScopeCategory {
		t.Fatalf("old scope removed: %+v", kept)
	}
	created := mustOne(t, items, "Общий")
	if created.Scope != budget.ScopeAllExpense || created.Amount != 2_222 || created.CopyForward {
		t.Fatalf("new scope: %+v", created)
	}
}

func TestBudgetCopyForwardFirstOfMonth(t *testing.T) {
	ctx, sqlDB, userID, accountID, categoryID := seedBudgetEnv(t)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "nov-src", userID, "Ноябрь", budget.ScopeCategory, &cat, &accountID, 15_000, "2026-11", 1, 1, "2026-11-01 00:00:00")
	var notifies int
	realtime.SetUserDataChangedHandler(func(id string) {
		if id == userID {
			notifies++
		}
	})
	t.Cleanup(func() { realtime.SetUserDataChangedHandler(nil) })

	local := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, local)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("created %d", n)
	}
	dec := mustList(t, ctx, sqlDB, userID, "2026-12")
	if len(dec) != 1 || dec[0].Name != "Ноябрь" || dec[0].Amount != 15_000 || dec[0].CopyForward || dec[0].AlertAtPercent != 90 {
		t.Fatalf("december: %+v", dec)
	}
	if dec[0].AccountID == nil || *dec[0].AccountID != accountID {
		t.Fatalf("account: %+v", dec[0].AccountID)
	}
	if len(mustList(t, ctx, sqlDB, userID, "2027-01")) != 0 {
		t.Fatal("january was created")
	}
	n, err = budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, local)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("repeat created %d", n)
	}
	if len(mustList(t, ctx, sqlDB, userID, "2026-12")) != 1 {
		t.Fatal("repeat added a row")
	}
	if notifies != 1 {
		t.Fatalf("cache invalidations = %d", notifies)
	}
}

func TestBudgetCopyForwardFirstOfMonthLeavesExisting(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "nov-src-2", userID, "Ноябрь", budget.ScopeCategory, &cat, nil, 15_000, "2026-11", 1, 1, "2026-11-01 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "dec-exists", userID, "Свой декабрь", budget.ScopeCategory, &cat, nil, 99_000, "2026-12", 1, 1, "2026-10-02 00:00:00")
	var notifies int
	realtime.SetUserDataChangedHandler(func(string) { notifies++ })
	t.Cleanup(func() { realtime.SetUserDataChangedHandler(nil) })

	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("created %d", n)
	}
	got, err := budget.Get(ctx, sqlDB, userID, "dec-exists")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Свой декабрь" || got.Amount != 99_000 || !got.CopyForward || got.UpdatedAt != "2026-10-02 00:00:00" {
		t.Fatalf("december overwritten: %+v", got)
	}
	if len(mustList(t, ctx, sqlDB, userID, "2027-01")) != 0 {
		t.Fatal("january was created")
	}
	if notifies != 0 {
		t.Fatalf("cache invalidated without a write: %d", notifies)
	}
}

func TestBudgetCopyForwardOtherDaysDoNotCreate(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "nov-src-3", userID, "Ноябрь", budget.ScopeCategory, &cat, nil, 15_000, "2026-11", 1, 1, "2026-11-01 00:00:00")
	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("created %d", n)
	}
	if len(mustList(t, ctx, sqlDB, userID, "2026-12")) != 0 {
		t.Fatal("background created december on the 2nd")
	}
}

func adjacentMonths(t *testing.T, ctx context.Context, db *sql.DB, userID string) (month, next, next2 string) {
	t.Helper()
	var err error
	month, err = budget.CurrentMonthQuery(ctx, db, userID)
	if err != nil {
		t.Fatal(err)
	}
	next, err = budget.AddMonths(month, 1)
	if err != nil {
		t.Fatal(err)
	}
	next2, err = budget.AddMonths(month, 2)
	if err != nil {
		t.Fatal(err)
	}
	return month, next, next2
}

func mustList(t *testing.T, ctx context.Context, db *sql.DB, userID, month string) []budget.Budget {
	t.Helper()
	items, err := budget.List(ctx, db, userID, month)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func mustOne(t *testing.T, items []budget.Budget, name string) budget.Budget {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("budget %q not found in %+v", name, items)
	return budget.Budget{}
}

func insertRawBudget(t *testing.T, ctx context.Context, db *sql.DB, id, userID, name, scope string, categoryID, accountID *string, amount int64, month string, copyForward, active int, updatedAt string) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		INSERT INTO budgets (
			id, user_id, name, scope, category_id, amount, period, account_id,
			month, copy_forward, rollover, alert_at_percent, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'month', ?, ?, ?, 0, 90, ?, ?, ?)`,
		id, userID, name, scope, categoryID, amount, accountID, month, copyForward, active, updatedAt, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBudgetCopyKeepsCurrency(t *testing.T) {
	ctx, sqlDB, userID, _, categoryID := seedBudgetEnv(t)
	cat := categoryID
	insertRawBudget(t, ctx, sqlDB, "nov-rub", userID, "Рубли", budget.ScopeCategory, &cat, nil, 15_000, "2026-11", 1, 1, "2026-11-01 00:00:00")
	insertRawBudget(t, ctx, sqlDB, "nov-usd", userID, "Доллары", budget.ScopeCategory, &cat, nil, 2_000, "2026-11", 1, 0, "2026-11-01 00:00:00")
	if _, err := sqlDB.ExecContext(ctx, `UPDATE budgets SET currency = 'USD', is_active = 1 WHERE id = 'nov-usd'`); err != nil {
		t.Fatal(err)
	}

	n, err := budget.CopyMissingNextMonthOnDate(ctx, sqlDB, userID, time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("created %d, want 2", n)
	}
	dec := mustList(t, ctx, sqlDB, userID, "2026-12")
	if got := mustOne(t, dec, "Доллары").Currency; got != "USD" {
		t.Fatalf("december USD copy currency %q", got)
	}
	if got := mustOne(t, dec, "Рубли").Currency; got != "RUB" {
		t.Fatalf("december RUB copy currency %q", got)
	}

	copied, err := budget.CopyFromPreviousMonth(ctx, sqlDB, userID, "2027-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(copied) != 2 {
		t.Fatalf("copied %d, want 2", len(copied))
	}
	if got := mustOne(t, copied, "Доллары").Currency; got != "USD" {
		t.Fatalf("january USD copy currency %q", got)
	}
}
