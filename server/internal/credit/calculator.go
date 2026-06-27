package credit

import (
	"math"
	"time"
)

// MonthlyPayment calculates the annuity payment in kopecks.
// principal — сумма в копейках; annualRate — годовая ставка в процентах; termMonths — число платежей.
func MonthlyPayment(principal int64, annualRate float64, termMonths int) int64 {
	if termMonths <= 0 || principal <= 0 {
		return 0
	}
	monthlyRate := annualRate / 12 / 100
	if monthlyRate == 0 {
		return principal / int64(termMonths)
	}
	pow := math.Pow(1+monthlyRate, float64(termMonths))
	payment := float64(principal) * (monthlyRate * pow) / (pow - 1)
	return int64(math.Round(payment))
}

// MonthlyPaymentMortgage computes a stable monthly payment for mortgage schedules
// that use daily interest accrual by calendar day (monthly interval).
// It finds the smallest payment that keeps all interim payments valid
// (interest is covered and debt is not closed before the last period).
func MonthlyPaymentMortgage(principal int64, annualRate float64, termMonths int, issueDate time.Time) int64 {
	if termMonths <= 0 || principal <= 0 {
		return 0
	}
	if annualRate <= 0 {
		return principal / int64(termMonths)
	}
	lo := int64(1)
	hi := MonthlyPayment(principal, annualRate, termMonths)
	if hi <= 0 {
		hi = principal / int64(termMonths)
	}
	if hi <= 0 {
		hi = 1
	}
	for mortgageEqualPaymentResidual(principal, annualRate, termMonths, issueDate, hi) > 0 {
		if hi > math.MaxInt64/4 {
			break
		}
		hi *= 2
	}
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if mortgageEqualPaymentResidual(principal, annualRate, termMonths, issueDate, mid) > 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	best := lo + 1
	bestAbs := absInt64(mortgageEqualPaymentResidual(principal, annualRate, termMonths, issueDate, best))
	for _, candidate := range []int64{lo, lo + 1} {
		if candidate <= 0 {
			continue
		}
		if abs := absInt64(mortgageEqualPaymentResidual(principal, annualRate, termMonths, issueDate, candidate)); abs < bestAbs {
			best = candidate
			bestAbs = abs
		}
	}
	return best
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func mortgagePeriodInterestFloat(balance float64, annualRate float64, from, to time.Time) int64 {
	days := int(to.Sub(from).Hours() / 24)
	if days < 1 {
		days = 1
	}
	yearDays := 365.0
	if to.Year()%4 == 0 && (to.Year()%100 != 0 || to.Year()%400 == 0) {
		yearDays = 366.0
	}
	dailyRate := annualRate / 100 / yearDays
	return int64(balance * dailyRate * float64(days))
}

// RemainingAmount returns principal minus paid amount, floored at zero.
func RemainingAmount(principal, paid int64) int64 {
	r := principal - paid
	if r < 0 {
		return 0
	}
	return r
}
