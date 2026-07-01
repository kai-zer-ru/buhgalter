package budget

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

const (
	ScopeCategory    = "category"
	ScopeSubcategory = "subcategory"
	ScopeAllExpense  = "all_expense"
)

const (
	StatusOK       = "ok"
	StatusWarning  = "warning"
	StatusExceeded = "exceeded"
)

var (
	ErrNotFound        = errors.New("budget not found")
	ErrDuplicateActive = errors.New("active budget already exists for this scope")
	ErrInvalidScope    = errors.New("invalid budget scope")
	ErrInvalidAmount   = errors.New("invalid budget amount")
	ErrInvalidMonth    = errors.New("invalid month")
	ErrInvalidCategory = errors.New("invalid category")
	ErrInvalidSub      = errors.New("invalid subcategory")
	ErrInvalidAccount  = errors.New("invalid account")
	ErrAccountArchived = errors.New("account archived")
)

type Budget struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Scope           string  `json:"scope"`
	CategoryID      *string `json:"category_id"`
	CategoryName    *string `json:"category_name,omitempty"`
	CategoryIcon    *string `json:"category_icon,omitempty"`
	SubcategoryID   *string `json:"subcategory_id"`
	SubcategoryName *string `json:"subcategory_name,omitempty"`
	Amount          int64   `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	AccountID       *string `json:"account_id"`
	AccountName     *string `json:"account_name,omitempty"`
	Month           string  `json:"month"`
	CopyForward     bool    `json:"copy_forward"`
	AlertAtPercent  int64   `json:"alert_at_percent"`
	IsActive        bool    `json:"is_active"`
	Period          *Period `json:"period,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type Period struct {
	PeriodStart     string `json:"period_start"`
	PlannedAmount   int64  `json:"planned_amount"`
	PlannedDisplay  string `json:"planned_display"`
	RolloverAmount  int64  `json:"rollover_amount"`
}

type SummaryItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Scope           string  `json:"scope"`
	CategoryID      *string `json:"category_id"`
	CategoryName    *string `json:"category_name,omitempty"`
	CategoryIcon    *string `json:"category_icon,omitempty"`
	SubcategoryID   *string `json:"subcategory_id"`
	SubcategoryName *string `json:"subcategory_name,omitempty"`
	AccountID       *string `json:"account_id"`
	AccountName     *string `json:"account_name,omitempty"`
	Planned         int64   `json:"planned"`
	PlannedDisplay  string  `json:"planned_display"`
	Spent           int64   `json:"spent"`
	SpentDisplay    string  `json:"spent_display"`
	Remaining       int64   `json:"remaining"`
	RemainingDisplay string `json:"remaining_display"`
	Percent         int     `json:"percent"`
	Status          string  `json:"status"`
	AlertAtPercent   int64   `json:"alert_at_percent"`
	IsActive         bool    `json:"is_active"`
	CopyForward      bool    `json:"copy_forward"`
	ChildrenPlanned  int64   `json:"children_planned,omitempty"`
	ChildrenPlannedDisplay string `json:"children_planned_display,omitempty"`
	ChildrenSpent    int64   `json:"children_spent,omitempty"`
	ChildrenSpentDisplay string `json:"children_spent_display,omitempty"`
}

type SummaryResult struct {
	Items                 []SummaryItem `json:"items"`
	Month                 string        `json:"month"`
	CanCopyFromPrevious   bool          `json:"can_copy_from_previous"`
}

type Input struct {
	Name           string
	Scope          string
	CategoryID     *string
	SubcategoryID  *string
	Amount         int64
	AccountID      *string
	Month          string
	CopyForward    bool
	AlertAtPercent int64
	IsActive       bool
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries { return sqlcdb.New(db) }

func List(ctx context.Context, db *sql.DB, userID, month string) ([]Budget, error) {
	month, err := resolveMonth(ctx, db, userID, month)
	if err != nil {
		return nil, err
	}
	if err := prepareMonthForLoad(ctx, db, userID, month); err != nil {
		return nil, err
	}
	rows, err := listBudgetRows(ctx, db, userID, month)
	if err != nil {
		return nil, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	year, mon, err := parseMonth(month)
	if err != nil {
		return nil, ErrInvalidMonth
	}
	periodStart, _, err := monthBoundsExclusive(tz, year, mon)
	if err != nil {
		return nil, err
	}
	out := make([]Budget, 0, len(rows))
	for _, row := range rows {
		b := budgetFromListRow(row)
		period, err := ensurePeriod(ctx, db, row.ID, periodStart, row.Amount)
		if err != nil {
			return nil, err
		}
		b.Period = &period
		out = append(out, b)
	}
	return out, nil
}

func Get(ctx context.Context, db *sql.DB, userID, id string) (Budget, error) {
	row, err := queries(db).GetBudgetByID(ctx, sqlcdb.GetBudgetByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Budget{}, ErrNotFound
	}
	if err != nil {
		return Budget{}, err
	}
	return budgetFromGetRow(row), nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in Input) (Budget, error) {
	return insertBudget(ctx, db, userID, in)
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in Input, month string) (Budget, error) {
	existing, err := Get(ctx, db, userID, id)
	if err != nil {
		return Budget{}, err
	}
	in.Month = existing.Month
	in, err = resolveScopeRefs(ctx, db, userID, in)
	if err != nil {
		return Budget{}, err
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Budget{}, err
	}
	if err := checkActiveUniqueness(ctx, db, userID, in, id); err != nil {
		return Budget{}, err
	}
	now := timeutil.FormatUTC(timeutil.NowUTC())
	n, err := queries(db).UpdateBudget(ctx, sqlcdb.UpdateBudgetParams{
		Name:           in.Name,
		Scope:          in.Scope,
		CategoryID:     in.CategoryID,
		SubcategoryID:  in.SubcategoryID,
		Amount:         in.Amount,
		AccountID:      in.AccountID,
		CopyForward:    boolToInt(in.CopyForward),
		AlertAtPercent: in.AlertAtPercent,
		IsActive:       boolToInt(in.IsActive),
		UpdatedAt:      now,
		ID:             id,
		UserID:         userID,
	})
	if err != nil {
		return Budget{}, err
	}
	if n == 0 {
		return Budget{}, ErrNotFound
	}
	if month != "" && existing.Amount != in.Amount {
		tz, err := userTimezone(ctx, db, userID)
		if err != nil {
			return Budget{}, err
		}
		year, mon, err := parseMonth(month)
		if err != nil {
			return Budget{}, ErrInvalidMonth
		}
		periodStart, _, err := monthBoundsExclusive(tz, year, mon)
		if err != nil {
			return Budget{}, err
		}
		if _, err := ensurePeriod(ctx, db, id, periodStart, in.Amount); err != nil {
			return Budget{}, err
		}
		if _, err := queries(db).UpdateBudgetPeriodPlannedAmount(ctx, sqlcdb.UpdateBudgetPeriodPlannedAmountParams{
			PlannedAmount: in.Amount,
			UpdatedAt:     now,
			BudgetID:      id,
			PeriodStart:   periodStart,
		}); err != nil {
			return Budget{}, err
		}
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	n, err := queries(db).DeleteBudget(ctx, sqlcdb.DeleteBudgetParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func Summary(ctx context.Context, db *sql.DB, userID, month string) (SummaryResult, error) {
	month, err := resolveMonth(ctx, db, userID, month)
	if err != nil {
		return SummaryResult{}, err
	}
	if err := prepareMonthForLoad(ctx, db, userID, month); err != nil {
		return SummaryResult{}, err
	}
	canCopy, err := canCopyFromPrevious(ctx, db, userID, month)
	if err != nil {
		return SummaryResult{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return SummaryResult{}, err
	}
	year, mon, err := parseMonth(month)
	if err != nil {
		return SummaryResult{}, ErrInvalidMonth
	}
	periodStart, periodEnd, err := monthBoundsExclusive(tz, year, mon)
	if err != nil {
		return SummaryResult{}, err
	}
	rows, err := listBudgetRows(ctx, db, userID, month)
	if err != nil {
		return SummaryResult{}, err
	}
	out := make([]SummaryItem, 0)
	for _, row := range rows {
		if row.IsActive != 1 {
			continue
		}
		period, err := ensurePeriod(ctx, db, row.ID, periodStart, row.Amount)
		if err != nil {
			return SummaryResult{}, err
		}
		spent, err := spentForBudget(ctx, db, userID, row.Scope, row.CategoryID, row.SubcategoryID, row.AccountID, periodStart, periodEnd)
		if err != nil {
			return SummaryResult{}, err
		}
		planned := period.PlannedAmount
		percent, status := computeStatus(spent, planned, row.AlertAtPercent)
		remaining := planned - spent
		out = append(out, SummaryItem{
			ID:               row.ID,
			Name:             row.Name,
			Scope:            row.Scope,
			CategoryID:       row.CategoryID,
			CategoryName:     row.CategoryName,
			CategoryIcon:     row.CategoryIcon,
			SubcategoryID:    row.SubcategoryID,
			SubcategoryName:  row.SubcategoryName,
			AccountID:        row.AccountID,
			AccountName:      row.AccountName,
			Planned:          planned,
			PlannedDisplay:   money.FormatRubles(planned),
			Spent:            spent,
			SpentDisplay:     money.FormatRubles(spent),
			Remaining:        remaining,
			RemainingDisplay: money.FormatRubles(remaining),
			Percent:          percent,
			Status:           status,
			AlertAtPercent:   row.AlertAtPercent,
			IsActive:         true,
			CopyForward:      row.CopyForward == 1,
		})
	}
	out = enrichAndSortSummary(out)
	return SummaryResult{Items: out, Month: month, CanCopyFromPrevious: canCopy}, nil
}

func ensurePeriod(ctx context.Context, db *sql.DB, budgetID, periodStart string, planned int64) (Period, error) {
	row, err := queries(db).GetBudgetPeriod(ctx, sqlcdb.GetBudgetPeriodParams{
		BudgetID: budgetID, PeriodStart: periodStart,
	})
	if err == nil {
		return Period{
			PeriodStart:    row.PeriodStart,
			PlannedAmount:  row.PlannedAmount,
			PlannedDisplay: money.FormatRubles(row.PlannedAmount),
			RolloverAmount: row.RolloverAmount,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Period{}, err
	}
	id := uuid.NewString()
	now := timeutil.FormatUTC(timeutil.NowUTC())
	if err := queries(db).InsertBudgetPeriod(ctx, sqlcdb.InsertBudgetPeriodParams{
		ID:             id,
		BudgetID:       budgetID,
		PeriodStart:    periodStart,
		PlannedAmount:  planned,
		RolloverAmount: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return Period{}, err
	}
	return Period{
		PeriodStart:    periodStart,
		PlannedAmount:  planned,
		PlannedDisplay: money.FormatRubles(planned),
		RolloverAmount: 0,
	}, nil
}

// ComputeStatus returns percent used and status for tests and integrations.
func ComputeStatus(spent, planned, alertAt int64) (int, string) {
	return computeStatus(spent, planned, alertAt)
}

func computeStatus(spent, planned, alertAt int64) (int, string) {
	if planned <= 0 {
		return 0, StatusOK
	}
	percent := int(spent * 100 / planned)
	if spent > planned {
		return percent, StatusExceeded
	}
	if alertAt > 0 && int64(percent) >= alertAt {
		return percent, StatusWarning
	}
	return percent, StatusOK
}

func validateInput(ctx context.Context, db *sql.DB, userID string, in Input) error {
	if in.Name == "" {
		return fmt.Errorf("name required")
	}
	if in.Amount <= 0 {
		return ErrInvalidAmount
	}
	if in.AlertAtPercent < 0 || in.AlertAtPercent > 100 {
		return fmt.Errorf("invalid alert_at_percent")
	}
	switch in.Scope {
	case ScopeCategory, ScopeSubcategory, ScopeAllExpense:
	default:
		return ErrInvalidScope
	}
	if in.Scope == ScopeCategory {
		if in.CategoryID == nil || *in.CategoryID == "" {
			return ErrInvalidCategory
		}
		if err := validateExpenseCategory(ctx, db, userID, *in.CategoryID); err != nil {
			return err
		}
	}
	if in.Scope == ScopeSubcategory {
		if in.SubcategoryID == nil || *in.SubcategoryID == "" {
			return ErrInvalidSub
		}
		sub, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{
			ID: *in.SubcategoryID, UserID: userID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidSub
		}
		if err != nil {
			return err
		}
		if err := validateExpenseCategory(ctx, db, userID, sub.CategoryID); err != nil {
			return err
		}
	}
	if in.AccountID != nil && *in.AccountID != "" {
		acc, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{
			ID: *in.AccountID, UserID: userID,
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
	}
	return nil
}

func resolveScopeRefs(ctx context.Context, db *sql.DB, userID string, in Input) (Input, error) {
	if in.Scope != ScopeSubcategory {
		return in, nil
	}
	if in.SubcategoryID == nil || *in.SubcategoryID == "" {
		return Input{}, ErrInvalidSub
	}
	sub, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{
		ID: *in.SubcategoryID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Input{}, ErrInvalidSub
	}
	if err != nil {
		return Input{}, err
	}
	catID := sub.CategoryID
	in.CategoryID = &catID
	return in, nil
}

func validateExpenseCategory(ctx context.Context, db *sql.DB, userID, categoryID string) error {
	cat, err := queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{
		ID: categoryID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCategory
	}
	if err != nil {
		return err
	}
	if cat.Type != "expense" {
		return ErrInvalidCategory
	}
	return nil
}

func checkActiveUniqueness(ctx context.Context, db *sql.DB, userID string, in Input, excludeID string) error {
	if !in.IsActive {
		return nil
	}
	exclude := ""
	if excludeID != "" {
		exclude = excludeID
	}
	cnt, err := queries(db).CountActiveBudgetConflict(ctx, sqlcdb.CountActiveBudgetConflictParams{
		UserID:    userID,
		Scope:     in.Scope,
		IFNULL:    in.CategoryID,
		IFNULL_2:  in.SubcategoryID,
		Month:     in.Month,
		ExcludeID: exclude,
	})
	if err != nil {
		return err
	}
	if cnt > 0 {
		return ErrDuplicateActive
	}
	return nil
}

func userTimezone(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "UTC", nil
	}
	return tz, nil
}

func budgetFromListRow(row sqlcdb.ListBudgetsByUserRow) Budget {
	return Budget{
		ID:              row.ID,
		Name:            row.Name,
		Scope:           row.Scope,
		CategoryID:      row.CategoryID,
		CategoryName:    row.CategoryName,
		CategoryIcon:    row.CategoryIcon,
		SubcategoryID:   row.SubcategoryID,
		SubcategoryName: row.SubcategoryName,
		Amount:          row.Amount,
		AmountDisplay:   money.FormatRubles(row.Amount),
		AccountID:       row.AccountID,
		AccountName:     row.AccountName,
		Month:           row.Month,
		CopyForward:     row.CopyForward == 1,
		AlertAtPercent:  row.AlertAtPercent,
		IsActive:        row.IsActive == 1,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func budgetFromGetRow(row sqlcdb.GetBudgetByIDRow) Budget {
	return Budget{
		ID:              row.ID,
		Name:            row.Name,
		Scope:           row.Scope,
		CategoryID:      row.CategoryID,
		CategoryName:    row.CategoryName,
		CategoryIcon:    row.CategoryIcon,
		SubcategoryID:   row.SubcategoryID,
		SubcategoryName: row.SubcategoryName,
		Amount:          row.Amount,
		AmountDisplay:   money.FormatRubles(row.Amount),
		AccountID:       row.AccountID,
		AccountName:     row.AccountName,
		Month:           row.Month,
		CopyForward:     row.CopyForward == 1,
		AlertAtPercent:  row.AlertAtPercent,
		IsActive:        row.IsActive == 1,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// MonthBounds returns period_start and period_end (exclusive) for a YYYY-MM month in user TZ.
func MonthBounds(ctx context.Context, db *sql.DB, userID, month string) (start, endExclusive string, err error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return "", "", err
	}
	year, mon, err := parseMonth(month)
	if err != nil {
		return "", "", ErrInvalidMonth
	}
	return monthBoundsExclusive(tz, year, mon)
}

// CurrentMonthQuery returns YYYY-MM for the user's current month.
func CurrentMonthQuery(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return "", err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	now := timeutil.NowUTC().In(loc)
	return monthQueryValue(now.Year(), now.Month()), nil
}
