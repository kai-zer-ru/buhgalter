package timeutil

import (
	"testing"
	"time"
)

func TestFormatDisplayInTimezone(t *testing.T) {
	const tz = "UTC"
	const api = "2026-12-31 09:30:45"

	if got := FormatDisplayDateInTimezone(api, tz); got != "31.12.2026" {
		t.Fatalf("date: got %q", got)
	}
	if got := FormatDisplayDateTimeInTimezone(api, tz); got != "31.12.2026 09:30:45" {
		t.Fatalf("datetime: got %q", got)
	}
	if got := FormatDisplayDateTimeShortInTimezone(api, tz); got != "31.12.2026 09:30" {
		t.Fatalf("datetime short: got %q", got)
	}
}

func TestFormatDisplayRFC3339(t *testing.T) {
	const tz = "Europe/Moscow"
	const rfc = "2026-06-30T08:13:42Z"
	got := FormatDisplayDateTimeShortInTimezone(rfc, tz)
	// MSK = UTC+3
	if got != "30.06.2026 11:13" {
		t.Fatalf("got %q", got)
	}
}

func TestParseFlexibleUTC(t *testing.T) {
	if _, err := ParseFlexibleUTC("2026-06-30 08:13:42"); err != nil {
		t.Fatal(err)
	}
	tm, err := ParseFlexibleUTC("2026-06-30T08:13:42Z")
	if err != nil {
		t.Fatal(err)
	}
	if !tm.Equal(time.Date(2026, 6, 30, 8, 13, 42, 0, time.UTC)) {
		t.Fatalf("got %v", tm)
	}
}
