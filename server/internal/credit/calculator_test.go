package credit

import "testing"

func TestMonthlyPaymentZeroRate(t *testing.T) {
	got := MonthlyPayment(120000, 0, 12)
	if got != 10000 {
		t.Fatalf("expected 10000, got %d", got)
	}
}

func TestMonthlyPayment12Percent(t *testing.T) {
	// 1_000_000 kopecks = 10000 rub, 12% annual, 12 months
	got := MonthlyPayment(1_000_000, 12, 12)
	if got < 88000 || got > 89000 {
		t.Fatalf("unexpected payment %d", got)
	}
}

func TestMonthlyPaymentEdgeTerm(t *testing.T) {
	if MonthlyPayment(10000, 0, 1) != 10000 {
		t.Fatal("single payment should equal principal at 0%")
	}
	if MonthlyPayment(0, 10, 12) != 0 {
		t.Fatal("zero principal")
	}
}

func TestRemainingAmount(t *testing.T) {
	if RemainingAmount(100000, 30000) != 70000 {
		t.Fatal("remaining mismatch")
	}
	if RemainingAmount(100000, 150000) != 0 {
		t.Fatal("overpaid should be 0")
	}
}
