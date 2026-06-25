package credit

import (
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestUpdateScheduleAmounts(t *testing.T) {
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
	var pendingTxID string
	var oldAmount int64
	for _, p := range c.Schedule {
		if p.Kind == "scheduled" && !p.IsApplied {
			pendingID = p.ID
			oldAmount = p.Amount
			if p.TransactionID != nil {
				pendingTxID = *p.TransactionID
			}
			break
		}
	}
	if pendingID == "" {
		t.Fatal("no pending payment")
	}

	newAmount := oldAmount + 100
	updated, err := UpdateScheduleAmounts(ctx, sqlDB, userID, c.ID, []ScheduleAmountUpdate{
		{PaymentID: pendingID, Amount: newAmount},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got int64
	for _, p := range updated.Schedule {
		if p.ID == pendingID {
			got = p.Amount
			break
		}
	}
	if got != newAmount {
		t.Fatalf("payment amount %d want %d", got, newAmount)
	}

	if pendingTxID != "" {
		var txAmount int64
		if err := sqlDB.QueryRowContext(ctx, `SELECT amount FROM transactions WHERE id = ?`, pendingTxID).Scan(&txAmount); err != nil {
			t.Fatal(err)
		}
		if txAmount != newAmount {
			t.Fatalf("future tx amount %d want %d", txAmount, newAmount)
		}
	}

	_, err = UpdateScheduleAmounts(ctx, sqlDB, userID, c.ID, []ScheduleAmountUpdate{
		{PaymentID: pendingID, Amount: 0},
	})
	if err != ErrInvalidAmount {
		t.Fatalf("zero amount err %v", err)
	}

	for _, p := range c.Schedule {
		if p.IsApplied {
			_, err = UpdateScheduleAmounts(ctx, sqlDB, userID, c.ID, []ScheduleAmountUpdate{
				{PaymentID: p.ID, Amount: p.Amount + 100},
			})
			if err != ErrCannotEditPayment {
				t.Fatalf("applied payment err %v", err)
			}
			break
		}
	}
}
