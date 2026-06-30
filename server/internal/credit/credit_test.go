package credit

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func seedCreditEnv(t *testing.T) (context.Context, *db.Handle, string, string) {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "credituser", hash, "Credit", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	accountID := "acc-credit"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Ипотека-счёт', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, db.NewHandle(mgr), userID, accountID
}

func TestPayNextScheduledAlternateAccount(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	altAccountID := "acc-alt"
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Другой счёт', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		altAccountID, userID)
	if err != nil {
		t.Fatal(err)
	}

	issue := timeutil.NowUTC().AddDate(0, 1, 0)
	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount: 60_000,
		IssueDate:       issue,
		TermMonths:      6,
		PaymentInterval: IntervalMonth,
		DebitAccountID:  accountID,
	})
	if err != nil {
		t.Fatal(err)
	}

	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount: c.MonthlyPayment, PaymentDate: timeutil.NowUTC(), AccountID: altAccountID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if paid.DebitAccountID != accountID {
		t.Fatalf("credit debit account changed: %s", paid.DebitAccountID)
	}

	var txAccountID string
	for _, p := range paid.Schedule {
		if p.TransactionID != nil {
			if err := sqlDB.QueryRowContext(ctx, `SELECT account_id FROM transactions WHERE id = ?`, *p.TransactionID).Scan(&txAccountID); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if txAccountID != altAccountID {
		t.Fatalf("expected tx on %s, got %s", altAccountID, txAccountID)
	}
}

func TestCreateListGetCredit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		Name:               strPtr("Ипотека"),
		PrincipalAmount:    1_000_000,
		IssueDate:          issue,
		TermMonths:         12,
		InterestRate:       0,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.RemainingAmount != 1_000_000 {
		t.Fatalf("remaining %d", c.RemainingAmount)
	}
	if len(c.Schedule) != 12 {
		t.Fatalf("schedule len %d", len(c.Schedule))
	}

	list, err := List(ctx, sqlDB, userID, "active")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len %d", len(list))
	}

	got, err := GetByID(ctx, sqlDB, userID, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID {
		t.Fatalf("GetByID: %+v", got)
	}
}

func TestPreviewSchedule(t *testing.T) {
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	preview, monthly, err := PreviewSchedule(PreviewInput{
		Principal:       600_000,
		IssueDate:       issue,
		TermMonths:      6,
		InterestRate:    0,
		PaymentInterval: IntervalMonth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != 6 {
		t.Fatalf("preview len %d", len(preview))
	}
	if monthly <= 0 {
		t.Fatalf("monthly %d", monthly)
	}
}

func TestUpdateAndCloseCredit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    300_000,
		IssueDate:          issue,
		TermMonths:         3,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	newName := "Обновлённый"
	newMonthly := int64(120_000)
	updated, err := Update(ctx, sqlDB, userID, c.ID, UpdateInput{
		Name:           &newName,
		MonthlyPayment: &newMonthly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name == nil || *updated.Name != newName {
		t.Fatalf("name not updated: %+v", updated.Name)
	}

	closed, err := Close(ctx, sqlDB, userID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != "closed" {
		t.Fatalf("status %s", closed.Status)
	}
}

func TestCreateValidationErrors(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC()

	_, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount: 0, IssueDate: issue, TermMonths: 12,
		PaymentInterval: IntervalMonth, DebitAccountID: accountID,
	})
	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	_, err = Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount: 100_000, IssueDate: issue, TermMonths: 0,
		PaymentInterval: IntervalMonth, DebitAccountID: accountID,
	})
	if err != ErrInvalidTerm {
		t.Fatalf("expected ErrInvalidTerm, got %v", err)
	}
}

func TestTodayCutoffUTC(t *testing.T) {
	cutoff, err := TodayCutoffUTC("Europe/Moscow", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if cutoff == "" {
		t.Fatal("expected non-empty cutoff")
	}
}

func TestCompleteCredit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	closed, err := Complete(ctx, sqlDB, userID, c.ID, CompleteInput{
		AffectsBalance: false,
		PaymentDate:    timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != "closed" {
		t.Fatalf("status %s", closed.Status)
	}
}

func TestApplyDuePayments(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)
	localTime := timeutil.NowUTC().In(time.FixedZone("MSK", 3*3600)).Format("15:04")

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		DebitTimeLocal:     &localTime,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		UPDATE credit_payments SET payment_date = datetime('now', '-1 day')
		WHERE id = (
			SELECT id FROM credit_payments
			WHERE credit_id = ? AND is_applied = 0 AND kind = 'scheduled' LIMIT 1
		)`, c.ID)
	if err != nil {
		t.Fatal(err)
	}

	cutoff, err := TodayCutoffUTC("Europe/Moscow", timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	n, err := ApplyDuePayments(ctx, sqlDB, userID, cutoff, localTime)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("applied %d", n)
	}

	var txKind string
	err = sqlDB.QueryRowContext(ctx, `
		SELECT t.kind FROM transactions t
		JOIN credit_payments cp ON cp.transaction_id = t.id
		WHERE cp.credit_id = ? AND cp.is_applied = 1
		ORDER BY cp.payment_date DESC LIMIT 1`, c.ID).Scan(&txKind)
	if err != nil {
		t.Fatal(err)
	}
	if txKind != "manual" {
		t.Fatalf("expected manual tx after apply, got %s", txKind)
	}
}

func TestPaymentTransactionDate(t *testing.T) {
	payDate, err := timeutil.ParseUTC("2026-06-15 00:00:00")
	if err != nil {
		t.Fatal(err)
	}
	debitTime := "10:00"
	got, err := paymentTransactionDate(payDate, &debitTime, "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	local := got.In(loc)
	if local.Format("15:04") != "10:00" {
		t.Fatalf("expected 10:00 local, got %s", local.Format("15:04"))
	}
	if local.Format("2006-01-02") != "2026-06-15" {
		t.Fatalf("expected 2026-06-15, got %s", local.Format("2006-01-02"))
	}
}

func TestApplyDuePaymentsUsesDebitTimeForTransaction(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)
	debitTime := "10:00"

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		DebitTimeLocal:     &debitTime,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		UPDATE credit_payments SET payment_date = datetime('now', '-1 day')
		WHERE id = (
			SELECT id FROM credit_payments
			WHERE credit_id = ? AND is_applied = 0 AND kind = 'scheduled' LIMIT 1
		)`, c.ID)
	if err != nil {
		t.Fatal(err)
	}

	cutoff, err := TodayCutoffUTC("Europe/Moscow", timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	n, err := ApplyDuePayments(ctx, sqlDB, userID, cutoff, debitTime)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("applied %d", n)
	}

	var txDate string
	err = sqlDB.QueryRowContext(ctx, `
		SELECT t.transaction_date FROM transactions t
		JOIN credit_payments cp ON cp.transaction_id = t.id
		WHERE cp.credit_id = ? AND cp.kind = 'auto' LIMIT 1`, c.ID).Scan(&txDate)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := timeutil.ParseUTC(txDate)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	local := parsed.In(loc)
	if local.Format("15:04") != debitTime {
		t.Fatalf("expected transaction at %s local, got %s (%s)", debitTime, local.Format("15:04"), txDate)
	}
}

func TestApplyDuePaymentsWithoutPrecreatedTx(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)
	localTime := timeutil.NowUTC().In(time.FixedZone("MSK", 3*3600)).Format("15:04")

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		DebitTimeLocal:     &localTime,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		UPDATE credit_payments SET payment_date = datetime('now', '-1 day')
		WHERE id = (
			SELECT id FROM credit_payments
			WHERE credit_id = ? AND is_applied = 0 AND kind = 'scheduled' LIMIT 1
		)`, c.ID)
	if err != nil {
		t.Fatal(err)
	}

	cutoff, err := TodayCutoffUTC("Europe/Moscow", timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	n, err := ApplyDuePayments(ctx, sqlDB, userID, cutoff, localTime)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("applied %d", n)
	}
	got, err := GetByID(ctx, sqlDB, userID, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaidAmount <= 0 {
		t.Fatalf("paid %d", got.PaidAmount)
	}
}

func TestDeleteKeepTransactions(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount: c.MonthlyPayment, PaymentDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var txID string
	for _, p := range paid.Schedule {
		if p.TransactionID != nil {
			txID = *p.TransactionID
			break
		}
	}
	if txID == "" {
		t.Fatal("expected transaction")
	}
	if err := Delete(ctx, sqlDB, userID, c.ID, "keep_transactions"); err != nil {
		t.Fatal(err)
	}
	var cnt int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions WHERE id = ?`, txID).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("transaction should remain, count %d", cnt)
	}
}

func TestCreateWithInterestRate(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    1_000_000,
		IssueDate:          issue,
		TermMonths:         12,
		InterestRate:       12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.MonthlyPayment <= 0 {
		t.Fatalf("monthly %d", c.MonthlyPayment)
	}
	preview, monthly, err := PreviewSchedule(PreviewInput{
		Principal: 1_000_000, IssueDate: issue, TermMonths: 12,
		InterestRate: 12, PaymentInterval: IntervalMonth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != 12 || monthly <= 0 {
		t.Fatalf("preview len %d monthly %d", len(preview), monthly)
	}
}

func TestCreateCredit36MonthsWithInterest(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    1_000_000,
		IssueDate:          issue,
		TermMonths:         36,
		InterestRate:       12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Schedule) != 36 {
		t.Fatalf("schedule len %d, want 36", len(c.Schedule))
	}
}

func TestRepairShortSchedules(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    1_000_000,
		IssueDate:          issue,
		TermMonths:         36,
		InterestRate:       12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	payments, err := queries(sqlDB).ListCreditPayments(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 36 {
		t.Fatalf("setup: expected 36 payments, got %d", len(payments))
	}
	for i := 22; i < len(payments); i++ {
		if _, err := sqlDB.ExecContext(ctx, `DELETE FROM credit_payments WHERE id = ?`, payments[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := RepairShortSchedules(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	payments, err = queries(sqlDB).ListCreditPayments(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 36 {
		t.Fatalf("after repair: expected 36 payments, got %d", len(payments))
	}
}

func TestAutoCloseOnFullPayment(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)
	localTime := timeutil.NowUTC().In(time.FixedZone("MSK", 3*3600)).Format("15:04")

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    12_000,
		IssueDate:          issue,
		TermMonths:         1,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		DebitTimeLocal:     &localTime,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		UPDATE credit_payments SET payment_date = datetime('now', '-1 day'), amount = ?
		WHERE credit_id = ?`, c.PrincipalAmount, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	cutoff, err := TodayCutoffUTC("Europe/Moscow", timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyDuePayments(ctx, sqlDB, userID, cutoff, localTime); err != nil {
		t.Fatal(err)
	}
	got, err := GetByID(ctx, sqlDB, userID, c.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "closed" {
		t.Fatalf("expected closed, got %s paid %d", got.Status, got.PaidAmount)
	}
}

func TestCreateRetroactiveCredit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -2, 0)
	added := true

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		AddedRetroactively: &added,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !c.AddedRetroactively {
		t.Fatal("expected retroactive flag")
	}
	var retro int
	for _, p := range c.Schedule {
		if p.Kind == "retroactive" {
			retro++
		}
	}
	if retro == 0 {
		t.Fatal("expected retroactive payments")
	}
}

func TestCreateRetroactiveCreditWithAccountDebit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	_, err := sqlDB.ExecContext(ctx,
		`UPDATE accounts SET initial_balance = 1000000 WHERE id = ?`, accountID)
	if err != nil {
		t.Fatal(err)
	}
	issue := timeutil.NowUTC().AddDate(0, -2, 0)
	added := true
	debitCount := 1

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:       120_000,
		IssueDate:             issue,
		TermMonths:            12,
		PaymentInterval:       IntervalMonth,
		DebitAccountID:        accountID,
		AddedRetroactively:    &added,
		RetroactiveDebitCount: debitCount,
		CreateTransactions:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	var retroTotal, debited, notDebited int
	var debitedAmount int64
	for _, p := range c.Schedule {
		if p.Kind != "retroactive" {
			continue
		}
		retroTotal++
		if p.TransactionID != nil {
			debited++
			debitedAmount += p.Amount
			if p.ExcludeFromStats {
				t.Fatal("debited retro payment should count in stats")
			}
		} else {
			notDebited++
			if !p.ExcludeFromStats {
				t.Fatal("non-debited retro should be excluded from stats")
			}
		}
	}
	if retroTotal < 1 {
		t.Fatalf("expected at least 1 retro payment, got %d", retroTotal)
	}
	if debited != debitCount {
		t.Fatalf("expected %d debited retro payments, got %d", debitCount, debited)
	}
	if notDebited != retroTotal-debitCount {
		t.Fatalf("unexpected non-debited retro count %d", notDebited)
	}
	var txCount int
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM transactions WHERE user_id = ? AND account_id = ?`,
		userID, accountID,
	).Scan(&txCount); err != nil {
		t.Fatal(err)
	}
	if txCount != debitCount {
		t.Fatalf("expected %d transactions, got %d", debitCount, txCount)
	}
	if c.PaidAmount < debitedAmount {
		t.Fatalf("paid_amount %d should include debited %d", c.PaidAmount, debitedAmount)
	}
}

func TestCreateRetroactiveDebitCountInvalid(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -2, 0)
	added := true
	tooMany := 99

	_, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:       120_000,
		IssueDate:             issue,
		TermMonths:            12,
		PaymentInterval:       IntervalMonth,
		DebitAccountID:        accountID,
		AddedRetroactively:    &added,
		RetroactiveDebitCount: tooMany,
	})
	if !errors.Is(err, ErrInvalidRetroactiveDebit) {
		t.Fatalf("expected ErrInvalidRetroactiveDebit, got %v", err)
	}
}

func TestRemovePayment(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 2, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var pendingID string
	for _, p := range c.Schedule {
		if p.Kind == "scheduled" && !p.IsApplied {
			pendingID = p.ID
			break
		}
	}
	if pendingID == "" {
		t.Fatal("no pending payment")
	}
	_, err = RemovePayment(ctx, sqlDB, userID, c.ID, pendingID)
	if !errors.Is(err, ErrOnlyLatestPaymentDelete) {
		t.Fatalf("expected ErrOnlyLatestPaymentDelete, got %v", err)
	}
}

func TestRemoveAppliedScheduledPaymentRestoresUnpaidRow(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	var appliedID string
	beforeLen := len(paid.Schedule)
	for _, p := range paid.Schedule {
		if p.Kind == "scheduled" && p.IsApplied && p.TransactionID != nil {
			appliedID = p.ID
			break
		}
	}
	if appliedID == "" {
		t.Fatal("no applied scheduled payment")
	}

	updated, err := RemovePayment(ctx, sqlDB, userID, c.ID, appliedID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Schedule) != beforeLen {
		t.Fatalf("expected schedule len %d, got %d", beforeLen, len(updated.Schedule))
	}

	var restored *CreditPayment
	for i := range updated.Schedule {
		if updated.Schedule[i].ID == appliedID {
			restored = &updated.Schedule[i]
			break
		}
	}
	if restored == nil {
		t.Fatal("restored schedule row not found")
	}
	if restored.IsApplied {
		t.Fatal("restored row must be unpaid")
	}
	if restored.TransactionID != nil {
		t.Fatal("restored row must not keep transaction link")
	}
	if restored.Kind != "scheduled" {
		t.Fatalf("restored row kind should be scheduled, got %s", restored.Kind)
	}
}

func TestRepairScheduleOnList(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, 1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    60_000,
		IssueDate:          issue,
		TermMonths:         6,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM credit_payments WHERE credit_id = ?`, c.ID); err != nil {
		t.Fatal(err)
	}
	list, err := List(ctx, sqlDB, userID, "active")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len %d", len(list))
	}
	got, err := GetByID(ctx, sqlDB, userID, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Schedule) == 0 {
		t.Fatal("expected repaired schedule on get")
	}
}

func TestOnTransactionDeleteRevertsAppliedPayment(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaidAmount <= 0 {
		t.Fatalf("paid amount %d", paid.PaidAmount)
	}

	var txID string
	for _, p := range paid.Schedule {
		if p.TransactionID != nil && p.IsApplied {
			txID = *p.TransactionID
			break
		}
	}
	if txID == "" {
		t.Fatal("expected applied payment with transaction")
	}
	paidBefore := paid.PaidAmount
	q := queries(sqlDB)
	if err := OnTransactionDelete(ctx, q, userID, txID); err != nil {
		t.Fatal(err)
	}
	got, err := GetByID(ctx, sqlDB, userID, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaidAmount >= paidBefore {
		t.Fatalf("paid should roll back: before %d after %d", paidBefore, got.PaidAmount)
	}
}

func TestPayNextScheduledAndDelete(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaidAmount <= 0 {
		t.Fatalf("paid amount %d", paid.PaidAmount)
	}

	if err := Delete(ctx, sqlDB, userID, c.ID, "cascade"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetByID(ctx, sqlDB, userID, c.ID, false); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPayNextScheduledWithFutureDateAllowed(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -1, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	futurePayDate := timeutil.NowUTC().AddDate(0, 1, 0)
	paid, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: futurePayDate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaidAmount <= 0 {
		t.Fatalf("paid amount %d", paid.PaidAmount)
	}
	var foundFutureTx bool
	for _, p := range paid.Schedule {
		if p.Kind == "scheduled" && p.IsApplied && p.TransactionKind != nil && *p.TransactionKind == "future" {
			foundFutureTx = true
			break
		}
	}
	if !foundFutureTx {
		t.Fatal("expected applied scheduled payment with future transaction kind")
	}
}

func TestRemovePaymentRejectsNotLatestApplied(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	issue := timeutil.NowUTC().AddDate(0, -2, 0)

	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount:    120_000,
		IssueDate:          issue,
		TermMonths:         12,
		PaymentInterval:    IntervalMonth,
		DebitAccountID:     accountID,
		CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paid1, err := PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: timeutil.NowUTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = PayNextScheduled(ctx, sqlDB, userID, c.ID, PayPaymentInput{
		Amount:      c.MonthlyPayment,
		PaymentDate: timeutil.NowUTC().AddDate(0, 0, 1),
	})
	if err != nil {
		t.Fatal(err)
	}
	var firstAppliedID string
	for _, p := range paid1.Schedule {
		if p.IsApplied && p.Kind == "scheduled" && p.TransactionID != nil {
			firstAppliedID = p.ID
			break
		}
	}
	if firstAppliedID == "" {
		t.Fatal("first applied payment not found")
	}
	_, err = RemovePayment(ctx, sqlDB, userID, c.ID, firstAppliedID)
	if !errors.Is(err, ErrOnlyLatestPaymentDelete) {
		t.Fatalf("expected ErrOnlyLatestPaymentDelete, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
