package credit

import "math"

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

// RemainingAmount returns principal minus paid amount, floored at zero.
func RemainingAmount(principal, paid int64) int64 {
	r := principal - paid
	if r < 0 {
		return 0
	}
	return r
}
