package stats

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestMonthEndInTZCapsAtNow(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("Europe/Moscow")
	monthStart := time.Date(2026, 3, 1, 0, 0, 0, 0, loc)
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	got := monthEndInTZ(monthStart, loc, now)
	want := timeutil.FormatUTC(now)
	if got != want {
		t.Fatalf("expected now %s, got %s", want, got)
	}
}

func TestMonthEndInTZPastMonth(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("Europe/Moscow")
	monthStart := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	got := monthEndInTZ(monthStart, loc, now)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, loc)
	want := timeutil.FormatUTC(end.UTC())
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestMonthCountCap(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("UTC")
	from := time.Date(2010, 1, 1, 0, 0, 0, 0, loc)
	to := time.Date(2020, 1, 1, 0, 0, 0, 0, loc)
	if monthCount(from, to) != 121 {
		t.Fatalf("want 121 months, got %d", monthCount(from, to))
	}
}

func TestNetWorthMonthRangeDefaults(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("Europe/Moscow")
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, loc)
	from, to, err := netWorthMonthRange("", "", now.UTC(), loc)
	if err != nil {
		t.Fatal(err)
	}
	if to.Year() != 2026 || to.Month() != time.June {
		t.Fatalf("end month: %v", to)
	}
	if from.Year() != 2025 || from.Month() != time.July {
		t.Fatalf("start month: %v", from)
	}
}
