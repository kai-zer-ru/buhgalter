package credit

import (
	"errors"
	"math"
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
	Principal         int64
	TermMonths        int
	MonthlyPayment    int64
	UserSetPayment    bool
	PaymentInterval   PaymentInterval
	CreditKind        string
	IssueDate         time.Time
	InterestRate      float64
	SeedPayments      []ScheduleSeed
	FirstPaymentToday bool
}

var ErrMonthlyPaymentTooLowForInterest = errors.New("monthly payment is too low for interest accrual")
var ErrMonthlyPaymentTooHighForTerm = errors.New("monthly payment is too high for selected term")

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
		return nil, ErrInvalidTerm
	}
	if !in.PaymentInterval.Valid() {
		return nil, ErrInvalidInterval
	}
	if in.PaymentInterval == IntervalManual {
		return generateManualSchedule(in)
	}
	if in.MonthlyPayment <= 0 {
		return nil, ErrInvalidAmount
	}

	if in.InterestRate > 0 {
		if len(in.SeedPayments) == in.TermMonths {
			entries := make([]ScheduleEntry, len(in.SeedPayments))
			for i, seed := range in.SeedPayments {
				if seed.Amount <= 0 {
					return nil, ErrInvalidAmount
				}
				entries[i] = ScheduleEntry{
					PaymentDate: timeutil.FormatUTC(seed.PaymentDate),
					Amount:      seed.Amount,
				}
			}
			return entries, nil
		}
		if normalizeCreditKind(in.CreditKind) == CreditKindMortgage && in.PaymentInterval == IntervalMonth {
			if in.UserSetPayment {
				return generateMortgageUserSetSchedule(in)
			}
			return generateMortgageInterestSchedule(in)
		}
		return generateInterestSchedule(in)
	}

	entries := make([]ScheduleEntry, 0, in.TermMonths)

	for _, seed := range in.SeedPayments {
		if seed.Amount <= 0 {
			return nil, ErrInvalidAmount
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
		if len(entries) == 0 && in.FirstPaymentToday {
			entries = append(entries, ScheduleEntry{
				PaymentDate: timeutil.FormatUTC(in.IssueDate),
				Amount:      in.MonthlyPayment,
			})
			continue
		}
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
		if i == 0 && in.FirstPaymentToday {
			dates = append(dates, in.IssueDate)
			lastDate = in.IssueDate
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

	for i := 0; i < in.TermMonths; i++ {
		interest := int64(balance * rate) // truncate kopecks, bank-style
		var amount int64
		if i == in.TermMonths-1 {
			balanceK := int64(balance)
			amount = balanceK + interest
			balance = 0
		} else {
			amount = in.MonthlyPayment
			principalPart := amount - interest
			if principalPart < 0 {
				return nil, ErrMonthlyPaymentTooLowForInterest
			}
			if in.UserSetPayment && i < in.TermMonths-1 && principalPart >= int64(balance) {
				return nil, ErrMonthlyPaymentTooHighForTerm
			}
			balance -= float64(principalPart)
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(dates[i]),
			Amount:      amount,
		})
	}
	return entries, nil
}

func mortgageEqualPaymentResidual(principal int64, annualRate float64, termMonths int, issueDate time.Time, payment int64) int64 {
	dates, err := schedulePaymentDates(ScheduleInput{
		TermMonths:      termMonths,
		IssueDate:       issueDate,
		PaymentInterval: IntervalMonth,
	})
	if err != nil {
		return principal
	}
	balance := float64(principal)
	prevDate := issueDate
	for i := 0; i < termMonths; i++ {
		interest := mortgagePeriodInterestFloat(balance, annualRate, prevDate, dates[i])
		balance = balance + float64(interest) - float64(payment)
		prevDate = dates[i]
	}
	return int64(math.Round(balance))
}

// generateMortgageInterestSchedule builds a mortgage schedule with daily interest accrual.
// Payments 1..N-1 are equal; the last payment clears remaining principal plus period interest.
func generateMortgageInterestSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	dates, err := schedulePaymentDates(in)
	if err != nil {
		return nil, err
	}
	balance := float64(in.Principal)
	prevDate := in.IssueDate
	entries := make([]ScheduleEntry, 0, in.TermMonths)

	for i := 0; i < in.TermMonths-1; i++ {
		interest := mortgagePeriodInterestFloat(balance, in.InterestRate, prevDate, dates[i])
		amount := in.MonthlyPayment
		balance = balance + float64(interest) - float64(amount)
		if amount <= 0 {
			return nil, ErrInvalidAmount
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(dates[i]),
			Amount:      amount,
		})
		prevDate = dates[i]
	}

	interest := mortgagePeriodInterestFloat(balance, in.InterestRate, prevDate, dates[in.TermMonths-1])
	remaining := int64(math.Round(balance))
	if remaining < 0 {
		return nil, ErrMonthlyPaymentTooHighForTerm
	}
	amount := remaining + interest
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	entries = append(entries, ScheduleEntry{
		PaymentDate: timeutil.FormatUTC(dates[in.TermMonths-1]),
		Amount:      amount,
	})
	return entries, nil
}

// generateMortgageUserSetSchedule builds a mortgage schedule from a user-provided monthly payment
// (e.g. copied from a bank contract). Interim rows use the entered amount; the last row closes
// any positive remainder or keeps the same amount when the bank payment exceeds model payoff.
func generateMortgageUserSetSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	dates, err := schedulePaymentDates(in)
	if err != nil {
		return nil, err
	}
	balance := float64(in.Principal)
	prevDate := in.IssueDate
	entries := make([]ScheduleEntry, 0, in.TermMonths)

	for i := 0; i < in.TermMonths-1; i++ {
		interest := mortgagePeriodInterestFloat(balance, in.InterestRate, prevDate, dates[i])
		amount := in.MonthlyPayment
		balance = balance + float64(interest) - float64(amount)
		if amount <= 0 {
			return nil, ErrInvalidAmount
		}
		entries = append(entries, ScheduleEntry{
			PaymentDate: timeutil.FormatUTC(dates[i]),
			Amount:      amount,
		})
		prevDate = dates[i]
	}

	interest := mortgagePeriodInterestFloat(balance, in.InterestRate, prevDate, dates[in.TermMonths-1])
	remaining := int64(math.Round(balance))
	var amount int64
	if remaining > 0 {
		amount = remaining + interest
	} else {
		amount = in.MonthlyPayment
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	entries = append(entries, ScheduleEntry{
		PaymentDate: timeutil.FormatUTC(dates[in.TermMonths-1]),
		Amount:      amount,
	})
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
		return ErrInvalidAmount
	}
	return nil
}

func generateManualSchedule(in ScheduleInput) ([]ScheduleEntry, error) {
	if len(in.SeedPayments) != in.TermMonths {
		return nil, ErrInvalidTerm
	}
	entries := make([]ScheduleEntry, 0, len(in.SeedPayments))
	for _, seed := range in.SeedPayments {
		if seed.Amount <= 0 {
			return nil, ErrInvalidAmount
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
