package budget

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var (
	ErrCopyTargetExists = errors.New("budget already exists for target month")
	ErrNothingToCopy    = errors.New("no budgets to copy from previous month")
)

func resolveMonth(ctx context.Context, db *sql.DB, userID, month string) (string, error) {
	if month != "" {
		if _, _, err := parseMonth(month); err != nil {
			return "", ErrInvalidMonth
		}
		return month, nil
	}
	return CurrentMonthQuery(ctx, db, userID)
}

func listBudgetRows(ctx context.Context, db *sql.DB, userID, month string) ([]sqlcdb.ListBudgetsByUserRow, error) {
	filter := month
	return queries(db).ListBudgetsByUser(ctx, sqlcdb.ListBudgetsByUserParams{
		UserID: userID, Column2: filter, Month: filter,
	})
}

func countActiveBudgetsMonth(ctx context.Context, db *sql.DB, userID, month string) (int64, error) {
	return queries(db).CountActiveBudgetsByUserMonth(ctx, sqlcdb.CountActiveBudgetsByUserMonthParams{
		UserID: userID, Month: month,
	})
}

func prepareMonthForLoad(ctx context.Context, db *sql.DB, userID, month string) error {
	return maybeAutoCopyFromPrevious(ctx, db, userID, month)
}

func canCopyFromPrevious(ctx context.Context, db *sql.DB, userID, month string) (bool, error) {
	cur, err := countActiveBudgetsMonth(ctx, db, userID, month)
	if err != nil || cur > 0 {
		return false, err
	}
	prevMonth, err := addMonths(month, -1)
	if err != nil {
		return false, err
	}
	prev, err := countActiveBudgetsMonth(ctx, db, userID, prevMonth)
	if err != nil {
		return false, err
	}
	return prev > 0, nil
}

func maybeAutoCopyFromPrevious(ctx context.Context, db *sql.DB, userID, month string) error {
	prevMonth, err := addMonths(month, -1)
	if err != nil {
		return err
	}
	rows, err := listBudgetRows(ctx, db, userID, prevMonth)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.IsActive != 1 || row.CopyForward != 1 {
			continue
		}
		in := inputFromRow(row)
		in.Month = month
		in.CopyForward = false
		if err := checkActiveUniqueness(ctx, db, userID, in, ""); err != nil {
			if errors.Is(err, ErrDuplicateActive) {
				continue
			}
			return err
		}
		if _, err := insertBudget(ctx, db, userID, in); err != nil {
			return err
		}
	}
	return nil
}

// CopyToNextMonth clones a budget into the following calendar month.
func CopyToNextMonth(ctx context.Context, db *sql.DB, userID, id string) (Budget, error) {
	src, err := Get(ctx, db, userID, id)
	if err != nil {
		return Budget{}, err
	}
	targetMonth, err := addMonths(src.Month, 1)
	if err != nil {
		return Budget{}, err
	}
	in := Input{
		Name:           src.Name,
		Scope:          src.Scope,
		CategoryID:     src.CategoryID,
		SubcategoryID:  src.SubcategoryID,
		Amount:         src.Amount,
		AccountID:      src.AccountID,
		AlertAtPercent: src.AlertAtPercent,
		IsActive:       src.IsActive,
		Month:          targetMonth,
		CopyForward:    false,
	}
	if err := checkActiveUniqueness(ctx, db, userID, in, ""); err != nil {
		if errors.Is(err, ErrDuplicateActive) {
			return Budget{}, ErrCopyTargetExists
		}
		return Budget{}, err
	}
	return insertBudget(ctx, db, userID, in)
}

// CopyFromPreviousMonth copies all active budgets from the previous month into month.
func CopyFromPreviousMonth(ctx context.Context, db *sql.DB, userID, month string) ([]Budget, error) {
	month, err := resolveMonth(ctx, db, userID, month)
	if err != nil {
		return nil, err
	}
	prevMonth, err := addMonths(month, -1)
	if err != nil {
		return nil, err
	}
	rows, err := listBudgetRows(ctx, db, userID, prevMonth)
	if err != nil {
		return nil, err
	}
	var out []Budget
	for _, row := range rows {
		if row.IsActive != 1 {
			continue
		}
		in := inputFromRow(row)
		in.Month = month
		in.CopyForward = false
		if err := checkActiveUniqueness(ctx, db, userID, in, ""); err != nil {
			if errors.Is(err, ErrDuplicateActive) {
				continue
			}
			return nil, err
		}
		b, err := insertBudget(ctx, db, userID, in)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return nil, ErrNothingToCopy
	}
	return out, nil
}

func inputFromRow(row sqlcdb.ListBudgetsByUserRow) Input {
	return Input{
		Name:           row.Name,
		Scope:          row.Scope,
		CategoryID:     row.CategoryID,
		SubcategoryID:  row.SubcategoryID,
		Amount:         row.Amount,
		AccountID:      row.AccountID,
		AlertAtPercent: row.AlertAtPercent,
		IsActive:       row.IsActive == 1,
		Month:          row.Month,
		CopyForward:    row.CopyForward == 1,
	}
}

func insertBudget(ctx context.Context, db *sql.DB, userID string, in Input) (Budget, error) {
	in, err := resolveScopeRefs(ctx, db, userID, in)
	if err != nil {
		return Budget{}, err
	}
	if in.Month == "" {
		var err error
		in.Month, err = CurrentMonthQuery(ctx, db, userID)
		if err != nil {
			return Budget{}, err
		}
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Budget{}, err
	}
	if err := checkActiveUniqueness(ctx, db, userID, in, ""); err != nil {
		return Budget{}, err
	}
	id := uuid.NewString()
	now := timeutil.FormatUTC(timeutil.NowUTC())
	if err := queries(db).InsertBudget(ctx, sqlcdb.InsertBudgetParams{
		ID:             id,
		UserID:         userID,
		Name:           in.Name,
		Scope:          in.Scope,
		CategoryID:     in.CategoryID,
		SubcategoryID:  in.SubcategoryID,
		Amount:         in.Amount,
		Period:         "month",
		AccountID:      in.AccountID,
		Month:          in.Month,
		CopyForward:    boolToInt(in.CopyForward),
		Rollover:       0,
		AlertAtPercent: in.AlertAtPercent,
		IsActive:       boolToInt(in.IsActive),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return Budget{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Budget{}, err
	}
	year, mon, err := parseMonth(in.Month)
	if err != nil {
		return Budget{}, err
	}
	periodStart, _, err := monthBoundsExclusive(tz, year, mon)
	if err != nil {
		return Budget{}, err
	}
	if _, err := ensurePeriod(ctx, db, id, periodStart, in.Amount); err != nil {
		return Budget{}, err
	}
	return Get(ctx, db, userID, id)
}
