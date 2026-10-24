package scheduler

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/features"
)

func testScheduler(t *testing.T) (*Scheduler, *sql.DB, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	sqlDB := mgr.DB()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	name := "sched"
	if err := sqlcdb.New(sqlDB).InsertUser(context.Background(), sqlcdb.InsertUserParams{
		ID:           "user-1",
		Login:        "sched",
		PasswordHash: "x",
		DisplayName:  &name,
		IsAdmin:      0,
		Status:       "active",
	}); err != nil {
		t.Fatal(err)
	}
	s := New(
		&CreditRunner{DB: sqlDB, Logger: logger},
		&RecurringRunner{DB: sqlDB, Logger: logger},
		&SubscriptionRunner{DB: sqlDB, Logger: logger},
		nil,
		logger,
	)
	return s, sqlDB, "user-1"
}

func slotKey(now time.Time) string {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return now.In(loc).Format("2006-01-02 15:04")
}

func TestLastRunStampedAfterSuccessfulApplyDue(t *testing.T) {
	s, _, uid := testScheduler(t)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	want := slotKey(now)

	s.runCreditPayments(now)
	s.runRecurring(now)
	s.runSubscriptions(now)

	if s.creditLastRun[uid] != want {
		t.Fatalf("credit lastRun=%q want %q", s.creditLastRun[uid], want)
	}
	if s.recurringLastRun[uid] != want {
		t.Fatalf("recurring lastRun=%q want %q", s.recurringLastRun[uid], want)
	}
	if s.subscriptionLastRun[uid] != want {
		t.Fatalf("subscription lastRun=%q want %q", s.subscriptionLastRun[uid], want)
	}

	s.runCreditPayments(now)
	s.runRecurring(now)
	s.runSubscriptions(now)

	if s.creditLastRun[uid] != want || s.recurringLastRun[uid] != want || s.subscriptionLastRun[uid] != want {
		t.Fatalf("second pass lastRun credit=%q recurring=%q sub=%q want %q",
			s.creditLastRun[uid], s.recurringLastRun[uid], s.subscriptionLastRun[uid], want)
	}
}

func TestLastRunNotStampedWhenApplyDueFails(t *testing.T) {
	s, sqlDB, uid := testScheduler(t)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	if _, err := sqlDB.Exec(`ALTER TABLE credit_payments RENAME TO credit_payments_off`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE recurring_operations RENAME TO recurring_operations_off`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE subscriptions RENAME TO subscriptions_off`); err != nil {
		t.Fatal(err)
	}

	s.runCreditPayments(now)
	s.runRecurring(now)
	s.runSubscriptions(now)

	if _, ok := s.creditLastRun[uid]; ok {
		t.Fatalf("credit lastRun set on error: %q", s.creditLastRun[uid])
	}
	if _, ok := s.recurringLastRun[uid]; ok {
		t.Fatalf("recurring lastRun set on error: %q", s.recurringLastRun[uid])
	}
	if _, ok := s.subscriptionLastRun[uid]; ok {
		t.Fatalf("subscription lastRun set on error: %q", s.subscriptionLastRun[uid])
	}

	if _, err := sqlDB.Exec(`ALTER TABLE credit_payments_off RENAME TO credit_payments`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE recurring_operations_off RENAME TO recurring_operations`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE subscriptions_off RENAME TO subscriptions`); err != nil {
		t.Fatal(err)
	}

	s.runCreditPayments(now)
	s.runRecurring(now)
	s.runSubscriptions(now)

	want := slotKey(now)
	if s.creditLastRun[uid] != want || s.recurringLastRun[uid] != want || s.subscriptionLastRun[uid] != want {
		t.Fatalf("retry after repair lastRun credit=%q recurring=%q sub=%q want %q",
			s.creditLastRun[uid], s.recurringLastRun[uid], s.subscriptionLastRun[uid], want)
	}
}

func TestBudgetCopyForwardRunsOnceOnTheFirst(t *testing.T) {
	s, sqlDB, uid := testScheduler(t)
	s.Budget = &BudgetRunner{DB: sqlDB}
	// 2026-10-31 11:00 UTC is 2026-11-01 01:00 in Pacific/Kiritimati (UTC+14)
	// and still 31 October in UTC and Europe/Moscow.
	setSchedulerTimezone(t, sqlDB, uid, "Pacific/Kiritimati")
	if _, err := sqlDB.Exec(`
		INSERT INTO budgets (
			id, user_id, name, scope, amount, period, month, copy_forward,
			alert_at_percent, is_active, created_at, updated_at
		) VALUES ('b-nov', ?, 'Ноябрь', 'all_expense', 10000, 'month', '2026-11', 1, 90, 1, '2026-11-01 00:00:00', '2026-11-01 00:00:00')`, uid); err != nil {
		t.Fatal(err)
	}
	countMonth := func(month string) int {
		t.Helper()
		var n int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM budgets WHERE user_id = ? AND month = ?`, uid, month).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	localFirst := time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)
	s.runBudgetCopyForward(localFirst)
	if countMonth("2026-12") != 1 {
		t.Fatalf("december rows = %d", countMonth("2026-12"))
	}
	if _, err := sqlDB.Exec(`DELETE FROM budgets WHERE user_id = ? AND month = '2026-12'`, uid); err != nil {
		t.Fatal(err)
	}
	s.runBudgetCopyForward(localFirst)
	if countMonth("2026-12") != 0 {
		t.Fatal("same local day created december again")
	}
	// 2026-11-01 11:00 UTC is 2 November in Kiritimati and 1 November in Moscow.
	s.runBudgetCopyForward(time.Date(2026, 11, 1, 11, 0, 0, 0, time.UTC))
	if countMonth("2026-12") != 0 {
		t.Fatal("2 November created december")
	}
}

func TestBudgetCopyForwardRetriesSameDayAfterError(t *testing.T) {
	s, sqlDB, uid := testScheduler(t)
	s.Budget = &BudgetRunner{DB: sqlDB}
	setSchedulerTimezone(t, sqlDB, uid, "Pacific/Kiritimati")
	if _, err := sqlDB.Exec(`
		INSERT INTO budgets (
			id, user_id, name, scope, amount, period, month, copy_forward,
			alert_at_percent, is_active, created_at, updated_at
		) VALUES ('b-nov', ?, 'Ноябрь', 'all_expense', 10000, 'month', '2026-11', 1, 90, 1, '2026-11-01 00:00:00', '2026-11-01 00:00:00')`, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE budgets RENAME TO budgets_off`); err != nil {
		t.Fatal(err)
	}
	restored := false
	restore := func() {
		if restored {
			return
		}
		if _, err := sqlDB.Exec(`ALTER TABLE budgets_off RENAME TO budgets`); err != nil {
			t.Fatal(err)
		}
		restored = true
	}
	t.Cleanup(restore)
	localFirst := time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)
	s.runBudgetCopyForward(localFirst)
	restore()
	s.runBudgetCopyForward(localFirst)
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM budgets WHERE user_id = ? AND month = '2026-12'`, uid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("retry after error created %d rows", n)
	}
}

func TestBudgetCopyForwardSkipsWhenFeatureOff(t *testing.T) {
	s, sqlDB, uid := testScheduler(t)
	s.Budget = &BudgetRunner{DB: sqlDB}
	setSchedulerTimezone(t, sqlDB, uid, "Pacific/Kiritimati")
	if _, err := sqlDB.Exec(`
		INSERT INTO budgets (
			id, user_id, name, scope, amount, period, month, copy_forward,
			alert_at_percent, is_active, created_at, updated_at
		) VALUES ('b-nov', ?, 'Ноябрь', 'all_expense', 10000, 'month', '2026-11', 1, 90, 1, '2026-11-01 00:00:00', '2026-11-01 00:00:00')`, uid); err != nil {
		t.Fatal(err)
	}
	if err := features.SetOne(context.Background(), sqlDB, features.Budget, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = features.SetOne(context.Background(), sqlDB, features.Budget, true)
		features.Invalidate()
	})
	s.runBudgetCopyForward(time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC))
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM budgets WHERE user_id = ? AND month = '2026-12'`, uid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("feature off created %d rows", n)
	}
}

func setSchedulerTimezone(t *testing.T, sqlDB *sql.DB, userID, tz string) {
	t.Helper()
	if _, err := sqlDB.Exec(`UPDATE users SET timezone = ? WHERE id = ?`, tz, userID); err != nil {
		t.Fatal(err)
	}
}
