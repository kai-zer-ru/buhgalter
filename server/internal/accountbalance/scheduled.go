package accountbalance

import (
	"context"
	"database/sql"
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
		sum, err := sumScheduleCharges(schedule.Input{
			Period:     s.Period,
			Weekday:    s.Weekday,
			DayOfMonth: s.DayOfMonth,
			StartDate:  parseScheduleStart(s.StartDate),
			TimeLocal:  s.TimeLocal,
		}, tz, s.NextRunAt, s.Amount, monthStartT, monthEndT, now, true)
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
