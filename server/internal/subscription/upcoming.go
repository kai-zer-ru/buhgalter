package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/schedule"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const UpcomingCount = 3

var (
	ErrInvalidUpcoming = errors.New("invalid upcoming_run_ats")
)

func init() {
	db.RegisterOpenHook(BackfillUpcomingRunAts)
}

// SeedUpcoming returns UpcomingCount successive run-at times from schedule rules.
func SeedUpcoming(in schedule.Input, tz string, ref time.Time) ([]string, error) {
	out := make([]string, 0, UpcomingCount)
	cur := ref
	for i := 0; i < UpcomingCount; i++ {
		next, err := schedule.NextRunAt(in, tz, cur)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
		t, err := timeutil.ParseUTC(next)
		if err != nil {
			return nil, err
		}
		cur = t.Add(time.Second)
	}
	return out, nil
}

// ValidateUpcoming checks exactly UpcomingCount strictly ascending UTC datetimes.
func ValidateUpcoming(dates []string) error {
	if len(dates) != UpcomingCount {
		return ErrInvalidUpcoming
	}
	var prev time.Time
	for i, raw := range dates {
		t, err := timeutil.ParseUTC(strings.TrimSpace(raw))
		if err != nil {
			return ErrInvalidUpcoming
		}
		if i > 0 && !t.After(prev) {
			return ErrInvalidUpcoming
		}
		prev = t
	}
	return nil
}

// NormalizeUpcoming applies time_local in tz to each date (calendar day preserved).
func NormalizeUpcoming(dates []string, timeLocal, tz string) ([]string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	hour, minute, err := schedule.ParseLocalTime(timeLocal)
	if err != nil {
		return nil, ErrInvalidTime
	}
	out := make([]string, len(dates))
	for i, raw := range dates {
		t, err := timeutil.ParseUTC(strings.TrimSpace(raw))
		if err != nil {
			return nil, ErrInvalidUpcoming
		}
		local := t.In(loc)
		normalized := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)
		out[i] = timeutil.FormatUTC(normalized.UTC())
	}
	return out, nil
}

// AdvanceUpcoming pops the first date and appends a new third using learned interval
// (last-prev) or schedule.NextRunAt fallback.
func AdvanceUpcoming(current []string, in schedule.Input, tz string) ([]string, error) {
	if err := ValidateUpcoming(current); err != nil {
		return nil, err
	}
	remaining := []string{current[1], current[2]}
	third, err := appendNextUpcoming(remaining[0], remaining[1], in, tz)
	if err != nil {
		return nil, err
	}
	out := []string{remaining[0], remaining[1], third}
	if err := ValidateUpcoming(out); err != nil {
		return nil, err
	}
	return out, nil
}

func appendNextUpcoming(prev, last string, in schedule.Input, tz string) (string, error) {
	prevT, err := timeutil.ParseUTC(prev)
	if err != nil {
		return "", ErrInvalidUpcoming
	}
	lastT, err := timeutil.ParseUTC(last)
	if err != nil {
		return "", ErrInvalidUpcoming
	}
	delta := lastT.Sub(prevT)
	if delta > 0 {
		return timeutil.FormatUTC(lastT.Add(delta)), nil
	}
	return schedule.NextRunAt(in, tz, lastT.Add(time.Second))
}

func encodeUpcoming(dates []string) (string, error) {
	if err := ValidateUpcoming(dates); err != nil {
		return "", err
	}
	b, err := json.Marshal(dates)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeUpcoming(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, ErrInvalidUpcoming
	}
	var dates []string
	if err := json.Unmarshal([]byte(raw), &dates); err != nil {
		return nil, ErrInvalidUpcoming
	}
	if err := ValidateUpcoming(dates); err != nil {
		return nil, err
	}
	return dates, nil
}

func resolveUpcoming(in Input, tz string, ref time.Time) ([]string, string, error) {
	sched := schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}
	var dates []string
	var err error
	if len(in.UpcomingRunAts) > 0 {
		dates, err = NormalizeUpcoming(in.UpcomingRunAts, in.TimeLocal, tz)
		if err != nil {
			return nil, "", err
		}
		if err := ValidateUpcoming(dates); err != nil {
			return nil, "", err
		}
	} else {
		dates, err = SeedUpcoming(sched, tz, ref)
		if err != nil {
			return nil, "", err
		}
	}
	encoded, err := encodeUpcoming(dates)
	if err != nil {
		return nil, "", err
	}
	return dates, encoded, nil
}

// BackfillUpcomingRunAts fills empty upcoming_run_ats for existing rows (migration 051).
func BackfillUpcomingRunAts(ctx context.Context, database *sql.DB) error {
	q := queries(database)
	rows, err := q.ListSubscriptionsForUpcomingBackfill(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	now := timeutil.FormatUTC(timeutil.NowUTC())
	for _, row := range rows {
		tz, err := userTimezone(ctx, database, row.UserID)
		if err != nil {
			tz = "Europe/Moscow"
		}
		sched := schedule.Input{
			Period: row.Period, Weekday: row.Weekday, DayOfMonth: row.DayOfMonth,
			StartDate: mustParse(row.StartDate), TimeLocal: row.TimeLocal,
		}
		dates, err := seedFromStoredNext(sched, tz, row.NextRunAt)
		if err != nil {
			dates, err = SeedUpcoming(sched, tz, timeutil.NowUTC())
			if err != nil {
				continue
			}
		}
		encoded, err := encodeUpcoming(dates)
		if err != nil {
			continue
		}
		_, _ = q.SetSubscriptionUpcoming(ctx, sqlcdb.SetSubscriptionUpcomingParams{
			NextRunAt: dates[0], UpcomingRunAts: encoded, UpdatedAt: now, ID: row.ID,
		})
	}
	return nil
}

func seedFromStoredNext(in schedule.Input, tz, nextRunAt string) ([]string, error) {
	first, err := timeutil.ParseUTC(nextRunAt)
	if err != nil {
		return nil, err
	}
	out := []string{timeutil.FormatUTC(first)}
	cur := first.Add(time.Second)
	for i := 1; i < UpcomingCount; i++ {
		next, err := schedule.NextRunAt(in, tz, cur)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
		t, err := timeutil.ParseUTC(next)
		if err != nil {
			return nil, err
		}
		cur = t.Add(time.Second)
	}
	return out, ValidateUpcoming(out)
}

// ComputeUpcomingPreview seeds upcoming dates from schedule fields without persisting.
func ComputeUpcomingPreview(in Input, tz string, ref time.Time) ([]string, error) {
	if err := schedule.ValidatePeriodFields(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, true); err != nil {
		return nil, err
	}
	return SeedUpcoming(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, tz, ref)
}
