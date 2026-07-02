package notify

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestWorkerCreditPaymentReminder(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-credit"
	now := time.Now().UTC()
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'creduser', 'hash', 'Cred', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES ('acc-n', ?, 'Счёт', 'cash', 0, 'active', ?, ?)`, userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credits (id, user_id, name, principal_amount, issue_date, term_months, interest_rate, payment_interval,
			paid_amount, monthly_payment, debit_account_id, added_retroactively, recorded_at, status, created_at, updated_at)
		VALUES ('cr-1', ?, 'Кредит', 100000, datetime('now', '-1 month'), 12, 0, 'month', 0, 10000, 'acc-n', 0, datetime('now'), 'active', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	payDate := now.Format("2006-01-02 15:04:05")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credit_payments (id, credit_id, amount, payment_date, kind, is_applied, exclude_from_stats, created_at)
		VALUES ('cp-1', 'cr-1', 10000, ?, 'scheduled', 0, 0, datetime('now'))`, payDate)
	if err != nil {
		t.Fatal(err)
	}

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	secret := "12345678901234567890123456789012"
	_, err = manager.DB().ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned, debt_days_before, credit_days_before,
			notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 1, 0, 1, 0, '00:00', ?)`,
		userID, token, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(manager.DB(), slog.Default())
	settings := sqlNotificationSettings(t, manager, ctx, userID)
	if err := worker.runForUser(ctx, userID, now, now.In(time.FixedZone("MSK", 3*3600)), settings); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected telegram notifier to be called")
	}
}

func TestWorkerCreditPaymentShortfall(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-credit-short"
	now := time.Now().UTC()
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'credshort', 'hash', 'Cred', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-short', ?, 'Счёт', 'cash', 880000, 880000, 'active', ?, ?)`, userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credits (id, user_id, name, principal_amount, issue_date, term_months, interest_rate, payment_interval,
			paid_amount, monthly_payment, debit_account_id, added_retroactively, recorded_at, status, created_at, updated_at)
		VALUES ('cr-short', ?, 'Кредит', 100000, datetime('now', '-1 month'), 12, 0, 'month', 0, 10000, 'acc-short', 0, datetime('now'), 'active', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	payDate := now.Format("2006-01-02 15:04:05")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credit_payments (id, credit_id, amount, payment_date, kind, is_applied, exclude_from_stats, created_at)
		VALUES ('cp-short', 'cr-short', 1000000, ?, 'scheduled', 0, 0, datetime('now'))`, payDate)
	if err != nil {
		t.Fatal(err)
	}

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	secret := "12345678901234567890123456789012"
	_, err = manager.DB().ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned, trigger_negative_balance,
			debt_days_before, credit_days_before, notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 1, 0, 1, 1, 0, '00:00', ?)`,
		userID, token, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(manager.DB(), slog.Default())
	settings := sqlNotificationSettings(t, manager, ctx, userID)
	if err := worker.runForUser(ctx, userID, now, now.In(time.FixedZone("MSK", 3*3600)), settings); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected telegram notifier to be called")
	}
	var message string
	err = manager.DB().QueryRowContext(ctx, `
		SELECT message FROM notification_log WHERE user_id = ? AND trigger_type = ? ORDER BY created_at DESC LIMIT 1`,
		userID, TriggerCreditPayment).Scan(&message)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "На балансе не хватает 1 200.00 ₽!") {
		t.Fatalf("expected shortfall suffix in %q", message)
	}
}

func TestWorkerCreditPaymentNoShortfallWhenDisabled(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u-credit-no-short"
	now := time.Now().UTC()
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'crednoshort', 'hash', 'Cred', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-noshort', ?, 'Счёт', 'cash', 0, 0, 'active', ?, ?)`, userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credits (id, user_id, name, principal_amount, issue_date, term_months, interest_rate, payment_interval,
			paid_amount, monthly_payment, debit_account_id, added_retroactively, recorded_at, status, created_at, updated_at)
		VALUES ('cr-noshort', ?, 'Кредит', 100000, datetime('now', '-1 month'), 12, 0, 'month', 0, 10000, 'acc-noshort', 0, datetime('now'), 'active', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	payDate := now.Format("2006-01-02 15:04:05")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO credit_payments (id, credit_id, amount, payment_date, kind, is_applied, exclude_from_stats, created_at)
		VALUES ('cp-noshort', 'cr-noshort', 1000000, ?, 'scheduled', 0, 0, datetime('now'))`, payDate)
	if err != nil {
		t.Fatal(err)
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	secret := "12345678901234567890123456789012"
	_, err = manager.DB().ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned, trigger_negative_balance,
			debt_days_before, credit_days_before, notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 1, 0, 0, 1, 0, '00:00', ?)`,
		userID, token, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(manager.DB(), slog.Default())
	settings := sqlNotificationSettings(t, manager, ctx, userID)
	if err := worker.runForUser(ctx, userID, now, now.In(time.FixedZone("MSK", 3*3600)), settings); err != nil {
		t.Fatal(err)
	}
	var message string
	err = manager.DB().QueryRowContext(ctx, `
		SELECT message FROM notification_log WHERE user_id = ? AND trigger_type = ? ORDER BY created_at DESC LIMIT 1`,
		userID, TriggerCreditPayment).Scan(&message)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(message, "не хватает") {
		t.Fatalf("did not expect shortfall suffix in %q", message)
	}
}

func TestWorkerPlannedOperations(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	userID := "u-plan"
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'planuser', 'hash', 'Plan', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES ('acc-p', ?, 'Кошелёк', 'cash', 0, 'active', ?, ?)`, userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	catID := "cat-expense"
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO categories (id, user_id, name, type, icon, sort_order, is_primary, is_system, created_at)
		VALUES (?, ?, 'Еда', 'expense', 'default', 0, 1, 0, ?)`,
		catID, userID, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	txDate := now.Format("2006-01-02 15:04:05")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, account_id, type, kind, amount, description, category_id,
			transaction_date, affects_balance, created_at, updated_at
		) VALUES ('tx-future', ?, 'acc-p', 'expense', 'future', 5000, 'Подписка', ?, ?, 1, ?, ?)`,
		userID, catID, txDate, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	secret := "12345678901234567890123456789012"
	_, err = manager.DB().ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned, debt_days_before, credit_days_before,
			notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 0, 1, 1, 0, '00:00', ?)`,
		userID, token, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(manager.DB(), slog.Default())
	settings := sqlNotificationSettings(t, manager, ctx, userID)
	msk := time.FixedZone("MSK", 3*3600)
	if err := worker.runForUser(ctx, userID, now, now.In(msk), settings); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected planned operation notification")
	}
}

func TestWorkerRunAllUsers(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	worker := NewWorker(manager.DB(), slog.Default())
	worker.run(time.Now().UTC())
}

func TestWorkerSendsOverdueDebtNotification(t *testing.T) {
	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	userID := "u1"
	now := time.Now().UTC()
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'user', 'hash', 'User', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `INSERT INTO debtors (id, user_id, name, created_at) VALUES ('debtor-1', ?, 'Денис', ?)`, userID, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	dueDate := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO debts (id, user_id, debtor_id, direction, amount, affects_balance, debt_date, due_date, is_settled, created_at)
		VALUES ('debt-1', ?, 'debtor-1', 'lent', 100000, 0, ?, ?, 0, ?)`,
		userID, now.Format("2006-01-02 15:04:05"), dueDate, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	secret := "12345678901234567890123456789012"
	_, err = manager.DB().ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned, debt_days_before, credit_days_before,
			notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 1, 0, 0, 1, 1, '00:00', ?)`,
		userID, token, now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(manager.DB(), slog.Default())
	settings := sqlNotificationSettings(t, manager, ctx, userID)
	if err := worker.runForUser(ctx, userID, now, now.In(time.FixedZone("MSK", 3*3600)), settings); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected telegram notifier to be called")
	}

	var count int
	if err := manager.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_log WHERE user_id = ? AND trigger_type = ?`, userID, TriggerDebtOverdue).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected notification log entry")
	}
}

func TestParseNotificationSendTime(t *testing.T) {
	h, m := parseNotificationSendTime("09:30")
	if h != 9 || m != 30 {
		t.Fatalf("time %d:%d", h, m)
	}
	h, m = parseNotificationSendTime("bad")
	if h != 0 || m != 0 {
		t.Fatalf("bad time %d:%d", h, m)
	}
}

func TestPickDebtTrigger(t *testing.T) {
	t.Parallel()

	settings := sqlcdb.NotificationSetting{
		DebtDaysBefore:                2,
		MyDebtOverdueDaysLimit:        7,
		OwedDebtOverdueStartAfterDays: 3,
		OwedDebtOverdueDaysLimit:      5,
	}

	if trigger, ok := pickDebtTrigger(settings, "borrowed", 2); !ok || trigger != TriggerDebtDueSoon {
		t.Fatalf("expected borrowed due soon for diff=2, got ok=%v trigger=%q", ok, trigger)
	}
	if trigger, ok := pickDebtTrigger(settings, "borrowed", 0); !ok || trigger != TriggerDebtDueSoon {
		t.Fatalf("expected borrowed due soon for diff=0, got ok=%v trigger=%q", ok, trigger)
	}
	if trigger, ok := pickDebtTrigger(settings, "borrowed", -7); !ok || trigger != TriggerDebtOverdue {
		t.Fatalf("expected borrowed overdue for diff=-7, got ok=%v trigger=%q", ok, trigger)
	}
	if _, ok := pickDebtTrigger(settings, "borrowed", -8); ok {
		t.Fatal("expected borrowed overdue limit to stop at day 8")
	}

	if trigger, ok := pickDebtTrigger(settings, "lent", 0); !ok || trigger != TriggerDebtDueSoon {
		t.Fatalf("expected lent due day trigger, got ok=%v trigger=%q", ok, trigger)
	}
	if _, ok := pickDebtTrigger(settings, "lent", -3); ok {
		t.Fatal("expected lent overdue delay to skip first 3 overdue days")
	}
	if trigger, ok := pickDebtTrigger(settings, "lent", -4); !ok || trigger != TriggerDebtOverdue {
		t.Fatalf("expected lent overdue trigger starting after delay, got ok=%v trigger=%q", ok, trigger)
	}
	if _, ok := pickDebtTrigger(settings, "lent", -9); ok {
		t.Fatal("expected lent overdue limit to stop after configured send days")
	}
}

func sqlNotificationSettings(t *testing.T, manager *db.Manager, ctx context.Context, userID string) sqlcdb.NotificationSetting {
	t.Helper()
	settings, err := sqlcdb.New(manager.DB()).GetNotificationSettings(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	return settings
}
