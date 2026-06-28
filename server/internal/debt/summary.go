package debt

import (
	"context"
	"database/sql"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Summary struct {
	IOwe            int64 `json:"i_owe"`
	OwedToMe        int64 `json:"owed_to_me"`
	OverdueIOwe     int64 `json:"overdue_i_owe"`
	OverdueOwedToMe int64 `json:"overdue_owed_to_me"`
	ActiveCount     int64 `json:"active_count"`
}

type summaryRow struct {
	Direction string
	Amount    int64
	DueDate   string
}

func ComputeSummary(rows []summaryRow, tz string, now time.Time) (Summary, error) {
	var s Summary
	for _, row := range rows {
		s.ActiveCount++
		due, err := timeutil.ParseUTC(row.DueDate)
		if err != nil {
			return Summary{}, err
		}
		overdue, err := timeutil.IsOverdueInTZ(due, now, tz)
		if err != nil {
			return Summary{}, err
		}
		switch row.Direction {
		case "borrowed":
			s.IOwe += row.Amount
			if overdue {
				s.OverdueIOwe += row.Amount
			}
		case "lent":
			s.OwedToMe += row.Amount
			if overdue {
				s.OverdueOwedToMe += row.Amount
			}
		}
	}
	return s, nil
}

func SummaryForUser(ctx context.Context, db *sql.DB, userID string) (Summary, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Summary{}, err
	}
	rows, err := queries(db).ListActiveDebtsForSummary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	input := make([]summaryRow, 0, len(rows))
	for _, row := range rows {
		input = append(input, summaryRow{
			Direction: row.Direction,
			Amount:    row.Amount,
			DueDate:   row.DueDate,
		})
	}
	return ComputeSummary(input, tz, timeutil.NowUTC())
}
