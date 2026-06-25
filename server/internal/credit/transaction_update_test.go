package credit

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestGuardTransactionUpdateBlocksAppliedPayment(t *testing.T) {
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
		PaymentDate: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
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

	if err := GuardTransactionUpdate(ctx, sqlDB, userID, txID); err != ErrCannotEditPayment {
		t.Fatalf("expected ErrCannotEditPayment, got %v", err)
	}
}
