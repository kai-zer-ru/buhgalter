package recurring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Operation struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Amount          int64   `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	Description     *string `json:"description"`
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	CategoryID      string  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	SubcategoryID   *string `json:"subcategory_id"`
	SubcategoryName *string `json:"subcategory_name"`
	Period          string  `json:"period"`
	Weekday         *int64  `json:"weekday"`
	DayOfMonth      *int64  `json:"day_of_month"`
	StartDate       string  `json:"start_date"`
	TimeLocal       string  `json:"time_local"`
	NextRunAt       string  `json:"next_run_at"`
	LastRunAt       *string `json:"last_run_at"`
	Active          bool    `json:"active"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type Input struct {
	Type          string
	Amount        int64
	Description   *string
	AccountID     string
	CategoryID    string
	SubcategoryID *string
	Period        string
	Weekday       *int64
	DayOfMonth    *int64
	StartDate     time.Time
	TimeLocal     string
	Active        bool
}

var (
	ErrNotFound        = errors.New("recurring operation not found")
	ErrInvalidType     = errors.New("invalid type")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrInvalidPeriod   = errors.New("invalid period")
	ErrInvalidWeekday  = errors.New("invalid weekday")
	ErrInvalidDay      = errors.New("invalid day")
	ErrInvalidTime     = errors.New("invalid time")
	ErrInvalidAccount  = errors.New("invalid account")
	ErrAccountArchived = errors.New("account archived")
	ErrInvalidCategory = errors.New("invalid category")
	ErrInvalidSub      = errors.New("invalid subcategory")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries { return sqlcdb.New(db) }

func opFromRow(row sqlcdb.ListRecurringOperationsByUserRow) Operation {
	return Operation{
		ID:              row.ID,
		Type:            row.Type,
		Amount:          row.Amount,
		AmountDisplay:   money.FormatRubles(row.Amount),
		Description:     row.Description,
		AccountID:       row.AccountID,
		AccountName:     row.AccountName,
		CategoryID:      row.CategoryID,
		CategoryName:    row.CategoryName,
		SubcategoryID:   row.SubcategoryID,
		SubcategoryName: row.SubcategoryName,
		Period:          row.Period,
		Weekday:         row.Weekday,
		DayOfMonth:      row.DayOfMonth,
		StartDate:       row.StartDate,
		TimeLocal:       row.TimeLocal,
		NextRunAt:       row.NextRunAt,
		LastRunAt:       row.LastRunAt,
		Active:          row.Active == 1,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func opFromGetRow(row sqlcdb.GetRecurringOperationByIDRow) Operation {
	return Operation{
		ID:              row.ID,
		Type:            row.Type,
		Amount:          row.Amount,
		AmountDisplay:   money.FormatRubles(row.Amount),
		Description:     row.Description,
		AccountID:       row.AccountID,
		AccountName:     row.AccountName,
		CategoryID:      row.CategoryID,
		CategoryName:    row.CategoryName,
		SubcategoryID:   row.SubcategoryID,
		SubcategoryName: row.SubcategoryName,
		Period:          row.Period,
		Weekday:         row.Weekday,
		DayOfMonth:      row.DayOfMonth,
		StartDate:       row.StartDate,
		TimeLocal:       row.TimeLocal,
		NextRunAt:       row.NextRunAt,
		LastRunAt:       row.LastRunAt,
		Active:          row.Active == 1,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func getByID(ctx context.Context, db *sql.DB, userID, id string) (Operation, error) {
	row, err := queries(db).GetRecurringOperationByID(ctx, sqlcdb.GetRecurringOperationByIDParams{
		ID: id, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound
	}
	if err != nil {
		return Operation{}, err
	}
	return opFromGetRow(row), nil
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Operation, error) {
	rows, err := queries(db).ListRecurringOperationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Operation, 0, len(rows))
	for _, row := range rows {
		out = append(out, opFromRow(row))
	}
	return out, nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in Input) (Operation, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Operation{}, err
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Operation{}, err
	}
	now := timeutil.NowUTC()
	next, err := nextRunAt(in, tz, now)
	if err != nil {
		return Operation{}, err
	}
	id := uuid.NewString()
	ts := timeutil.FormatUTC(now)
	if err := queries(db).InsertRecurringOperation(ctx, sqlcdb.InsertRecurringOperationParams{
		ID:            id,
		UserID:        userID,
		Type:          in.Type,
		Amount:        in.Amount,
		Description:   in.Description,
		AccountID:     in.AccountID,
		CategoryID:    in.CategoryID,
		SubcategoryID: in.SubcategoryID,
		Period:        in.Period,
		Weekday:       in.Weekday,
		DayOfMonth:    in.DayOfMonth,
		StartDate:     timeutil.FormatUTC(in.StartDate),
		TimeLocal:     in.TimeLocal,
		NextRunAt:     next,
		LastRunAt:     nil,
		Active:        boolToInt(in.Active),
		CreatedAt:     ts,
		UpdatedAt:     ts,
	}); err != nil {
		return Operation{}, err
	}
	return getByID(ctx, db, userID, id)
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in Input) (Operation, error) {
	if _, err := getByID(ctx, db, userID, id); err != nil {
		return Operation{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Operation{}, err
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Operation{}, err
	}
	next, err := nextRunAt(in, tz, timeutil.NowUTC())
	if err != nil {
		return Operation{}, err
	}
	n, err := queries(db).UpdateRecurringOperation(ctx, sqlcdb.UpdateRecurringOperationParams{
		Type:          in.Type,
		Amount:        in.Amount,
		Description:   in.Description,
		AccountID:     in.AccountID,
		CategoryID:    in.CategoryID,
		SubcategoryID: in.SubcategoryID,
		Period:        in.Period,
		Weekday:       in.Weekday,
		DayOfMonth:    in.DayOfMonth,
		StartDate:     timeutil.FormatUTC(in.StartDate),
		TimeLocal:     in.TimeLocal,
		NextRunAt:     next,
		Active:        boolToInt(in.Active),
		UpdatedAt:     timeutil.FormatUTC(timeutil.NowUTC()),
		ID:            id,
		UserID:        userID,
	})
	if err != nil {
		return Operation{}, err
	}
	if n == 0 {
		return Operation{}, ErrNotFound
	}
	return getByID(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	n, err := queries(db).DeleteRecurringOperation(ctx, sqlcdb.DeleteRecurringOperationParams{
		ID: id, UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func ApplyDue(ctx context.Context, db *sql.DB, userID string, now time.Time, tz string) (int, error) {
	cutoff := timeutil.FormatUTC(now)
	due, err := queries(db).ListDueRecurringOperations(ctx, sqlcdb.ListDueRecurringOperationsParams{
		UserID: userID, NextRunAt: cutoff,
	})
	if err != nil {
		return 0, err
	}
	if len(due) == 0 {
		return 0, nil
	}
	q := queries(db)
	applied := 0
	for _, op := range due {
		if err := validateDueOperation(ctx, db, userID, op); err != nil {
			continue
		}
		txID := uuid.NewString()
		createdAt := timeutil.FormatUTC(now)
		if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
			ID:                txID,
			UserID:            userID,
			AccountID:         op.AccountID,
			Type:              op.Type,
			Kind:              "manual",
			Amount:            op.Amount,
			Description:       op.Description,
			CategoryID:        strPtr(op.CategoryID),
			SubcategoryID:     op.SubcategoryID,
			TransferGroupID:   nil,
			TransferAccountID: nil,
			TransactionDate:   op.NextRunAt,
			AffectsBalance:    1,
			CreatedAt:         createdAt,
			UpdatedAt:         createdAt,
		}); err != nil {
			continue
		}
		nextInput := Input{
			Type:          op.Type,
			Amount:        op.Amount,
			Description:   op.Description,
			AccountID:     op.AccountID,
			CategoryID:    op.CategoryID,
			SubcategoryID: op.SubcategoryID,
			Period:        op.Period,
			Weekday:       op.Weekday,
			DayOfMonth:    op.DayOfMonth,
			StartDate:     mustParse(op.StartDate),
			TimeLocal:     op.TimeLocal,
			Active:        op.Active == 1,
		}
		next, err := nextRunAt(nextInput, tz, mustParse(op.NextRunAt).Add(time.Second))
		if err != nil {
			continue
		}
		_, _ = q.MarkRecurringOperationRan(ctx, sqlcdb.MarkRecurringOperationRanParams{
			NextRunAt: next,
			LastRunAt: strPtr(op.NextRunAt),
			UpdatedAt: createdAt,
			ID:        op.ID,
			UserID:    userID,
		})
		applied++
	}
	return applied, nil
}

func validateInput(ctx context.Context, db *sql.DB, userID string, in Input) error {
	if in.Type != "income" && in.Type != "expense" {
		return ErrInvalidType
	}
	if in.Amount <= 0 {
		return ErrInvalidAmount
	}
	if _, _, err := parseLocalTime(in.TimeLocal); err != nil {
		return ErrInvalidTime
	}
	if err := validatePeriodFields(in); err != nil {
		return err
	}
	acc, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{
		ID: in.AccountID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if acc.Status != "active" {
		return ErrAccountArchived
	}
	cat, err := queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{
		ID: in.CategoryID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCategory
	}
	if err != nil {
		return err
	}
	if cat.Type != in.Type {
		return ErrInvalidCategory
	}
	if in.SubcategoryID != nil && *in.SubcategoryID != "" {
		sub, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{
			ID: *in.SubcategoryID, UserID: userID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidSub
		}
		if err != nil {
			return err
		}
		if sub.CategoryID != in.CategoryID {
			return ErrInvalidSub
		}
	}
	return nil
}

func validateDueOperation(ctx context.Context, db *sql.DB, userID string, op sqlcdb.RecurringOperation) error {
	_, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{
		ID: op.AccountID, UserID: userID,
	})
	if err != nil {
		return err
	}
	_, err = queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{
		ID: op.CategoryID, UserID: userID,
	})
	return err
}

func validatePeriodFields(in Input) error {
	switch in.Period {
	case "week", "two_weeks":
		if in.Weekday == nil || *in.Weekday < 1 || *in.Weekday > 7 {
			return ErrInvalidWeekday
		}
		in.DayOfMonth = nil
	case "month", "year":
		if in.DayOfMonth == nil || *in.DayOfMonth < 1 || *in.DayOfMonth > 31 {
			return ErrInvalidDay
		}
		in.Weekday = nil
	default:
		return ErrInvalidPeriod
	}
	return nil
}

func userTimezone(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func nextRunAt(in Input, tz string, ref time.Time) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	hour, minute, err := parseLocalTime(in.TimeLocal)
	if err != nil {
		return "", err
	}
	threshold := ref.In(loc)
	startLocal := in.StartDate.In(loc)
	startPoint := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), hour, minute, 0, 0, loc)
	if threshold.Before(startPoint) {
		threshold = startPoint
	}

	var candidate time.Time
	switch in.Period {
	case "week":
		candidate = nextWeekdayAtOrAfter(threshold, int(*in.Weekday), hour, minute)
	case "two_weeks":
		first := nextWeekdayAtOrAfter(startPoint, int(*in.Weekday), hour, minute)
		candidate = first
		for candidate.Before(threshold) {
			candidate = candidate.AddDate(0, 0, 14)
		}
	case "month":
		candidate = nextMonthDayAtOrAfter(threshold, int(*in.DayOfMonth), hour, minute)
	case "year":
		candidate = nextYearDayAtOrAfter(threshold, startPoint.Month(), int(*in.DayOfMonth), hour, minute)
	default:
		return "", ErrInvalidPeriod
	}
	return timeutil.FormatUTC(candidate.UTC()), nil
}

func nextWeekdayAtOrAfter(base time.Time, weekday, hour, minute int) time.Time {
	target := toGoWeekday(weekday)
	candidate := time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
	diff := (int(target) - int(candidate.Weekday()) + 7) % 7
	candidate = candidate.AddDate(0, 0, diff)
	if candidate.Before(base) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

func nextMonthDayAtOrAfter(base time.Time, day, hour, minute int) time.Time {
	y, m, _ := base.Date()
	candidate := monthDayAt(base.Location(), y, m, day, hour, minute)
	if candidate.Before(base) {
		candidate = monthDayAt(base.Location(), y, m+1, day, hour, minute)
	}
	return candidate
}

func nextYearDayAtOrAfter(base time.Time, anchorMonth time.Month, day, hour, minute int) time.Time {
	y := base.Year()
	candidate := monthDayAt(base.Location(), y, anchorMonth, day, hour, minute)
	if candidate.Before(base) {
		candidate = monthDayAt(base.Location(), y+1, anchorMonth, day, hour, minute)
	}
	return candidate
}

func monthDayAt(loc *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	for month > 12 {
		year++
		month -= 12
	}
	for month < 1 {
		year--
		month += 12
	}
	maxDay := daysInMonth(year, month)
	if day > maxDay {
		day = maxDay
	}
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func parseLocalTime(v string) (int, int, error) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

func toGoWeekday(v int) time.Weekday {
	switch v {
	case 1:
		return time.Monday
	case 2:
		return time.Tuesday
	case 3:
		return time.Wednesday
	case 4:
		return time.Thursday
	case 5:
		return time.Friday
	case 6:
		return time.Saturday
	default:
		return time.Sunday
	}
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func mustParse(v string) time.Time {
	t, err := timeutil.ParseUTC(v)
	if err != nil {
		panic(fmt.Sprintf("invalid stored datetime: %s", v))
	}
	return t
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
