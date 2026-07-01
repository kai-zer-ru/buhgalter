package budget

import (
	"fmt"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func parseMonth(month string) (year int, mon time.Month, err error) {
	if month == "" {
		now := timeutil.NowUTC()
		return now.Year(), now.Month(), nil
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month")
	}
	return t.Year(), t.Month(), nil
}

func monthBoundsExclusive(tz string, year int, month time.Month) (start, endExclusive string, err error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", "", fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEndExclusive := monthStart.AddDate(0, 1, 0)
	return timeutil.FormatUTC(monthStart.UTC()), timeutil.FormatUTC(monthEndExclusive.UTC()), nil
}

func monthQueryValue(year int, month time.Month) string {
	return fmt.Sprintf("%04d-%02d", year, int(month))
}

func AddMonths(month string, delta int) (string, error) {
	return addMonths(month, delta)
}

func addMonths(month string, delta int) (string, error) {
	year, mon, err := parseMonth(month)
	if err != nil {
		return "", ErrInvalidMonth
	}
	t := time.Date(year, mon, 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0)
	return monthQueryValue(t.Year(), t.Month()), nil
}
