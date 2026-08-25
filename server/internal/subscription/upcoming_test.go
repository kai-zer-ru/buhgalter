package subscription_test

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/schedule"
	"github.com/kai-zer-ru/buhgalter/internal/subscription"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestSeedUpcomingCount(t *testing.T) {
	day := int64(15)
	in := schedule.Input{
		Period: "month", DayOfMonth: &day,
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeLocal: "08:00",
	}
	dates, err := subscription.SeedUpcoming(in, "UTC", time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := subscription.ValidateUpcoming(dates); err != nil {
		t.Fatal(err)
	}
	if len(dates) != subscription.UpcomingCount {
		t.Fatalf("len %d", len(dates))
	}
	t0, _ := timeutil.ParseUTC(dates[0])
	t1, _ := timeutil.ParseUTC(dates[1])
	if !t1.After(t0) {
		t.Fatalf("not ascending: %v", dates)
	}
}

func TestAdvanceUpcomingLearnedInterval(t *testing.T) {
	// 28-day spacing must persist after advance.
	base := time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC)
	current := []string{
		timeutil.FormatUTC(base),
		timeutil.FormatUTC(base.AddDate(0, 0, 28)),
		timeutil.FormatUTC(base.AddDate(0, 0, 56)),
	}
	day := int64(1)
	in := schedule.Input{
		Period: "month", DayOfMonth: &day,
		StartDate: base, TimeLocal: "08:00",
	}
	next, err := subscription.AdvanceUpcoming(current, in, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if next[0] != current[1] || next[1] != current[2] {
		t.Fatalf("shift failed: %v", next)
	}
	got, err := timeutil.ParseUTC(next[2])
	if err != nil {
		t.Fatal(err)
	}
	want := base.AddDate(0, 0, 84)
	if !got.Equal(want) {
		t.Fatalf("learned interval: got %v want %v", got, want)
	}
}

func TestCreateSeedsUpcoming(t *testing.T) {
	mgr, userID, accID := setup(t)
	day := int64(1)
	sub, err := subscription.Create(t.Context(), mgr.DB(), userID, subscription.Input{
		Name: "Spotify", Amount: 16900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeLocal: "08:00", Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := subscription.ValidateUpcoming(sub.UpcomingRunAts); err != nil {
		t.Fatalf("upcoming: %v %v", sub.UpcomingRunAts, err)
	}
	if sub.NextRunAt != sub.UpcomingRunAts[0] {
		t.Fatalf("next_run_at %s != upcoming[0] %s", sub.NextRunAt, sub.UpcomingRunAts[0])
	}
}

func TestApplyDueAdvancesUpcomingJSON(t *testing.T) {
	mgr, userID, accID := setup(t)
	ctx := t.Context()
	day := int64(1)
	base := time.Date(2020, 1, 1, 8, 0, 0, 0, time.UTC)
	upcoming := []string{
		timeutil.FormatUTC(base),
		timeutil.FormatUTC(base.AddDate(0, 0, 28)),
		timeutil.FormatUTC(base.AddDate(0, 0, 56)),
	}
	sub, err := subscription.Create(ctx, mgr.DB(), userID, subscription.Input{
		Name: "Netflix28", Amount: 99900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: base, TimeLocal: "08:00", Active: true,
		UpcomingRunAts: upcoming,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := subscription.ValidateUpcoming(sub.UpcomingRunAts); err != nil {
		t.Fatal(err)
	}
	before := append([]string(nil), sub.UpcomingRunAts...)
	n, err := subscription.ApplyDue(ctx, mgr.DB(), userID, timeutil.NowUTC(), "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("applied %d", n)
	}
	got, err := subscription.List(ctx, mgr.DB(), userID)
	if err != nil || len(got) != 1 {
		t.Fatalf("list: %+v %v", got, err)
	}
	if got[0].NextRunAt != before[1] {
		t.Fatalf("next after apply: %s want %s (upcoming=%v)", got[0].NextRunAt, before[1], got[0].UpcomingRunAts)
	}
	prev, _ := timeutil.ParseUTC(before[1])
	last, _ := timeutil.ParseUTC(before[2])
	third, _ := timeutil.ParseUTC(got[0].UpcomingRunAts[2])
	if !third.Equal(last.Add(last.Sub(prev))) {
		t.Fatalf("third=%v want %v (from %v %v)", third, last.Add(last.Sub(prev)), prev, last)
	}
}
