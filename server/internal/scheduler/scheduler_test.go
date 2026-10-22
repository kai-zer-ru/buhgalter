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
