package stats

import (
	"testing"
	"time"
)

func TestPercentage(t *testing.T) {
	t.Parallel()
	if got := percentage(0, 10); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
	if got := percentage(200, 25); got != 12.5 {
		t.Fatalf("expected 12.5, got %v", got)
	}
	if got := percentage(3, 1); got != 33.3 {
		t.Fatalf("expected 33.3, got %v", got)
	}
}

func TestPeriodKeyWithTimezone(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("Asia/Vladivostok")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	utc := time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC)
	local := utc.In(loc)
	if got := periodKey(local, "day"); got != "2026-01-02" {
		t.Fatalf("expected local day 2026-01-02, got %s", got)
	}
	if got := periodKey(local, "month"); got != "2026-01-01" {
		t.Fatalf("expected month start 2026-01-01, got %s", got)
	}
	if got := periodKey(time.Date(2026, 1, 4, 12, 0, 0, 0, loc), "week"); got != "2025-12-29" {
		t.Fatalf("expected week start 2025-12-29, got %s", got)
	}
}
