package subscription_test

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
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

func insertExpense(t *testing.T, mgr *db.Manager, userID, accID, txID string, amount int64, at time.Time) {
	t.Helper()
	now := timeutil.FormatUTC(timeutil.NowUTC())
	_, err := mgr.DB().Exec(`
		INSERT INTO transactions (id, user_id, account_id, type, kind, amount, transaction_date, created_at, updated_at)
		VALUES (?, ?, ?, 'expense', 'manual', ?, ?, ?, ?)`,
		txID, userID, accID, amount, timeutil.FormatUTC(at), now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAttachEarlyPaymentAdvancesNextRun(t *testing.T) {
	mgr, userID, accID := setup(t)
	ctx := t.Context()
	day := int64(15)
	// Nearest charge is tomorrow; bank already charged today.
	next := timeutil.NowUTC().UTC().Truncate(time.Hour).Add(24 * time.Hour)
	upcoming := []string{
		timeutil.FormatUTC(next),
		timeutil.FormatUTC(next.AddDate(0, 1, 0)),
		timeutil.FormatUTC(next.AddDate(0, 2, 0)),
	}
	sub, err := subscription.Create(ctx, mgr.DB(), userID, subscription.Input{
		Name: "YandexPlus", Amount: 29900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: next.AddDate(0, -1, 0), TimeLocal: "08:00", Active: true,
		UpcomingRunAts: upcoming,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := append([]string(nil), sub.UpcomingRunAts...)
	txID := "tx-early-1"
	insertExpense(t, mgr, userID, accID, txID, 29900, next.Add(-12*time.Hour))

	res, err := subscription.AttachTransactions(ctx, mgr.DB(), userID, sub.ID, []string{txID})
	if err != nil {
		t.Fatal(err)
	}
	if res.AttachedCount != 1 {
		t.Fatalf("attached %d", res.AttachedCount)
	}
	list, err := subscription.List(ctx, mgr.DB(), userID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	got := list[0]
	if got.NextRunAt != before[1] {
		t.Fatalf("next_run_at after early attach: got %s want %s (upcoming=%v)", got.NextRunAt, before[1], got.UpcomingRunAts)
	}
	if got.LastRunAt == nil || *got.LastRunAt != before[0] {
		t.Fatalf("last_run_at=%v want %s", got.LastRunAt, before[0])
	}
}

func TestAttachHistoricalDoesNotAdvanceNextRun(t *testing.T) {
	mgr, userID, accID := setup(t)
	ctx := t.Context()
	day := int64(15)
	next := timeutil.NowUTC().UTC().Truncate(time.Hour).Add(48 * time.Hour)
	upcoming := []string{
		timeutil.FormatUTC(next),
		timeutil.FormatUTC(next.AddDate(0, 1, 0)),
		timeutil.FormatUTC(next.AddDate(0, 2, 0)),
	}
	sub, err := subscription.Create(ctx, mgr.DB(), userID, subscription.Input{
		Name: "Spotify", Amount: 16900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: next.AddDate(0, -6, 0), TimeLocal: "08:00", Active: true,
		UpcomingRunAts: upcoming,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeNext := sub.NextRunAt
	txID := "tx-hist-1"
	insertExpense(t, mgr, userID, accID, txID, 16900, next.AddDate(0, -3, 0))

	res, err := subscription.AttachTransactions(ctx, mgr.DB(), userID, sub.ID, []string{txID})
	if err != nil {
		t.Fatal(err)
	}
	if res.AttachedCount != 1 {
		t.Fatalf("attached %d", res.AttachedCount)
	}
	list, err := subscription.List(ctx, mgr.DB(), userID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v", err)
	}
	if list[0].NextRunAt != beforeNext {
		t.Fatalf("historical attach must not advance: got %s want %s", list[0].NextRunAt, beforeNext)
	}
	var sid *string
	_ = mgr.DB().QueryRow(`SELECT subscription_id FROM transactions WHERE id = ?`, txID).Scan(&sid)
	if sid == nil || *sid != sub.ID {
		t.Fatalf("expected linked subscription_id, got %v", sid)
	}
}
