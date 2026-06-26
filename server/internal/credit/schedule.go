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
	PaymentDate   string `json:"payment_date"`
	Amount        int64  `json:"amount"`
	AmountDisplay string `json:"amount_display,omitempty"`
}

type ScheduleInput struct {
	Principal       int64
	TermMonths      int
	MonthlyPayment  int64
	PaymentInterval PaymentInterval
	CreditKind      string
	IssueDate       time.Time
	InterestRate    float64
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

	if in.InterestRate > 0 {
		if len(in.SeedPayments) == in.TermMonths {
			entries := make([]ScheduleEntry, len(in.SeedPayments))
			for i, seed := range in.SeedPayments {
				if seed.Amount <= 0 {
					return nil, fmt.Errorf("seed payment amount must be positive")
				}
				entries[i] = ScheduleEntry{
					PaymentDate: timeutil.FormatUTC(seed.PaymentDate),
					Amount:      seed.Amount,
				}
			}
			return entries, nil
		}
		return generateInterestSchedule(in)
	}

	entries := make([]ScheduleEntry, 0, in.TermMonths)

	for _, seed := range in.SeedPayments {
		if seed.Amount <= 0 {
			return nil, fmt.Errorf("seed payment amount must be positive")
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(seed.PaymentDate),
			Amount:      seed.Amount,
		})
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

	for len(entries) < in.TermMonths {
		lastDate = nextPaymentDate(lastDate, in.PaymentInterval)
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(lastDate),
			Amount:      in.MonthlyPayment,
		})
	}

	if err := normalizeScheduleSum(entries, in.Principal); err != nil {
		return nil, err
	}

	return entries, nil
}

func periodRate(annualRate float64, interval PaymentInterval) float64 {
	switch interval {
	case IntervalWeek:
		return annualRate / 52 / 100
	case IntervalTwoWeeks:
		return annualRate / 26 / 100
	default:
		return annualRate / 12 / 100
	}
}

func schedulePaymentDates(in ScheduleInput) ([]time.Time, error) {
	dates := make([]time.Time, 0, in.TermMonths)
	lastDate := in.IssueDate
	for i := 0; i < in.TermMonths; i++ {
		if i < len(in.SeedPayments) {
			lastDate = in.SeedPayments[i].PaymentDate
			dates = append(dates, lastDate)
			continue
		}
		lastDate = nextPaymentDate(lastDate, in.PaymentInterval)
		dates = append(dates, lastDate)
	}
	return dates, nil
}

// generateInterestSchedule builds an amortizing schedule: equal payments except the
// last one, which clears the remaining principal plus period interest (bank-style).
func generateInterestSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	dates, err := schedulePaymentDates(in)
	if err != nil {
		return nil, err
	}
	rate := periodRate(in.InterestRate, in.PaymentInterval)
	balance := float64(in.Principal)
	entries := make([]ScheduleEntry, 0, in.TermMonths)
	prevDate := in.IssueDate

	for i := 0; i < in.TermMonths; i++ {
		interest := int64(balance * rate) // truncate kopecks, bank-style
		if normalizeCreditKind(in.CreditKind) == CreditKindMortgage && in.PaymentInterval == IntervalMonth {
			days := int(dates[i].Sub(prevDate).Hours() / 24)
			if days < 1 {
				days = 1
			}
			yearDays := 365.0
			if dates[i].Year()%4 == 0 && (dates[i].Year()%100 != 0 || dates[i].Year()%400 == 0) {
				yearDays = 366.0
			}
			dailyRate := in.InterestRate / 100 / yearDays
			interest = int64(balance * dailyRate * float64(days))
		}
		var amount int64
		if i == in.TermMonths-1 {
			balanceK := int64(balance)
			amount = balanceK + interest
			balance = 0
		} else {
			amount = in.MonthlyPayment
			principalPart := amount - interest
			if principalPart < 0 {
				return nil, fmt.Errorf("monthly payment is too low for interest accrual")
			}
			balance -= float64(principalPart)
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(dates[i]),
			Amount:      amount,
		})
		prevDate = dates[i]
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
func GenerateAutoSchedule(principal int64, term int, monthlyPayment int64, interval PaymentInterval, issueDate time.Time, interestRate float64) ([]ScheduleEntry, error) {
	return GenerateSchedule(ScheduleInput{
		Principal:       principal,
		TermMonths:      term,
		MonthlyPayment:  monthlyPayment,
		PaymentInterval: interval,
		CreditKind:      CreditKindConsumer,
		IssueDate:       issueDate,
		InterestRate:    interestRate,
	})
}
