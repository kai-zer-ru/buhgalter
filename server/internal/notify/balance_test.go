package notify

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestFormatWithBalanceShortfallAppendsSuffix(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-bal"
	now := mustTime("2026-06-01 12:00:00")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'baluser', 'hash', 'Bal', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-bal', ?, 'Счёт', 'cash', 880000, 880000, 'active', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(manager.DB())
	lookup := newBalanceLookup(q, userID)
	mainText := "Платёж по кредиту «Кредит»: 10 000.00 ₽. Дата: сегодня"
	text, err := formatWithBalanceShortfall(ctx, lookup, true, "ru", "RUB", map[string]*string{}, mainText, "acc-bal", 1000000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, mainText) {
		t.Fatalf("expected main text preserved: %q", text)
	}
	if !strings.Contains(text, "На балансе не хватает 1 200.00 ₽!") {
		t.Fatalf("expected shortfall suffix, got: %q", text)
	}
}

func TestFormatWithBalanceShortfallDisabled(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-bal-off"
	now := mustTime("2026-06-01 12:00:00")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'baloff', 'hash', 'Bal', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-off', ?, 'Счёт', 'cash', 0, 0, 'active', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(manager.DB())
	lookup := newBalanceLookup(q, userID)
	mainText := "Платёж: 10 000.00 ₽"
	text, err := formatWithBalanceShortfall(ctx, lookup, false, "ru", "RUB", map[string]*string{}, mainText, "acc-off", 1000000)
	if err != nil {
		t.Fatal(err)
	}
	if text != mainText {
		t.Fatalf("expected unchanged text, got %q", text)
	}
}

func TestFormatWithBalanceShortfallSufficientFunds(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-bal-ok"
	now := mustTime("2026-06-01 12:00:00")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'balok', 'hash', 'Bal', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-ok', ?, 'Счёт', 'cash', 2000000, 2000000, 'active', ?, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(manager.DB())
	lookup := newBalanceLookup(q, userID)
	mainText := "Платёж: 10 000.00 ₽"
	text, err := formatWithBalanceShortfall(ctx, lookup, true, "ru", "RUB", map[string]*string{}, mainText, "acc-ok", 1000000)
	if err != nil {
		t.Fatal(err)
	}
	if text != mainText {
		t.Fatalf("expected unchanged text when balance is enough, got %q", text)
	}
}
