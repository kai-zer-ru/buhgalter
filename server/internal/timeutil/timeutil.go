package timeutil

import (
	"fmt"
	"strings"
	"time"
)

const Layout = "2006-01-02 15:04:05"

func ParseUTC(s string) (time.Time, error) {
	t, err := time.ParseInLocation(Layout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid datetime %q: expected format YYYY-MM-DD HH:MM:SS", s)
	}
	return t, nil
}

// ParseFlexibleUTC parses API datetimes and RFC3339 timestamps stored in the DB.
func ParseFlexibleUTC(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := ParseUTC(s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid datetime %q", s)
}

func FormatUTC(t time.Time) string {
	return t.UTC().Format(Layout)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
