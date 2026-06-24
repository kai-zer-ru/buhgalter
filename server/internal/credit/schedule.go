package credit

import (
	"fmt"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type PaymentInterval string

const (
	IntervalMonth    PaymentInterval = "month"
	IntervalWeek     PaymentInterval = "week"
	IntervalTwoWeeks PaymentInterval = "two_weeks"
	IntervalManual   PaymentInterval = "manual"
)

type ScheduleSeed struct {
	PaymentDate time.Time
	Amount      int64
}

type ScheduleEntry struct {
	PaymentDate string `json:"payment_date"`
	Amount      int64  `json:"amount"`
	AmountDisplay string `json:"amount_display,omitempty"`
}

type ScheduleInput struct {
	Principal       int64
	TermMonths      int
	MonthlyPayment  int64
	PaymentInterval PaymentInterval
	IssueDate       time.Time
	SeedPayments    []ScheduleSeed
}

func (pi PaymentInterval) Valid() bool {
	switch pi {
	case IntervalMonth, IntervalWeek, IntervalTwoWeeks, IntervalManual:
		return true
	default:
		return false
	}
}

func nextPaymentDate(from time.Time, interval PaymentInterval) time.Time {
	switch interval {
	case IntervalWeek:
		return from.AddDate(0, 0, 7)
	case IntervalTwoWeeks:
		return from.AddDate(0, 0, 14)
	default:
		return from.AddDate(0, 1, 0)
	}
}

// GenerateSchedule builds payment schedule entries without persisting.
func GenerateSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	if in.TermMonths <= 0 {
		return nil, fmt.Errorf("term must be positive")
	}
	if !in.PaymentInterval.Valid() {
		return nil, fmt.Errorf("invalid payment interval")
	}
	if in.PaymentInterval == IntervalManual {
		return generateManualSchedule(in)
	}
	if in.MonthlyPayment <= 0 {
		return nil, fmt.Errorf("monthly payment must be positive")
	}

	entries := make([]ScheduleEntry, 0, in.TermMonths)
	remaining := in.Principal

	if len(in.SeedPayments) > 0 {
		for _, seed := range in.SeedPayments {
			if seed.Amount <= 0 {
				return nil, fmt.Errorf("seed payment amount must be positive")
			}
			entries = append(entries, ScheduleEntry{
				PaymentDate: timeutil.FormatUTC(seed.PaymentDate),
				Amount:      seed.Amount,
			})
			remaining -= seed.Amount
		}
	}

	var lastDate time.Time
	if len(entries) > 0 {
		last, err := timeutil.ParseUTC(entries[len(entries)-1].PaymentDate)
		if err != nil {
			return nil, err
		}
		lastDate = last
	} else {
		lastDate = in.IssueDate
	}

	for len(entries) < in.TermMonths && remaining > 0 {
		lastDate = nextPaymentDate(lastDate, in.PaymentInterval)
		amount := in.MonthlyPayment
		if len(entries) == in.TermMonths-1 || amount >= remaining {
			amount = remaining
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(lastDate),
			Amount:      amount,
		})
		remaining -= amount
	}

	if len(entries) == in.TermMonths {
		if err := normalizeScheduleSum(entries, in.Principal); err != nil {
			return nil, err
		}
	}

	return entries, nil
}

// normalizeScheduleSum adjusts the last payment so entry amounts sum to principal.
func normalizeScheduleSum(entries []ScheduleEntry, principal int64) error {
	if len(entries) == 0 {
		return nil
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum == principal {
		return nil
	}
	last := &entries[len(entries)-1]
	last.Amount += principal - sum
	if last.Amount <= 0 {
		return fmt.Errorf("schedule sum does not match principal")
	}
	return nil
}

func generateManualSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	if len(in.SeedPayments) != in.TermMonths {
		return nil, fmt.Errorf("manual schedule requires %d payments", in.TermMonths)
	}
	entries := make([]ScheduleEntry, 0, len(in.SeedPayments))
	for _, seed := range in.SeedPayments {
		if seed.Amount <= 0 {
			return nil, fmt.Errorf("seed payment amount must be positive")
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(seed.PaymentDate),
			Amount:      seed.Amount,
		})
	}
	if err := normalizeScheduleSum(entries, in.Principal); err != nil {
		return nil, err
	}
	return entries, nil
}

// GenerateAutoSchedule builds schedule from issue_date without seed rows.
func GenerateAutoSchedule(principal int64, term int, monthlyPayment int64, interval PaymentInterval, issueDate time.Time) ([]ScheduleEntry, error) {
	return GenerateSchedule(ScheduleInput{
		Principal:       principal,
		TermMonths:      term,
		MonthlyPayment:  monthlyPayment,
		PaymentInterval: interval,
		IssueDate:       issueDate,
	})
}
