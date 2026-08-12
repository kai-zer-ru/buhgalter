package accountbalance

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func scheduledTestDB(t *testing.T) *sql.DB {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr.DB()
}

func seedUserAcc(t *testing.T, database *sql.DB) (userID, accountID string) {
	t.Helper()
	ctx := context.Background()
	userID = "user-1"
	accountID = "acc-1"
	_, err := database.ExecContext(ctx, `INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'test', 'hash', 'UTC')`, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Test', 'cash', 100000, 100000, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return userID, accountID
}

func seedCategory(t *testing.T, database *sql.DB, userID, catType string) string {
	t.Helper()
	id := "cat-" + catType
	_, err := database.ExecContext(context.Background(), `
		INSERT INTO categories (id, user_id, name, type, icon, sort_order, is_system, created_at)
		VALUES (?, ?, ?, ?, 'other', 0, 0, datetime('now'))`,
		id, userID, "Cat "+catType, catType)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// atLocal builds a UTC instant on the given day at timeLocal (HH:MM) so it matches schedule.NextRunAt.
func atLocal(day time.Time, timeLocal string) time.Time {
	hour, minute := 10, 0
	if t, err := time.Parse("15:04", timeLocal); err == nil {
		hour, minute = t.Hour(), t.Minute()
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.UTC)
}

func insertSubscription(t *testing.T, database *sql.DB, userID, accountID string, amount int64, period string, weekday, dayOfMonth *int64, startDate, timeLocal, nextRunAt string, active int) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), `
		INSERT INTO subscriptions (
			id, user_id, name, amount, account_id, period, weekday, day_of_month,
			start_date, time_local, next_run_at, active, created_at, updated_at
		) VALUES (?, ?, 'Sub', ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		"sub-1", userID, amount, accountID, period, weekday, dayOfMonth, startDate, timeLocal, nextRunAt, active)
	if err != nil {
		t.Fatal(err)
	}
}

func insertRecurring(t *testing.T, database *sql.DB, id, userID, accountID, catID, typ string, amount int64, period string, weekday, dayOfMonth *int64, startDate, timeLocal, nextRunAt string, active int) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), `
		INSERT INTO recurring_operations (
			id, user_id, type, amount, description, account_id, category_id,
			period, weekday, day_of_month, start_date, time_local, next_run_at, active,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		id, userID, typ, amount, accountID, catID, period, weekday, dayOfMonth, startDate, timeLocal, nextRunAt, active)
	if err != nil {
		t.Fatal(err)
	}
}

func futureInMonth(t *testing.T, now time.Time) time.Time {
	t.Helper()
	_, monthEnd, err := timeutil.MonthBoundsUTC("UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	monthEndT, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(48 * time.Hour)
	if !future.Before(monthEndT) {
		future = monthEndT.Add(-time.Hour)
	}
	if !future.After(now) {
		t.Skip("no room for future run in current month")
	}
	return future
}

func nextMonthRun(t *testing.T, now time.Time) time.Time {
	t.Helper()
	_, monthEnd, err := timeutil.MonthBoundsUTC("UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	monthEndT, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		t.Fatal(err)
	}
	return monthEndT.Add(48 * time.Hour)
}

func TestScheduledEffects_MonthlySubscription(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	next := atLocal(futureInMonth(t, now), "10:00")
	day := int64(next.Day())
	insertSubscription(t, database, userID, accountID, 5000, "month", nil, &day,
		timeutil.FormatUTC(now.AddDate(0, -1, 0)), "10:00", timeutil.FormatUTC(next), 1)

	deltas, has, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if deltas[accountID] != -5000 {
		t.Fatalf("expected -5000, got %d", deltas[accountID])
	}
	if _, ok := has[accountID]; !ok {
		t.Fatal("expected account in has set")
	}
}

func TestScheduledEffects_RecurringIncomeAndExpense(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	expCat := seedCategory(t, database, userID, "expense")
	incCat := seedCategory(t, database, userID, "income")
	now := timeutil.NowUTC()
	next := atLocal(futureInMonth(t, now), "09:00")
	day := int64(next.Day())
	start := timeutil.FormatUTC(now.AddDate(0, -2, 0))
	nextStr := timeutil.FormatUTC(next)
	insertRecurring(t, database, "ro-exp", userID, accountID, expCat, "expense", 3000, "month", nil, &day, start, "09:00", nextStr, 1)
	insertRecurring(t, database, "ro-inc", userID, accountID, incCat, "income", 10000, "month", nil, &day, start, "09:00", nextStr, 1)

	deltas, _, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	// -3000 + 10000
	if deltas[accountID] != 7000 {
		t.Fatalf("expected 7000, got %d", deltas[accountID])
	}
}

func TestScheduledEffects_WeeklyMultiple(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	_, monthEnd, err := timeutil.MonthBoundsUTC("UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	monthEndT, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		t.Fatal(err)
	}
	// First run tomorrow; need at least two weekly runs before month end.
	first := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	if first.After(monthEndT) || first.AddDate(0, 0, 7).After(monthEndT) {
		t.Skip("not enough days left in month for two weekly runs")
	}
	weekday := int64(first.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := timeutil.FormatUTC(first.AddDate(0, 0, -14))
	insertSubscription(t, database, userID, accountID, 1000, "week", &weekday, nil,
		start, "12:00", timeutil.FormatUTC(first), 1)

	deltas, _, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for tRun := first; !tRun.After(monthEndT); tRun = tRun.AddDate(0, 0, 7) {
		count++
	}
	want := -int64(count) * 1000
	if deltas[accountID] != want {
		t.Fatalf("expected %d (%d runs), got %d", want, count, deltas[accountID])
	}
}

func TestScheduledEffects_NextMonthIgnored(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	next := nextMonthRun(t, now)
	day := int64(next.Day())
	insertSubscription(t, database, userID, accountID, 5000, "month", nil, &day,
		timeutil.FormatUTC(now), "10:00", timeutil.FormatUTC(next), 1)

	deltas, has, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if deltas[accountID] != 0 {
		t.Fatalf("expected 0, got %d", deltas[accountID])
	}
	if _, ok := has[accountID]; ok {
		t.Fatal("expected no has entry")
	}
}

func TestScheduledEffects_InactiveIgnored(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	next := futureInMonth(t, now)
	day := int64(next.Day())
	insertSubscription(t, database, userID, accountID, 5000, "month", nil, &day,
		timeutil.FormatUTC(now.AddDate(0, -1, 0)), "10:00", timeutil.FormatUTC(next), 0)

	deltas, _, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if deltas[accountID] != 0 {
		t.Fatalf("expected 0, got %d", deltas[accountID])
	}
}

func TestScheduledEffects_OverdueIncluded(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	overdue := atLocal(now.Add(-48*time.Hour), "10:00")
	day := int64(overdue.Day())
	insertSubscription(t, database, userID, accountID, 2500, "month", nil, &day,
		timeutil.FormatUTC(overdue.AddDate(0, -1, 0)), "10:00", timeutil.FormatUTC(overdue), 1)

	deltas, _, err := ScheduledEffectsByUser(context.Background(), database, userID, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if deltas[accountID] != -2500 {
		t.Fatalf("expected -2500, got %d", deltas[accountID])
	}
}

func TestForecastsByUser_SubscriptionPlusFutureTx(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	now := timeutil.NowUTC()
	next := atLocal(futureInMonth(t, now), "10:00")
	day := int64(next.Day())
	insertSubscription(t, database, userID, accountID, 4000, "month", nil, &day,
		timeutil.FormatUTC(now.AddDate(0, -1, 0)), "10:00", timeutil.FormatUTC(next), 1)

	futureTx := futureInMonth(t, now)
	_, err := database.ExecContext(context.Background(), `
		INSERT INTO transactions (id, user_id, account_id, type, kind, amount, transaction_date, created_at, updated_at)
		VALUES ('tx-f', ?, ?, 'expense', 'future', 2000, ?, datetime('now'), datetime('now'))`,
		userID, accountID, timeutil.FormatUTC(futureTx))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ForecastsByUser(context.Background(), database, userID, "UTC", map[string]int64{accountID: 100000})
	if err != nil {
		t.Fatal(err)
	}
	fc := out[accountID]
	if !fc.HasFutureThisMonth {
		t.Fatal("expected has_future_this_month")
	}
	if fc.Balance != 94000 {
		t.Fatalf("expected 94000 (100000-2000-4000), got %d", fc.Balance)
	}
}

func TestForecastsByUser_SubscriptionAndRecurring(t *testing.T) {
	database := scheduledTestDB(t)
	userID, accountID := seedUserAcc(t, database)
	expCat := seedCategory(t, database, userID, "expense")
	now := timeutil.NowUTC()
	nextDay := futureInMonth(t, now)
	day := int64(nextDay.Day())
	start := timeutil.FormatUTC(now.AddDate(0, -1, 0))
	subNext := timeutil.FormatUTC(atLocal(nextDay, "10:00"))
	recNext := timeutil.FormatUTC(atLocal(nextDay, "11:00"))
	insertSubscription(t, database, userID, accountID, 1500, "month", nil, &day, start, "10:00", subNext, 1)
	insertRecurring(t, database, "ro-1", userID, accountID, expCat, "expense", 2500, "month", nil, &day, start, "11:00", recNext, 1)

	out, err := ForecastsByUser(context.Background(), database, userID, "UTC", map[string]int64{accountID: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if out[accountID].Balance != 96000 {
		t.Fatalf("expected 96000, got %d", out[accountID].Balance)
	}
}
