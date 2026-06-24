package timeutil

import (
	"fmt"
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

func FormatUTC(t time.Time) string {
	return t.UTC().Format(Layout)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
