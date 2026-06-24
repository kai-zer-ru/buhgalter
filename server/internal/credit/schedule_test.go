package credit

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestGenerateAutoScheduleEqualParts(t *testing.T) {
	issue, _ := timeutil.ParseUTC("2024-01-15 00:00:00")
	entries, err := GenerateAutoSchedule(120000, 12, 10000, IntervalMonth, issue)
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
	entries, err := GenerateAutoSchedule(40000, 4, 10000, IntervalWeek, issue)
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
	entries, err := GenerateAutoSchedule(100001, 3, 33333, IntervalMonth, issue)
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
