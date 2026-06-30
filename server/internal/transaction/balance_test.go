package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr.DB()
}

func seedUserAccount(t *testing.T, database *sql.DB) (userID, accountID string) {
	t.Helper()
	ctx := context.Background()
	userID = "user-1"
	accountID = "acc-1"
	_, err := database.ExecContext(ctx, `INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'test', 'hash', 'UTC')`, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.ExecContext(ctx, `INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at) VALUES (?, ?, 'Test', 'cash', 100000, 100000, 'active', datetime('now'), datetime('now'))`, accountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return userID, accountID
}

var testTxSeq atomic.Uint64

func insertTx(t *testing.T, database *sql.DB, userID, accountID, txType, kind, date string, amount int64) {
	t.Helper()
	id := fmt.Sprintf("tx-%d", testTxSeq.Add(1))
	_, err := database.ExecContext(context.Background(), `
		INSERT INTO transactions (id, user_id, account_id, type, kind, amount, transaction_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		id, userID, accountID, txType, kind, amount, date)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBalanceManualOnly(t *testing.T) {
	database := testDB(t)
	userID, accountID := seedUserAccount(t, database)
	now := timeutil.FormatUTC(timeutil.NowUTC())
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-24 * time.Hour))

	insertTx(t, database, userID, accountID, "income", "manual", past, 5000)
	insertTx(t, database, userID, accountID, "income", "manual", past, 5000)
	insertTx(t, database, userID, accountID, "expense", "manual", past, 2000)
	if err := accountbalance.Refresh(context.Background(), database, userID, accountID); err != nil {
		t.Fatal(err)
	}

	bal, err := Balance(context.Background(), database, userID, accountID, 100000)
	if err != nil {
		t.Fatal(err)
	}
	// 100000 + 5000 + 5000 - 2000 = 108000
	if bal != 108000 {
		t.Fatalf("expected 108000, got %d", bal)
	}

	insertTx(t, database, userID, accountID, "income", "future", now, 99999)
	bal2, err := Balance(context.Background(), database, userID, accountID, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if bal2 != 108000 {
		t.Fatalf("future should not affect balance, got %d", bal2)
	}
}

// futureInCurrentMonthUTC returns a datetime after now but still within the current UTC month.
func futureInCurrentMonthUTC(t *testing.T) time.Time {
	t.Helper()
	now := timeutil.NowUTC()
	_, monthEnd, err := timeutil.MonthBoundsUTC("UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	monthEndTime, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(48 * time.Hour)
	if !future.Before(monthEndTime) {
		future = monthEndTime.Add(-time.Hour)
	}
	if !future.After(now) {
		t.Skip("no room for future transaction in current month")
	}
	return future
}

func TestForecastWithFutureInMonth(t *testing.T) {
	database := testDB(t)
	userID, accountID := seedUserAccount(t, database)
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-24 * time.Hour))
	future := timeutil.FormatUTC(futureInCurrentMonthUTC(t))

	insertTx(t, database, userID, accountID, "expense", "manual", past, 1000)
	insertTx(t, database, userID, accountID, "expense", "future", future, 3000)

	if err := accountbalance.Refresh(context.Background(), database, userID, accountID); err != nil {
		t.Fatal(err)
	}
	bal, err := Balance(context.Background(), database, userID, accountID, 100000)
	if err != nil {
		t.Fatal(err)
	}
	forecast, hasFuture, err := ForecastBalance(context.Background(), database, userID, accountID, bal)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFuture {
		t.Fatal("expected has_future_this_month")
	}
	// 99000 - 3000 future expense = 96000
	if forecast != 96000 {
		t.Fatalf("expected forecast 96000, got %d", forecast)
	}
}

func TestResolveKindFuture(t *testing.T) {
	database := testDB(t)
	userID, _ := seedUserAccount(t, database)
	future := timeutil.NowUTC().Add(72 * time.Hour)
	kind, err := ResolveKindForDate(context.Background(), database, userID, future)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "future" {
		t.Fatalf("expected future, got %s", kind)
	}
	past := timeutil.NowUTC().Add(-time.Hour)
	kind2, err := ResolveKindForDate(context.Background(), database, userID, past)
	if err != nil {
		t.Fatal(err)
	}
	if kind2 != "manual" {
		t.Fatalf("expected manual, got %s", kind2)
	}
}
