package credit

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestComputeFallbackNextPayment(t *testing.T) {
	issue := time.Date(2022, 11, 10, 12, 27, 0, 0, time.UTC)
	f := creditFields{
		status:          "active",
		issueDate:       timeutil.FormatUTC(issue),
		paymentInterval: string(IntervalMonth),
		principal:       315000000,
		paidAmount:      49 * 3500000,
		monthlyPayment:  3500000,
		termMonths:      120,
	}

	d, a, err := computeFallbackNextPayment(f)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || a == nil {
		t.Fatal("expected next payment")
	}
	if *a != 3500000 {
		t.Fatalf("amount %d", *a)
	}
	parsed, err := timeutil.ParseUTC(*d)
	if err != nil {
		t.Fatal(err)
	}
	expected := issue
	for i := 0; i <= 49; i++ {
		expected = nextPaymentDate(expected, IntervalMonth)
	}
	if !parsed.Equal(expected) {
		t.Fatalf("date %v expected %v", parsed, expected)
	}
}

func TestComputeFallbackNextPaymentClosed(t *testing.T) {
	f := creditFields{
		status:     "closed",
		principal:  10000,
		paidAmount: 10000,
	}
	d, a, err := computeFallbackNextPayment(f)
	if err != nil {
		t.Fatal(err)
	}
	if d != nil || a != nil {
		t.Fatal("closed credit should have no next payment")
	}
}
