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
	userID, err := auth.CreateUser(ctx, sqlDB, "credituser", hash, "Credit", false)
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
	updated, err := RemovePayment(ctx, sqlDB, userID, c.ID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Schedule) >= len(c.Schedule) {
		t.Fatal("expected schedule shrink")
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

func strPtr(s string) *string { return &s }
