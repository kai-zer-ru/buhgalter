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

// ScheduledEffectsByUser sums signed balance deltas from active subscriptions
// and recurring operations that still run in the current calendar month (user TZ).
// Positive = income, negative = expense. Also returns accounts with any contribution.
func ScheduledEffectsByUser(ctx context.Context, db *sql.DB, userID, tz string, now time.Time) (deltas map[string]int64, has map[string]struct{}, err error) {
	monthStart, monthEnd, err := timeutil.MonthBoundsUTC(tz, now)
	if err != nil {
		return nil, nil, err
	}
	monthStartT, err := timeutil.ParseUTC(monthStart)
	if err != nil {
		return nil, nil, err
	}
	monthEndT, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		return nil, nil, err
	}

	q := queries(db)
	deltas = make(map[string]int64)
	has = make(map[string]struct{})

	subs, err := q.ListActiveSubscriptionsForForecast(ctx, sqlcdb.ListActiveSubscriptionsForForecastParams{
		UserID: userID, NextRunAt: monthEnd,
	})
	if err != nil {
		return nil, nil, err
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
		deltas[s.AccountID] -= sum
		has[s.AccountID] = struct{}{}
	}

	recs, err := q.ListActiveRecurringForForecast(ctx, sqlcdb.ListActiveRecurringForForecastParams{
		UserID: userID, NextRunAt: monthEnd,
	})
	if err != nil {
		return nil, nil, err
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
			deltas[r.AccountID] += sum
		default:
			deltas[r.AccountID] -= sum
		}
		has[r.AccountID] = struct{}{}
	}

	return deltas, has, nil
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
