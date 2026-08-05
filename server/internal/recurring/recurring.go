package recurring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/balancehooks"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/schedule"
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
	ErrInvalidPeriod   = schedule.ErrInvalidPeriod
	ErrInvalidWeekday  = schedule.ErrInvalidWeekday
	ErrInvalidDay      = schedule.ErrInvalidDay
	ErrInvalidTime     = schedule.ErrInvalidTime
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
	affectedAccounts := make(map[string]struct{}, len(due))
	var lastAsOf time.Time
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
		affectedAccounts[op.AccountID] = struct{}{}
		if runAt, err := timeutil.ParseUTC(op.NextRunAt); err == nil {
			lastAsOf = runAt
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
	if len(affectedAccounts) > 0 {
		accountIDs := make([]string, 0, len(affectedAccounts))
		for id := range affectedAccounts {
			accountIDs = append(accountIDs, id)
		}
		if err := accountbalance.Refresh(ctx, db, userID, accountIDs...); err != nil {
			return applied, err
		}
		balancehooks.NotifyRefresh(ctx, db, userID, lastAsOf, accountIDs...)
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
	return schedule.ValidatePeriodFields(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, false)
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
	return schedule.NextRunAt(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, tz, ref)
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
