package schedule

import (
	"errors"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var (
	ErrInvalidPeriod  = errors.New("invalid period")
	ErrInvalidWeekday = errors.New("invalid weekday")
	ErrInvalidDay     = errors.New("invalid day")
	ErrInvalidTime    = errors.New("invalid time")
)

// Input holds schedule fields shared by recurring operations and subscriptions.
type Input struct {
	Period     string
	Weekday    *int64
	DayOfMonth *int64
	StartDate  time.Time
	TimeLocal  string
}

func ValidatePeriodFields(in Input, allowExtended bool) error {
	switch in.Period {
	case "week", "two_weeks":
		if in.Weekday == nil || *in.Weekday < 1 || *in.Weekday > 7 {
			return ErrInvalidWeekday
		}
	case "month", "year":
		if in.DayOfMonth == nil || *in.DayOfMonth < 1 || *in.DayOfMonth > 31 {
			return ErrInvalidDay
		}
	case "quarter", "half_year":
		if !allowExtended {
			return ErrInvalidPeriod
		}
		if in.DayOfMonth == nil || *in.DayOfMonth < 1 || *in.DayOfMonth > 31 {
			return ErrInvalidDay
		}
	default:
		return ErrInvalidPeriod
	}
	if _, _, err := ParseLocalTime(in.TimeLocal); err != nil {
		return ErrInvalidTime
	}
	return nil
}

func NextRunAt(in Input, tz string, ref time.Time) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	hour, minute, err := ParseLocalTime(in.TimeLocal)
	if err != nil {
		return "", err
	}
	threshold := ref.In(loc)
	startLocal := in.StartDate.In(loc)
	startPoint := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), hour, minute, 0, 0, loc)
	if threshold.Before(startPoint) {
		threshold = startPoint
	}

	var candidate time.Time
	switch in.Period {
	case "week":
		candidate = nextWeekdayAtOrAfter(threshold, int(*in.Weekday), hour, minute)
	case "two_weeks":
		first := nextWeekdayAtOrAfter(startPoint, int(*in.Weekday), hour, minute)
		candidate = first
		for candidate.Before(threshold) {
			candidate = candidate.AddDate(0, 0, 14)
		}
	case "month":
		candidate = nextMonthDayAtOrAfter(threshold, int(*in.DayOfMonth), hour, minute)
	case "quarter":
		candidate = nextNMonthDayAtOrAfter(threshold, startPoint, int(*in.DayOfMonth), hour, minute, 3)
	case "half_year":
		candidate = nextNMonthDayAtOrAfter(threshold, startPoint, int(*in.DayOfMonth), hour, minute, 6)
	case "year":
		candidate = nextYearDayAtOrAfter(threshold, startPoint.Month(), int(*in.DayOfMonth), hour, minute)
	default:
		return "", ErrInvalidPeriod
	}
	return timeutil.FormatUTC(candidate.UTC()), nil
}

func ParseLocalTime(v string) (int, int, error) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

func nextWeekdayAtOrAfter(base time.Time, weekday, hour, minute int) time.Time {
	target := toGoWeekday(weekday)
	candidate := time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
	diff := (int(target) - int(candidate.Weekday()) + 7) % 7
	candidate = candidate.AddDate(0, 0, diff)
	if candidate.Before(base) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

func nextMonthDayAtOrAfter(base time.Time, day, hour, minute int) time.Time {
	y, m, _ := base.Date()
	candidate := monthDayAt(base.Location(), y, m, day, hour, minute)
	if candidate.Before(base) {
		candidate = monthDayAt(base.Location(), y, m+1, day, hour, minute)
	}
	return candidate
}

func nextNMonthDayAtOrAfter(base, startPoint time.Time, day, hour, minute, everyMonths int) time.Time {
	candidate := monthDayAt(startPoint.Location(), startPoint.Year(), startPoint.Month(), day, hour, minute)
	if candidate.Before(startPoint) {
		candidate = monthDayAt(startPoint.Location(), startPoint.Year(), startPoint.Month()+time.Month(everyMonths), day, hour, minute)
	}
	for candidate.Before(base) {
		y, m, _ := candidate.Date()
		candidate = monthDayAt(candidate.Location(), y, m+time.Month(everyMonths), day, hour, minute)
	}
	return candidate
}

func nextYearDayAtOrAfter(base time.Time, anchorMonth time.Month, day, hour, minute int) time.Time {
	y := base.Year()
	candidate := monthDayAt(base.Location(), y, anchorMonth, day, hour, minute)
	if candidate.Before(base) {
		candidate = monthDayAt(base.Location(), y+1, anchorMonth, day, hour, minute)
	}
	return candidate
}

func monthDayAt(loc *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	for month > 12 {
		year++
		month -= 12
	}
	for month < 1 {
		year--
		month += 12
	}
	maxDay := daysInMonth(year, month)
	if day > maxDay {
		day = maxDay
	}
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func toGoWeekday(v int) time.Weekday {
	switch v {
	case 1:
		return time.Monday
	case 2:
		return time.Tuesday
	case 3:
		return time.Wednesday
	case 4:
		return time.Thursday
	case 5:
		return time.Friday
	case 6:
		return time.Saturday
	default:
		return time.Sunday
	}
}
