package accountbalance

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/schedule"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const maxScheduledOccurrences = 64

// ScheduledEffects is the per-account forecast contribution from active
// subscriptions and recurring operations in the current calendar month.
type ScheduledEffects struct {
	Deltas          map[string]int64
	HasSubscription map[string]struct{}
	HasRecurring    map[string]struct{}
}

// ScheduledEffectsByUser sums signed balance deltas from active subscriptions
// and recurring operations that still run in the current calendar month (user TZ).
// Positive = income, negative = expense.
func ScheduledEffectsByUser(ctx context.Context, db *sql.DB, userID, tz string, now time.Time) (ScheduledEffects, error) {
	out := ScheduledEffects{
		Deltas:          make(map[string]int64),
		HasSubscription: make(map[string]struct{}),
		HasRecurring:    make(map[string]struct{}),
	}
	monthStart, monthEnd, err := timeutil.MonthBoundsUTC(tz, now)
	if err != nil {
		return out, err
	}
	monthStartT, err := timeutil.ParseUTC(monthStart)
	if err != nil {
		return out, err
	}
	monthEndT, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		return out, err
	}

	q := queries(db)

	subs, err := q.ListActiveSubscriptionsForForecast(ctx, sqlcdb.ListActiveSubscriptionsForForecastParams{
		UserID: userID, NextRunAt: monthEnd,
	})
	if err != nil {
		return out, err
	}
	for _, s := range subs {
		in := schedule.Input{
			Period:     s.Period,
			Weekday:    s.Weekday,
			DayOfMonth: s.DayOfMonth,
			StartDate:  parseScheduleStart(s.StartDate),
			TimeLocal:  s.TimeLocal,
		}
		sum, err := sumSubscriptionCharges(in, tz, s.NextRunAt, s.UpcomingRunAts, s.Amount, monthStartT, monthEndT, now)
		if err != nil {
			continue
		}
		if sum == 0 {
			continue
		}
		out.Deltas[s.AccountID] -= sum
		out.HasSubscription[s.AccountID] = struct{}{}
	}

	recs, err := q.ListActiveRecurringForForecast(ctx, sqlcdb.ListActiveRecurringForForecastParams{
		UserID: userID, NextRunAt: monthEnd,
	})
	if err != nil {
		return out, err
	}
	for _, r := range recs {
		sum, err := sumScheduleCharges(schedule.Input{
			Period:     r.Period,
			Weekday:    r.Weekday,
			DayOfMonth: r.DayOfMonth,
			StartDate:  parseScheduleStart(r.StartDate),
			TimeLocal:  r.TimeLocal,
		}, tz, r.NextRunAt, r.Amount, monthStartT, monthEndT, now, false)
		if err != nil {
			continue
		}
		if sum == 0 {
			continue
		}
		switch r.Type {
		case "income":
			out.Deltas[r.AccountID] += sum
		default:
			out.Deltas[r.AccountID] -= sum
		}
		out.HasRecurring[r.AccountID] = struct{}{}
	}

	return out, nil
}

func parseScheduleStart(v string) time.Time {
	t, err := timeutil.ParseUTC(v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func sumSubscriptionCharges(
	in schedule.Input,
	tz, nextRunAt, upcomingJSON string,
	amount int64,
	monthStart, monthEnd, now time.Time,
) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	if err := schedule.ValidatePeriodFields(in, true); err != nil {
		return 0, err
	}
	queue := decodeUpcomingJSON(upcomingJSON)
	if len(queue) == 0 {
		return sumScheduleCharges(in, tz, nextRunAt, amount, monthStart, monthEnd, now, true)
	}
	var total int64
	var prev, last time.Time
	for i, raw := range queue {
		t, err := timeutil.ParseUTC(raw)
		if err != nil {
			continue
		}
		if t.After(monthEnd) {
			return total, nil
		}
		if !t.Before(monthStart) || !t.After(now) {
			total += amount
		}
		if i == len(queue)-2 {
			prev = t
		}
		if i == len(queue)-1 {
			last = t
		}
	}
	if last.IsZero() {
		return total, nil
	}
	// Continue past stored queue (e.g. weekly within month) via learned interval or period.
	t := last
	useDelta := !prev.IsZero() && last.After(prev)
	delta := last.Sub(prev)
	for i := 0; i < maxScheduledOccurrences; i++ {
		var nextT time.Time
		if useDelta {
			nextT = t.Add(delta)
		} else {
			nextStr, err := schedule.NextRunAt(in, tz, t.Add(time.Second))
			if err != nil {
				return total, err
			}
			nextT, err = timeutil.ParseUTC(nextStr)
			if err != nil {
				return total, err
			}
			if !nextT.After(t) {
				break
			}
		}
		if nextT.After(monthEnd) {
			break
		}
		if !nextT.Before(monthStart) || !nextT.After(now) {
			total += amount
		}
		t = nextT
	}
	return total, nil
}

func decodeUpcomingJSON(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var dates []string
	if err := json.Unmarshal([]byte(raw), &dates); err != nil || len(dates) == 0 {
		return nil
	}
	return dates
}

func sumScheduleCharges(
	in schedule.Input,
	tz, nextRunAt string,
	amount int64,
	monthStart, monthEnd, now time.Time,
	allowExtended bool,
) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	if err := schedule.ValidatePeriodFields(in, allowExtended); err != nil {
		return 0, err
	}
	t, err := timeutil.ParseUTC(nextRunAt)
	if err != nil {
		return 0, err
	}
	var total int64
	for i := 0; i < maxScheduledOccurrences; i++ {
		if t.After(monthEnd) {
			break
		}
		if !t.Before(monthStart) || !t.After(now) {
			total += amount
		}
		nextStr, err := schedule.NextRunAt(in, tz, t.Add(time.Second))
		if err != nil {
			return total, err
		}
		nextT, err := timeutil.ParseUTC(nextStr)
		if err != nil {
			return total, err
		}
		if !nextT.After(t) {
			break
		}
		t = nextT
	}
	return total, nil
}
