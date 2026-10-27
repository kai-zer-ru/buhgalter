package debt

import (
	"context"
	"database/sql"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Summary struct {
	IOwe            int64             `json:"i_owe"`
	OwedToMe        int64             `json:"owed_to_me"`
	OverdueIOwe     int64             `json:"overdue_i_owe"`
	OverdueOwedToMe int64             `json:"overdue_owed_to_me"`
	ActiveCount     int64             `json:"active_count"`
	ByCurrency      []CurrencySummary `json:"by_currency"`
}

type CurrencySummary struct {
	IOwe            int64  `json:"i_owe"`
	OwedToMe        int64  `json:"owed_to_me"`
	OverdueIOwe     int64  `json:"overdue_i_owe"`
	OverdueOwedToMe int64  `json:"overdue_owed_to_me"`
	ActiveCount     int64  `json:"active_count"`
	Currency        string `json:"currency"`
}

type summaryRow struct {
	Direction string
	Amount    int64
	DueDate   string
	Currency  string
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
	return SummaryForOwners(ctx, db, userID, []string{userID})
}

func SummaryForOwners(ctx context.Context, db *sql.DB, tzUserID string, ownerIDs []string) (Summary, error) {
	if len(ownerIDs) == 0 {
		return Summary{}, nil
	}
	tz, err := userTimezone(ctx, db, tzUserID)
	if err != nil {
		return Summary{}, err
	}
	rows, err := queries(db).ListActiveDebtsForSummary(ctx, ownerIDs)
	if err != nil {
		return Summary{}, err
	}
	input := make([]summaryRow, 0, len(rows))
	for _, row := range rows {
		input = append(input, summaryRow{
			Direction: row.Direction,
			Amount:    row.Amount,
			DueDate:   row.DueDate,
			Currency:  money.ProfileOrDefault(row.Currency),
		})
	}
	now := timeutil.NowUTC()
	profile, err := account.ProfileCurrency(ctx, db, tzUserID)
	if err != nil {
		return Summary{}, err
	}
	return summarizeByCurrency(input, tz, now, profile)
}

func summarizeByCurrency(rows []summaryRow, tz string, now time.Time, profile string) (Summary, error) {
	by := map[string][]summaryRow{}
	for _, row := range rows {
		by[row.Currency] = append(by[row.Currency], row)
	}
	codes := make([]string, 0, len(by))
	for c := range by {
		codes = append(codes, c)
	}
	order := money.OrderPresentWithProfileFirst(profile, codes)
	out := Summary{ByCurrency: []CurrencySummary{}}
	for _, c := range order {
		part, err := ComputeSummary(by[c], tz, now)
		if err != nil {
			return Summary{}, err
		}
		out.ByCurrency = append(out.ByCurrency, CurrencySummary{
			IOwe: part.IOwe, OwedToMe: part.OwedToMe,
			OverdueIOwe: part.OverdueIOwe, OverdueOwedToMe: part.OverdueOwedToMe,
			ActiveCount: part.ActiveCount, Currency: c,
		})
		if c == profile {
			out.IOwe = part.IOwe
			out.OwedToMe = part.OwedToMe
			out.OverdueIOwe = part.OverdueIOwe
			out.OverdueOwedToMe = part.OverdueOwedToMe
			out.ActiveCount = part.ActiveCount
		}
	}
	return out, nil
}
