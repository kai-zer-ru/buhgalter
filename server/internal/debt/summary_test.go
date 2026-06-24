package debt

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestComputeSummary(t *testing.T) {
	now := time.Date(2025, 6, 23, 12, 0, 0, 0, time.UTC)
	rows := []summaryRow{
		{Direction: "borrowed", Amount: 50000, DueDate: "2025-06-20 00:00:00"},
		{Direction: "borrowed", Amount: 30000, DueDate: "2025-07-01 00:00:00"},
		{Direction: "lent", Amount: 100000, DueDate: "2025-06-01 00:00:00"},
	}
	s, err := ComputeSummary(rows, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if s.IOwe != 80000 {
		t.Fatalf("i_owe: want 80000, got %d", s.IOwe)
	}
	if s.OwedToMe != 100000 {
		t.Fatalf("owed_to_me: want 100000, got %d", s.OwedToMe)
	}
	if s.OverdueIOwe != 50000 {
		t.Fatalf("overdue_i_owe: want 50000, got %d", s.OverdueIOwe)
	}
	if s.OverdueOwedToMe != 100000 {
		t.Fatalf("overdue_owed_to_me: want 100000, got %d", s.OverdueOwedToMe)
	}
	if s.ActiveCount != 3 {
		t.Fatalf("active_count: want 3, got %d", s.ActiveCount)
	}
}

func TestOverdueDetectionInUserTZ(t *testing.T) {
	now := time.Date(2025, 6, 23, 12, 0, 0, 0, time.UTC)
	duePast, err := timeutil.ParseUTC("2025-06-20 00:00:00")
	if err != nil {
		t.Fatal(err)
	}
	overdue, err := timeutil.IsOverdueInTZ(duePast, now, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if !overdue {
		t.Fatal("expected overdue in UTC")
	}

	dueToday, _ := timeutil.ParseUTC("2025-06-23 00:00:00")
	overdueToday, err := timeutil.IsOverdueInTZ(dueToday, now, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if overdueToday {
		t.Fatal("due on today should not be overdue")
	}

	// 2025-06-22 22:00 UTC = 2025-06-23 01:00 Moscow — due June 21 is overdue
	nowMoscow := time.Date(2025, 6, 22, 22, 0, 0, 0, time.UTC)
	dueJune21, _ := timeutil.ParseUTC("2025-06-21 12:00:00")
	overdueMoscow, err := timeutil.IsOverdueInTZ(dueJune21, nowMoscow, "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if !overdueMoscow {
		t.Fatal("past due date should be overdue in Moscow TZ")
	}
}
