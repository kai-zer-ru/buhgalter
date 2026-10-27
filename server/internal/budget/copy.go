package budget

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/realtime"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var (
	ErrCopyTargetExists = errors.New("budget already exists for target month")
	ErrNothingToCopy    = errors.New("no budgets to copy from previous month")
)

// copyForwardHook, when set, runs inside the save transaction before the next
// month is written. Tests use it to fail that write and check the source rolls back.
var copyForwardHook func() error

// SetCopyForwardHookForTest installs copyForwardHook. Pass nil to clear it.
func SetCopyForwardHookForTest(fn func() error) {
	copyForwardHook = fn
}

func resolveMonth(ctx context.Context, db *sql.DB, userID, month string) (string, error) {
	if month != "" {
		if _, _, err := parseMonth(month); err != nil {
			return "", ErrInvalidMonth
		}
		return month, nil
	}
	return CurrentMonthQuery(ctx, db, userID)
}

func listBudgetRows(ctx context.Context, db sqlcdb.DBTX, userID, month string) ([]sqlcdb.ListBudgetsByUserRow, error) {
	return listBudgetRowsForOwners(ctx, db, []string{userID}, month)
}

func listBudgetRowsForOwners(ctx context.Context, db sqlcdb.DBTX, ownerIDs []string, month string) ([]sqlcdb.ListBudgetsByUserRow, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	filter := month
	return queries(db).ListBudgetsByUser(ctx, sqlcdb.ListBudgetsByUserParams{
		OwnerIds: ownerIDs, Column2: filter, Month: filter,
	})
}

func countActiveBudgetsMonth(ctx context.Context, db *sql.DB, userID, month string) (int64, error) {
	return queries(db).CountActiveBudgetsByUserMonth(ctx, sqlcdb.CountActiveBudgetsByUserMonthParams{
		UserID: userID, Month: month,
	})
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
		Currency:       src.Currency,
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
			if errors.Is(err, ErrDuplicateActive) || isUniqueConstraintError(err) {
				continue
			}
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
		Currency:       row.Currency,
	}
}

func prepareBudgetInput(ctx context.Context, db sqlcdb.DBTX, userID string, in Input) (Input, error) {
	in, err := resolveScopeRefs(ctx, db, userID, in)
	if err != nil {
		return Input{}, err
	}
	if in.Month == "" {
		in.Month, err = currentMonth(ctx, db, userID)
		if err != nil {
			return Input{}, err
		}
	}
	if _, _, err := parseMonth(in.Month); err != nil {
		return Input{}, ErrInvalidMonth
	}
	in, err = resolveCurrency(ctx, db, userID, in)
	if err != nil {
		return Input{}, err
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Input{}, err
	}
	return in, nil
}

func insertBudget(ctx context.Context, db *sql.DB, userID string, in Input) (Budget, error) {
	in, err := prepareBudgetInput(ctx, db, userID, in)
	if err != nil {
		return Budget{}, err
	}
	if err := checkActiveUniqueness(ctx, db, userID, in, ""); err != nil {
		return Budget{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Budget{}, err
	}
	defer func() { _ = tx.Rollback() }()
	id, err := insertBudgetTx(ctx, tx, userID, in)
	if err != nil {
		return Budget{}, err
	}
	if !in.SkipCopyForward {
		if err := copyForwardOnSave(ctx, tx, userID, in); err != nil {
			return Budget{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Budget{}, err
	}
	return Get(ctx, db, userID, id)
}

func insertBudgetTx(ctx context.Context, tx sqlcdb.DBTX, userID string, in Input) (string, error) {
	tz, err := userTimezone(ctx, tx, userID)
	if err != nil {
		return "", err
	}
	year, mon, err := parseMonth(in.Month)
	if err != nil {
		return "", err
	}
	periodStart, _, err := monthBoundsExclusive(tz, year, mon)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	now := timeutil.FormatUTC(timeutil.NowUTC())
	q := queries(tx)
	if err := q.InsertBudget(ctx, sqlcdb.InsertBudgetParams{
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
		Currency:       in.Currency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return "", err
	}
	if err := q.InsertBudgetPeriod(ctx, sqlcdb.InsertBudgetPeriodParams{
		ID:             uuid.NewString(),
		BudgetID:       id,
		PeriodStart:    periodStart,
		PlannedAmount:  in.Amount,
		RolloverAmount: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return "", err
	}
	return id, nil
}

// copyForwardOnSave writes month+1 when the saved month has already started.
// The source flag is left as requested. The target row always has copy_forward off.
func copyForwardOnSave(ctx context.Context, tx sqlcdb.DBTX, userID string, src Input) error {
	if !src.CopyForward || src.Month == "" {
		return nil
	}
	current, err := currentMonth(ctx, tx, userID)
	if err != nil {
		return err
	}
	if src.Month > current {
		return nil
	}
	if copyForwardHook != nil {
		if err := copyForwardHook(); err != nil {
			return err
		}
	}
	next, err := addMonths(src.Month, 1)
	if err != nil {
		return err
	}
	return upsertCopiedMonth(ctx, tx, userID, src, next)
}

func upsertCopiedMonth(ctx context.Context, tx sqlcdb.DBTX, userID string, src Input, month string) error {
	target := src
	target.Month = month
	target.CopyForward = false
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		rows, err := listBudgetRows(ctx, tx, userID, month)
		if err != nil {
			return err
		}
		if existing := pickCopyTarget(rows, target); existing != nil {
			lastErr = updateCopiedBudget(ctx, tx, userID, existing.ID, target)
		} else {
			_, lastErr = insertBudgetTx(ctx, tx, userID, target)
		}
		if lastErr == nil {
			return nil
		}
		if errors.Is(lastErr, ErrDuplicateActive) || isUniqueConstraintError(lastErr) {
			continue
		}
		return lastErr
	}
	if lastErr == nil {
		return ErrDuplicateActive
	}
	return lastErr
}

func updateCopiedBudget(ctx context.Context, tx sqlcdb.DBTX, userID, id string, in Input) error {
	if err := checkActiveUniqueness(ctx, tx, userID, in, id); err != nil {
		return err
	}
	now := timeutil.FormatUTC(timeutil.NowUTC())
	n, err := queries(tx).UpdateBudget(ctx, sqlcdb.UpdateBudgetParams{
		Name:           in.Name,
		Scope:          in.Scope,
		CategoryID:     in.CategoryID,
		SubcategoryID:  in.SubcategoryID,
		Amount:         in.Amount,
		AccountID:      in.AccountID,
		CopyForward:    0,
		AlertAtPercent: in.AlertAtPercent,
		IsActive:       boolToInt(in.IsActive),
		UpdatedAt:      now,
		ID:             id,
		UserID:         userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return setPeriodPlanned(ctx, tx, userID, id, in.Month, in.Amount)
}

// pickCopyTarget chooses the active row of the same scope, otherwise the inactive
// row with the latest updated_at. account_id is not part of the identity; currency is.
func pickCopyTarget(rows []sqlcdb.ListBudgetsByUserRow, in Input) *sqlcdb.ListBudgetsByUserRow {
	var active *sqlcdb.ListBudgetsByUserRow
	var latest *sqlcdb.ListBudgetsByUserRow
	for i := range rows {
		row := &rows[i]
		if row.Scope != in.Scope || !sameID(row.CategoryID, in.CategoryID) || !sameID(row.SubcategoryID, in.SubcategoryID) ||
			row.Currency != in.Currency {
			continue
		}
		if row.IsActive == 1 {
			active = row
			continue
		}
		if latest == nil || row.UpdatedAt > latest.UpdatedAt {
			latest = row
		}
	}
	if active != nil {
		return active
	}
	return latest
}

// pickCopySources keeps one budget per scope and currency: the active row, otherwise the
// inactive row with the latest updated_at. Only that row may be copied forward.
func pickCopySources(rows []sqlcdb.ListBudgetsByUserRow) []sqlcdb.ListBudgetsByUserRow {
	chosen := make(map[string]int, len(rows))
	order := make([]string, 0, len(rows))
	for i := range rows {
		key := rows[i].Scope + "\x00" + idOrEmpty(rows[i].CategoryID) + "\x00" + idOrEmpty(rows[i].SubcategoryID) + "\x00" + rows[i].Currency
		prev, ok := chosen[key]
		if !ok {
			chosen[key] = i
			order = append(order, key)
			continue
		}
		best := &rows[prev]
		cur := &rows[i]
		if best.IsActive != 1 && cur.IsActive == 1 {
			chosen[key] = i
			continue
		}
		if best.IsActive != 1 && cur.UpdatedAt > best.UpdatedAt {
			chosen[key] = i
		}
	}
	out := make([]sqlcdb.ListBudgetsByUserRow, 0, len(order))
	for _, key := range order {
		out = append(out, rows[chosen[key]])
	}
	return out
}

func sameID(a, b *string) bool {
	return idOrEmpty(a) == idOrEmpty(b)
}

func idOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// CopyMissingNextMonthOnDate creates the missing next month for budgets of
// localDate's month when that date is the 1st and copy_forward is on.
// Existing next-month rows are left unchanged. Other days do nothing.
// localDate is the user's local calendar date, so tests do not depend on the clock.
func CopyMissingNextMonthOnDate(ctx context.Context, db *sql.DB, userID string, localDate time.Time) (int, error) {
	year, month, day := localDate.Date()
	if day != 1 {
		return 0, nil
	}
	sourceMonth := monthQueryValue(year, month)
	next, err := addMonths(sourceMonth, 1)
	if err != nil {
		return 0, err
	}
	rows, err := listBudgetRows(ctx, db, userID, sourceMonth)
	if err != nil {
		return 0, err
	}
	created := 0
	var passErr error
	remember := func(err error) {
		if err != nil && passErr == nil {
			passErr = err
		}
	}
	for _, row := range pickCopySources(rows) {
		if row.CopyForward != 1 {
			continue
		}
		if copyForwardHook != nil {
			if err := copyForwardHook(); err != nil {
				remember(err)
				continue
			}
		}
		in := inputFromRow(row)
		in.Month = next
		in.CopyForward = false
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			remember(err)
			continue
		}
		existing, err := listBudgetRows(ctx, tx, userID, next)
		if err != nil {
			_ = tx.Rollback()
			remember(err)
			continue
		}
		if pickCopyTarget(existing, in) != nil {
			_ = tx.Rollback()
			continue
		}
		if _, err := insertBudgetTx(ctx, tx, userID, in); err != nil {
			_ = tx.Rollback()
			if isUniqueConstraintError(err) {
				continue
			}
			remember(err)
			continue
		}
		if err := tx.Commit(); err != nil {
			remember(err)
			continue
		}
		created++
	}
	if created > 0 {
		realtime.NotifyOwnerDataChanged(ctx, db, userID)
	}
	return created, passErr
}
