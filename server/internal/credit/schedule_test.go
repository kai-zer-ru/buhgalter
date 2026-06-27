package credit

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestGenerateAutoScheduleEqualParts(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	entries, err := GenerateAutoSchedule(120000, 12, 10000, IntervalMonth, issue, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 12 {
		t.Fatalf("expected 12 entries, got %d", len(entries))
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 120000 {
		t.Fatalf("sum %d != principal", sum)
	}
}

func TestGenerateScheduleFromSeed(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	d1, _ := timeutil.ParseUTC("2024-02-01 00:00:00")
	d2, _ := timeutil.ParseUTC("2024-03-01 00:00:00")
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       100000,
		TermMonths:      5,
		MonthlyPayment:  20000,
		PaymentInterval: IntervalMonth,
		IssueDate:       issue,
		InterestRate:    0,
		SeedPayments: []ScheduleSeed{
			{PaymentDate: d1, Amount: 20000},
			{PaymentDate: d2, Amount: 20000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5, got %d", len(entries))
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 100000 {
		t.Fatalf("sum %d", sum)
	}
}

func TestGenerateScheduleWeekInterval(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-01 00:00:00")
	entries, err := GenerateAutoSchedule(40000, 4, 10000, IntervalWeek, issue, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4, got %d", len(entries))
	}
	first, _ := timeutil.ParseUTC(entries[0].PaymentDate)
	second, _ := timeutil.ParseUTC(entries[1].PaymentDate)
	if second.Sub(first) != 7*24*time.Hour {
		t.Fatalf("expected 7 day step, got %v", second.Sub(first))
	}
}

func TestGenerateManualSchedule(t *testing.T) {
	d1, _ := timeutil.ParseUTC("2024-02-01 00:00:00")
	d2, _ := timeutil.ParseUTC("2024-03-15 00:00:00")
	d3, _ := timeutil.ParseUTC("2024-05-01 00:00:00")
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       90000,
		TermMonths:      3,
		PaymentInterval: IntervalManual,
		SeedPayments: []ScheduleSeed{
			{PaymentDate: d1, Amount: 30000},
			{PaymentDate: d2, Amount: 30000},
			{PaymentDate: d3, Amount: 30000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3, got %d", len(entries))
	}
}

func TestGenerateManualScheduleAdjustsLast(t *testing.T) {
	d1, _ := timeutil.ParseUTC("2024-02-01 00:00:00")
	d2, _ := timeutil.ParseUTC("2024-03-15 00:00:00")
	d3, _ := timeutil.ParseUTC("2024-05-01 00:00:00")
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       100000,
		TermMonths:      3,
		PaymentInterval: IntervalManual,
		SeedPayments: []ScheduleSeed{
			{PaymentDate: d1, Amount: 33333},
			{PaymentDate: d2, Amount: 33333},
			{PaymentDate: d3, Amount: 33333},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries[2].Amount != 33334 {
		t.Fatalf("last payment want 33334, got %d", entries[2].Amount)
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 100000 {
		t.Fatalf("sum %d", sum)
	}
}

func TestGenerateScheduleFullSeedAdjustsLast(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	seeds := make([]ScheduleSeed, 12)
	d, _ := timeutil.ParseUTC("2024-02-01 00:00:00")
	for i := range seeds {
		seeds[i] = ScheduleSeed{PaymentDate: d.AddDate(0, i, 0), Amount: 25000}
	}
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       300001,
		TermMonths:      12,
		MonthlyPayment:  25000,
		PaymentInterval: IntervalMonth,
		IssueDate:       issue,
		InterestRate:    0,
		SeedPayments:    seeds,
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries[11].Amount != 25001 {
		t.Fatalf("last payment want 25001, got %d", entries[11].Amount)
	}
}

func TestGenerateScheduleLastPaymentAdjustment(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-01 00:00:00")
	entries, err := GenerateAutoSchedule(100001, 3, 33333, IntervalMonth, issue, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 100001 {
		t.Fatalf("sum %d", sum)
	}
}

func TestGenerateScheduleAnnuityTermMonths(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	principal := int64(1_000_000)
	term := 36
	rate := 12.0
	monthly := MonthlyPayment(principal, rate, term)
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       principal,
		TermMonths:      term,
		MonthlyPayment:  monthly,
		PaymentInterval: IntervalMonth,
		IssueDate:       issue,
		InterestRate:    rate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != term {
		t.Fatalf("expected %d entries, got %d", term, len(entries))
	}
	last := entries[len(entries)-1]
	for i, e := range entries[:len(entries)-1] {
		if e.Amount != monthly {
			t.Fatalf("payment %d: amount %d want %d", i+1, e.Amount, monthly)
		}
	}
	if last.Amount >= monthly {
		t.Fatalf("last payment %d should be less than regular %d", last.Amount, monthly)
	}
	if last.Amount <= 0 {
		t.Fatalf("last payment must be positive, got %d", last.Amount)
	}
}

// Alfa-Bank-style amortization: equal payments, last one clears remaining debt (smaller).
func TestGenerateScheduleAlfaBankExample(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	principal := int64(31_000_000) // 310 000 ₽
	term := 36
	// Rate from first payment in bank schedule (10002.30 ₽ interest on 310 000 ₽).
	rate := 10002.30 / 310000.0 * 12 * 100
	monthly := MonthlyPayment(principal, rate, term)

	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       principal,
		TermMonths:      term,
		MonthlyPayment:  monthly,
		PaymentInterval: IntervalMonth,
		IssueDate:       issue,
		InterestRate:    rate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != term {
		t.Fatalf("expected %d entries, got %d", term, len(entries))
	}
	for i := 0; i < term-1; i++ {
		if entries[i].Amount != monthly {
			t.Fatalf("payment %d: amount %d want %d", i+1, entries[i].Amount, monthly)
		}
	}
	last := entries[term-1]
	if last.Amount >= monthly {
		t.Fatalf("last payment %d should be less than regular %d", last.Amount, monthly)
	}
	if last.Amount <= 0 {
		t.Fatalf("last payment must be positive, got %d", last.Amount)
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum <= principal {
		t.Fatalf("total payments %d should exceed principal %d (includes interest)", sum, principal)
	}
}

func TestGenerateScheduleMortgageAutoPaymentLongTerm(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2021-01-01 00:00:00")
	principal := int64(3_595_550_000) // 35 955 500.00
	term := 360
	rate := 20.0
	monthly := MonthlyPaymentMortgage(principal, rate, term, issue)

	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       principal,
		TermMonths:      term,
		MonthlyPayment:  monthly,
		UserSetPayment:  false,
		PaymentInterval: IntervalMonth,
		CreditKind:      CreditKindMortgage,
		IssueDate:       issue,
		InterestRate:    rate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v (monthly=%d)", err, monthly)
	}
	if len(entries) != term {
		t.Fatalf("expected %d entries, got %d", term, len(entries))
	}
	last := entries[term-1]
	if last.Amount <= 0 {
		t.Fatalf("last payment must be positive, got %d", last.Amount)
	}
	for i, e := range entries {
		if e.Amount <= 0 {
			t.Fatalf("payment %d: non-positive amount %d", i+1, e.Amount)
		}
	}
	for i := 0; i < term-1; i++ {
		if entries[i].Amount != monthly {
			t.Fatalf("payment %d: amount %d want %d", i+1, entries[i].Amount, monthly)
		}
	}
}

func TestGenerateScheduleMortgageUserSetPaymentSameAsCalculated(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2021-01-01 00:00:00")
	principal := int64(3_595_550_000)
	term := 360
	rate := 20.0
	monthly := MonthlyPaymentMortgage(principal, rate, term, issue)

	_, errAuto := GenerateSchedule(ScheduleInput{
		Principal: principal, TermMonths: term, MonthlyPayment: monthly, UserSetPayment: false,
		PaymentInterval: IntervalMonth, CreditKind: CreditKindMortgage, IssueDate: issue, InterestRate: rate,
	})
	if errAuto != nil {
		t.Fatalf("auto: %v", errAuto)
	}
	_, errUser := GenerateSchedule(ScheduleInput{
		Principal: principal, TermMonths: term, MonthlyPayment: monthly, UserSetPayment: true,
		PaymentInterval: IntervalMonth, CreditKind: CreditKindMortgage, IssueDate: issue, InterestRate: rate,
	})
	if errUser != nil {
		t.Fatalf("user-set same as calculated: %v", errUser)
	}

	rounded := (monthly / 100) * 100
	preview, resolved, err := PreviewSchedule(PreviewInput{
		Principal: principal, TermMonths: term, InterestRate: rate,
		PaymentInterval: IntervalMonth, IssueDate: issue, CreditKind: CreditKindMortgage,
		MonthlyPayment: &rounded,
	})
	if err != nil {
		t.Fatalf("preview rounded payment: %v", err)
	}
	if len(preview) != term {
		t.Fatalf("preview len %d, want %d", len(preview), term)
	}
	if resolved != monthly {
		t.Fatalf("resolved monthly %d, want calculated %d", resolved, monthly)
	}
}

func TestGenerateScheduleMortgageUserSetBankPayment(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2021-01-01 00:00:00")
	principal := int64(3_595_550_000)
	term := 360
	rate := 20.0
	bankPayment := int64(61_074_247) // e.g. Sber contract payment
	entries, err := GenerateSchedule(ScheduleInput{
		Principal: principal, TermMonths: term, MonthlyPayment: bankPayment, UserSetPayment: true,
		PaymentInterval: IntervalMonth, CreditKind: CreditKindMortgage, IssueDate: issue, InterestRate: rate,
	})
	if err != nil {
		t.Fatalf("user-set bank payment: %v", err)
	}
	if len(entries) != term {
		t.Fatalf("expected %d entries, got %d", term, len(entries))
	}
	for i, e := range entries {
		if e.Amount <= 0 {
			t.Fatalf("payment %d: non-positive amount %d", i+1, e.Amount)
		}
	}
	for i := 0; i < term-1; i++ {
		if entries[i].Amount != bankPayment {
			t.Fatalf("payment %d: amount %d want %d", i+1, entries[i].Amount, bankPayment)
		}
	}
	if entries[term-1].Amount != bankPayment {
		t.Fatalf("last payment %d want bank payment %d", entries[term-1].Amount, bankPayment)
	}
}
