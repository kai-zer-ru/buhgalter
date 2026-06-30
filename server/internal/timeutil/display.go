package timeutil

import "time"

// User-facing display layouts (change here to update the whole project).
// See docs/date-time-display.md
const (
	// DisplayDateLayout — date only, e.g. 31.12.2026
	DisplayDateLayout = "02.01.2006"
	// DisplayDateTimeLayout — date and time with seconds, e.g. 31.12.2026 12:00:00
	DisplayDateTimeLayout = "02.01.2006 15:04:05"
	// DisplayDateTimeShortLayout — date and time without seconds (operation lists), e.g. 31.12.2026 12:00
	DisplayDateTimeShortLayout = "02.01.2006 15:04"
)

func displayLocation(timezone string) (*time.Location, error) {
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	return time.LoadLocation(timezone)
}

func formatDisplayInTimezone(value, timezone, layout string) string {
	tm, err := ParseFlexibleUTC(value)
	if err != nil {
		return value
	}
	loc, err := displayLocation(timezone)
	if err != nil {
		return value
	}
	return tm.In(loc).Format(layout)
}

// FormatDisplayDateInTimezone formats API UTC datetime as date in user timezone.
func FormatDisplayDateInTimezone(value, timezone string) string {
	return formatDisplayInTimezone(value, timezone, DisplayDateLayout)
}

// FormatDisplayDateTimeInTimezone formats API UTC datetime with seconds in user timezone.
func FormatDisplayDateTimeInTimezone(value, timezone string) string {
	return formatDisplayInTimezone(value, timezone, DisplayDateTimeLayout)
}

// FormatDisplayDateTimeShortInTimezone formats API UTC datetime without seconds
// (operation lists, notification date-time placeholders).
func FormatDisplayDateTimeShortInTimezone(value, timezone string) string {
	return formatDisplayInTimezone(value, timezone, DisplayDateTimeShortLayout)
}
