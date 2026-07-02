package timeutil

import (
	"fmt"
	"time"
)

// MonthBoundsForMonthUTC returns UTC datetime strings for [start, endExclusive) of the given
// calendar month in the IANA timezone. month is 1–12.
func MonthBoundsForMonthUTC(tz string, year int, month time.Month) (start, endExclusive string, err error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", "", fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEndExclusive := monthStart.AddDate(0, 1, 0)
	return FormatUTC(monthStart.UTC()), FormatUTC(monthEndExclusive.UTC()), nil
}

// MonthBoundsUTC returns UTC datetime strings for the start and end of the current month
// in the given IANA timezone.
func MonthBoundsUTC(tz string, now time.Time) (start, end string, err error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", "", fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	inTZ := now.In(loc)
	year, month, _ := inTZ.Date()
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
	return FormatUTC(monthStart.UTC()), FormatUTC(monthEnd.UTC()), nil
}

// TodayStartUTC returns UTC datetime for 00:00:00 of the current calendar day in tz.
func TodayStartUTC(tz string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	inTZ := now.In(loc)
	year, month, day := inTZ.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc).UTC(), nil
}

// IsOverdueInTZ reports whether dueDate (UTC) is before today's calendar date in tz.
func IsOverdueInTZ(dueDate, now time.Time, tz string) (bool, error) {
	todayStart, err := TodayStartUTC(tz, now)
	if err != nil {
		return false, err
	}
	return dueDate.Before(todayStart), nil
}

// IsFutureInTZ reports whether txDate (UTC) is after now when both are viewed in tz.
func IsFutureInTZ(txDate, now time.Time, tz string) (bool, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return false, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return txDate.In(loc).After(now.In(loc)), nil
}
