package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/balancehooks"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/recurring"
	"github.com/kai-zer-ru/buhgalter/internal/schedule"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Subscription struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Icon            *string `json:"icon"`
	WebsiteURL      *string `json:"website_url"`
	Amount          int64   `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name"`
	SubcategoryID   *string `json:"subcategory_id"`
	SubcategoryName *string `json:"subcategory_name"`
	SubcategoryIcon *string `json:"subcategory_icon"`
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
	Name        string
	Description *string
	Icon        *string
	WebsiteURL  *string
	Amount      int64
	AccountID   string
	Period      string
	Weekday     *int64
	DayOfMonth  *int64
	StartDate   time.Time
	TimeLocal   string
	Active      bool
}

type Summary struct {
	MonthlyTotal        int64          `json:"monthly_total"`
	MonthlyTotalDisplay string         `json:"monthly_total_display"`
	YearlyTotal         int64          `json:"yearly_total"`
	YearlyTotalDisplay  string         `json:"yearly_total_display"`
	Upcoming            []Subscription `json:"upcoming"`
}

type CandidateTransaction struct {
	ID              string   `json:"id"`
	AccountID       string   `json:"account_id"`
	AccountName     string   `json:"account_name"`
	Amount          int64    `json:"amount"`
	AmountDisplay   string   `json:"amount_display"`
	Description     *string  `json:"description"`
	CategoryID      *string  `json:"category_id"`
	CategoryName    *string  `json:"category_name"`
	SubcategoryID   *string  `json:"subcategory_id"`
	SubcategoryName *string  `json:"subcategory_name"`
	TransactionDate string   `json:"transaction_date"`
	Score           int      `json:"score"`
	MatchReasons    []string `json:"match_reasons"`
}

type CandidateResult struct {
	Items []CandidateTransaction `json:"items"`
	Total int                    `json:"total"`
}

type ConvertResult struct {
	Subscription Subscription `json:"subscription"`
}

type AttachResult struct {
	AttachedCount int      `json:"attached_count"`
	AttachedIDs   []string `json:"attached_ids"`
}

var (
	ErrNotFound          = errors.New("subscription not found")
	ErrInvalidName       = errors.New("invalid name")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrInvalidPeriod     = schedule.ErrInvalidPeriod
	ErrInvalidWeekday    = schedule.ErrInvalidWeekday
	ErrInvalidDay        = schedule.ErrInvalidDay
	ErrInvalidTime       = schedule.ErrInvalidTime
	ErrInvalidAccount    = errors.New("invalid account")
	ErrAccountArchived   = errors.New("account archived")
	ErrInvalidURL        = errors.New("invalid website url")
	ErrInvalidType       = errors.New("invalid type")
	ErrRecurringNotFound = errors.New("recurring operation not found")
	ErrAttachForbidden   = errors.New("transaction cannot be attached")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries { return sqlcdb.New(db) }

func fromRow(row sqlcdb.ListSubscriptionsByUserRow) Subscription {
	return Subscription{
		ID: row.ID, Name: row.Name, Description: row.Description, Icon: row.Icon, WebsiteURL: row.WebsiteUrl,
		Amount: row.Amount, AmountDisplay: money.FormatRubles(row.Amount),
		AccountID: row.AccountID, AccountName: row.AccountName,
		SubcategoryID: row.SubcategoryID, SubcategoryName: row.SubcategoryName, SubcategoryIcon: row.SubcategoryIcon,
		Period: row.Period, Weekday: row.Weekday, DayOfMonth: row.DayOfMonth,
		StartDate: row.StartDate, TimeLocal: row.TimeLocal, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt,
		Active: row.Active == 1, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func fromGet(row sqlcdb.GetSubscriptionByIDRow) Subscription {
	return Subscription{
		ID: row.ID, Name: row.Name, Description: row.Description, Icon: row.Icon, WebsiteURL: row.WebsiteUrl,
		Amount: row.Amount, AmountDisplay: money.FormatRubles(row.Amount),
		AccountID: row.AccountID, AccountName: row.AccountName,
		SubcategoryID: row.SubcategoryID, SubcategoryName: row.SubcategoryName, SubcategoryIcon: row.SubcategoryIcon,
		Period: row.Period, Weekday: row.Weekday, DayOfMonth: row.DayOfMonth,
		StartDate: row.StartDate, TimeLocal: row.TimeLocal, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt,
		Active: row.Active == 1, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func getByID(ctx context.Context, db *sql.DB, userID, id string) (Subscription, error) {
	row, err := queries(db).GetSubscriptionByID(ctx, sqlcdb.GetSubscriptionByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, err
	}
	return fromGet(row), nil
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Subscription, error) {
	rows, err := queries(db).ListSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Subscription, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in Input) (Subscription, error) {
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Subscription{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Subscription{}, err
	}
	now := timeutil.NowUTC()
	next, err := nextRun(in, tz, now)
	if err != nil {
		return Subscription{}, err
	}
	id := uuid.NewString()
	ts := timeutil.FormatUTC(now)
	if err := queries(db).InsertSubscription(ctx, sqlcdb.InsertSubscriptionParams{
		ID: id, UserID: userID, Name: strings.TrimSpace(in.Name), Description: in.Description,
		Icon: in.Icon, WebsiteUrl: in.WebsiteURL, Amount: in.Amount, AccountID: in.AccountID,
		SubcategoryID: nil, Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: timeutil.FormatUTC(in.StartDate), TimeLocal: in.TimeLocal,
		NextRunAt: next, LastRunAt: nil, Active: boolToInt(in.Active), CreatedAt: ts, UpdatedAt: ts,
	}); err != nil {
		return Subscription{}, err
	}
	return getByID(ctx, db, userID, id)
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in Input) (Subscription, error) {
	prev, err := getByID(ctx, db, userID, id)
	if err != nil {
		return Subscription{}, err
	}
	if err := validateInput(ctx, db, userID, in); err != nil {
		return Subscription{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Subscription{}, err
	}
	next, err := nextRun(in, tz, timeutil.NowUTC())
	if err != nil {
		return Subscription{}, err
	}
	newName := strings.TrimSpace(in.Name)
	subID := prev.SubcategoryID
	if prev.SubcategoryID != nil && *prev.SubcategoryID != "" && newName != prev.Name {
		sid, err := renameSubcategory(ctx, db, userID, id, *prev.SubcategoryID, prev.Name, newName, in.Icon)
		if err != nil {
			return Subscription{}, err
		}
		subID = sid
	} else if prev.SubcategoryID != nil && in.Icon != nil {
		_ = queries(db).UpdateSubcategory(ctx, sqlcdb.UpdateSubcategoryParams{
			Name: prev.Name, Icon: iconOrDefault(in.Icon), ID: *prev.SubcategoryID,
		})
	}
	n, err := queries(db).UpdateSubscription(ctx, sqlcdb.UpdateSubscriptionParams{
		Name: newName, Description: in.Description, Icon: in.Icon, WebsiteUrl: in.WebsiteURL,
		Amount: in.Amount, AccountID: in.AccountID, SubcategoryID: subID,
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: timeutil.FormatUTC(in.StartDate), TimeLocal: in.TimeLocal,
		NextRunAt: next, Active: boolToInt(in.Active), UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()),
		ID: id, UserID: userID,
	})
	if err != nil {
		return Subscription{}, err
	}
	if n == 0 {
		return Subscription{}, ErrNotFound
	}
	return getByID(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	n, err := queries(db).DeleteSubscription(ctx, sqlcdb.DeleteSubscriptionParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func SummaryForUser(ctx context.Context, db *sql.DB, userID string, upcomingDays int) (Summary, error) {
	if upcomingDays <= 0 {
		upcomingDays = 14
	}
	items, err := List(ctx, db, userID)
	if err != nil {
		return Summary{}, err
	}
	var monthly int64
	upcoming := make([]Subscription, 0)
	now := timeutil.NowUTC()
	until := now.AddDate(0, 0, upcomingDays)
	for _, s := range items {
		if !s.Active {
			continue
		}
		monthly += monthlyEquivalent(s.Amount, s.Period)
		if next, err := timeutil.ParseUTC(s.NextRunAt); err == nil {
			if !next.After(until) {
				upcoming = append(upcoming, s)
			}
		}
	}
	yearly := monthly * 12
	return Summary{
		MonthlyTotal: monthly, MonthlyTotalDisplay: money.FormatRubles(monthly),
		YearlyTotal: yearly, YearlyTotalDisplay: money.FormatRubles(yearly),
		Upcoming: upcoming,
	}, nil
}

func monthlyEquivalent(amount int64, period string) int64 {
	switch period {
	case "week":
		return amount * 52 / 12
	case "two_weeks":
		return amount * 26 / 12
	case "month":
		return amount
	case "quarter":
		return amount / 3
	case "half_year":
		return amount / 6
	case "year":
		return amount / 12
	default:
		return amount
	}
}

func ApplyDue(ctx context.Context, db *sql.DB, userID string, now time.Time, tz string) (int, error) {
	cutoff := timeutil.FormatUTC(now)
	due, err := queries(db).ListDueSubscriptions(ctx, sqlcdb.ListDueSubscriptionsParams{UserID: userID, NextRunAt: cutoff})
	if err != nil {
		return 0, err
	}
	if len(due) == 0 {
		return 0, nil
	}
	q := queries(db)
	applied := 0
	affected := map[string]struct{}{}
	var lastAsOf time.Time
	for _, op := range due {
		if _, err := q.GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: op.AccountID, UserID: userID}); err != nil {
			continue
		}
		subCatID, err := ensureSubcategory(ctx, db, userID, op.ID, op.Name, op.Icon)
		if err != nil {
			continue
		}
		catID, err := categoryseed.SubscriptionsCategoryID(ctx, db, userID)
		if err != nil {
			continue
		}
		txID := uuid.NewString()
		createdAt := timeutil.FormatUTC(now)
		if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
			ID: txID, UserID: userID, AccountID: op.AccountID, Type: "expense", Kind: "manual",
			Amount: op.Amount, Description: op.Description, CategoryID: &catID, SubcategoryID: &subCatID,
			TransferGroupID: nil, TransferAccountID: nil, TransactionDate: op.NextRunAt,
			AffectsBalance: 1, SubscriptionID: &op.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
		}); err != nil {
			continue
		}
		affected[op.AccountID] = struct{}{}
		if runAt, err := timeutil.ParseUTC(op.NextRunAt); err == nil {
			lastAsOf = runAt
		}
		next, err := nextRun(Input{
			Name: op.Name, Amount: op.Amount, AccountID: op.AccountID, Period: op.Period,
			Weekday: op.Weekday, DayOfMonth: op.DayOfMonth, StartDate: mustParse(op.StartDate),
			TimeLocal: op.TimeLocal, Active: true,
		}, tz, mustParse(op.NextRunAt).Add(time.Second))
		if err != nil {
			continue
		}
		_, _ = q.MarkSubscriptionRan(ctx, sqlcdb.MarkSubscriptionRanParams{
			NextRunAt: next, LastRunAt: strPtr(op.NextRunAt), SubcategoryID: &subCatID,
			UpdatedAt: createdAt, ID: op.ID, UserID: userID,
		})
		applied++
	}
	if len(affected) > 0 {
		ids := make([]string, 0, len(affected))
		for id := range affected {
			ids = append(ids, id)
		}
		if err := accountbalance.Refresh(ctx, db, userID, ids...); err != nil {
			return applied, err
		}
		balancehooks.NotifyRefresh(ctx, db, userID, lastAsOf, ids...)
	}
	return applied, nil
}

func AttachTransactions(ctx context.Context, db *sql.DB, userID, subscriptionID string, txIDs []string) (AttachResult, error) {
	sub, err := getByID(ctx, db, userID, subscriptionID)
	if err != nil {
		return AttachResult{}, err
	}
	subCatID, err := ensureSubcategory(ctx, db, userID, sub.ID, sub.Name, sub.Icon)
	if err != nil {
		return AttachResult{}, err
	}
	catID, err := categoryseed.SubscriptionsCategoryID(ctx, db, userID)
	if err != nil {
		return AttachResult{}, err
	}
	attached := make([]string, 0, len(txIDs))
	now := timeutil.FormatUTC(timeutil.NowUTC())
	q := queries(db)
	for _, txID := range txIDs {
		row, err := q.GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: txID, UserID: userID})
		if err != nil {
			continue
		}
		if row.Type != "expense" {
			continue
		}
		if row.SubscriptionID != nil {
			continue
		}
		if row.CategoryIsSystem != nil && *row.CategoryIsSystem == 1 {
			catName := ""
			if row.CategoryName != nil {
				catName = *row.CategoryName
			}
			if catName != categoryseed.SubscriptionsCategoryName {
				continue
			}
		}
		desc := row.Description
		if (desc == nil || strings.TrimSpace(*desc) == "") && sub.Description != nil {
			desc = sub.Description
		}
		n, err := q.AttachTransactionToSubscription(ctx, sqlcdb.AttachTransactionToSubscriptionParams{
			CategoryID: &catID, SubcategoryID: &subCatID, SubscriptionID: &subscriptionID,
			Description: desc, UpdatedAt: now, ID: txID, UserID: userID,
		})
		if err != nil || n == 0 {
			continue
		}
		attached = append(attached, txID)
	}
	return AttachResult{AttachedCount: len(attached), AttachedIDs: attached}, nil
}

func ConvertFromRecurring(ctx context.Context, db *sql.DB, userID, recurringID, name string, description *string, icon *string, websiteURL *string) (ConvertResult, error) {
	op, err := recurring.List(ctx, db, userID)
	if err != nil {
		return ConvertResult{}, err
	}
	var found *recurring.Operation
	for i := range op {
		if op[i].ID == recurringID {
			found = &op[i]
			break
		}
	}
	if found == nil {
		return ConvertResult{}, ErrRecurringNotFound
	}
	if found.Type != "expense" {
		return ConvertResult{}, ErrInvalidType
	}
	subName := strings.TrimSpace(name)
	if subName == "" {
		if found.Description != nil && strings.TrimSpace(*found.Description) != "" {
			subName = strings.TrimSpace(*found.Description)
		} else {
			subName = "Подписка"
		}
	}
	start, err := timeutil.ParseUTC(found.StartDate)
	if err != nil {
		return ConvertResult{}, err
	}
	timeLocal := strings.TrimSpace(found.TimeLocal)
	if timeLocal == "" {
		timeLocal = "08:00"
	}
	in := Input{
		Name: subName, Description: description, Icon: icon, WebsiteURL: websiteURL,
		Amount: found.Amount, AccountID: found.AccountID, Period: found.Period,
		Weekday: found.Weekday, DayOfMonth: found.DayOfMonth, StartDate: start,
		TimeLocal: timeLocal, Active: found.Active,
	}
	created, err := Create(ctx, db, userID, in)
	if err != nil {
		return ConvertResult{}, err
	}
	if err := recurring.Delete(ctx, db, userID, recurringID); err != nil {
		_ = Delete(ctx, db, userID, created.ID)
		return ConvertResult{}, err
	}
	return ConvertResult{Subscription: created}, nil
}

type CandidateFilter struct {
	Mode           string
	Query          string
	AccountID      *string
	AmountMin      *int64
	AmountMax      *int64
	DateFrom       *string
	DateTo         *string
	MinScore       int
	RefDescription *string
	Limit          int
	Offset         int
}

func ListCandidates(ctx context.Context, db *sql.DB, userID, subscriptionID string, f CandidateFilter) (CandidateResult, error) {
	sub, err := getByID(ctx, db, userID, subscriptionID)
	if err != nil {
		return CandidateResult{}, err
	}
	if f.Mode == "" {
		f.Mode = "auto"
	}
	if f.MinScore <= 0 {
		f.MinScore = 50
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
	var accountID, amountMin, amountMax, dateFrom, dateTo interface{}
	if f.AccountID != nil {
		accountID = *f.AccountID
	}
	if f.AmountMin != nil {
		amountMin = *f.AmountMin
	}
	if f.AmountMax != nil {
		amountMax = *f.AmountMax
	}
	if f.DateFrom != nil {
		dateFrom = *f.DateFrom
	}
	if f.DateTo != nil {
		dateTo = *f.DateTo
	}
	rows, err := queries(db).ListCandidateExpenseTransactions(ctx, sqlcdb.ListCandidateExpenseTransactionsParams{
		UserID: userID, AccountID: accountID, AmountMin: amountMin, AmountMax: amountMax,
		DateFrom: dateFrom, DateTo: dateTo,
	})
	if err != nil {
		return CandidateResult{}, err
	}
	ref := refText(sub.Name, sub.Description, f.RefDescription)
	items := make([]CandidateTransaction, 0, len(rows))
	for _, row := range rows {
		if row.CategoryIsSystem != nil && *row.CategoryIsSystem == 1 {
			name := ""
			if row.CategoryName != nil {
				name = *row.CategoryName
			}
			if name != categoryseed.SubscriptionsCategoryName {
				continue
			}
		}
		sc, reasons := scoreCandidate(scoreInput{
			TxAccountID: row.AccountID, TxAmount: row.Amount, TxDescription: row.Description,
			SubAccountID: sub.AccountID, SubAmount: sub.Amount, RefText: ref, Query: f.Query,
		})
		if f.Mode == "auto" && sc < f.MinScore {
			continue
		}
		items = append(items, CandidateTransaction{
			ID: row.ID, AccountID: row.AccountID, AccountName: row.AccountName,
			Amount: row.Amount, AmountDisplay: money.FormatRubles(row.Amount),
			Description: row.Description, CategoryID: row.CategoryID, CategoryName: row.CategoryName,
			SubcategoryID: row.SubcategoryID, SubcategoryName: row.SubcategoryName,
			TransactionDate: row.TransactionDate, Score: sc, MatchReasons: reasons,
		})
	}
	// sort by score desc, then date desc (already roughly date desc from SQL; stable re-sort)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Score > items[i].Score ||
				(items[j].Score == items[i].Score && items[j].TransactionDate > items[i].TransactionDate) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	total := len(items)
	if f.Offset > total {
		f.Offset = total
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}
	return CandidateResult{Items: items[f.Offset:end], Total: total}, nil
}

func ensureSubcategory(ctx context.Context, db *sql.DB, userID, subscriptionID, name string, icon *string) (string, error) {
	catID, err := categoryseed.SubscriptionsCategoryID(ctx, db, userID)
	if err != nil {
		return "", err
	}
	q := queries(db)
	existing, err := q.GetSubcategoryByCategoryAndName(ctx, sqlcdb.GetSubcategoryByCategoryAndNameParams{
		CategoryID: catID, Name: name, UserID: userID,
	})
	iconVal := iconOrDefault(icon)
	if err == nil {
		if existing.Icon != iconVal {
			_ = q.UpdateSubcategory(ctx, sqlcdb.UpdateSubcategoryParams{Name: name, Icon: iconVal, ID: existing.ID})
		}
		_, _ = q.SetSubscriptionSubcategory(ctx, sqlcdb.SetSubscriptionSubcategoryParams{
			SubcategoryID: &existing.ID, UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()),
			ID: subscriptionID, UserID: userID,
		})
		return existing.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	maxSortRaw, err := q.MaxSubcategorySortOrder(ctx, catID)
	if err != nil {
		return "", err
	}
	var maxSort int64
	switch v := maxSortRaw.(type) {
	case int64:
		maxSort = v
	case int:
		maxSort = int64(v)
	case float64:
		maxSort = int64(v)
	}
	id := uuid.NewString()
	if err := q.InsertSubcategory(ctx, sqlcdb.InsertSubcategoryParams{
		ID: id, CategoryID: catID, Name: name, Icon: iconVal, SortOrder: maxSort + 1,
		CreatedAt: timeutil.FormatUTC(timeutil.NowUTC()),
	}); err != nil {
		return "", err
	}
	_, _ = q.SetSubscriptionSubcategory(ctx, sqlcdb.SetSubscriptionSubcategoryParams{
		SubcategoryID: &id, UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()),
		ID: subscriptionID, UserID: userID,
	})
	return id, nil
}

func renameSubcategory(ctx context.Context, db *sql.DB, userID, subscriptionID, subID, oldName, newName string, icon *string) (*string, error) {
	count, err := queries(db).CountSubscriptionsBySubcategory(ctx, sqlcdb.CountSubscriptionsBySubcategoryParams{
		UserID: userID, SubcategoryID: &subID,
	})
	if err != nil {
		return nil, err
	}
	iconVal := iconOrDefault(icon)
	if count <= 1 {
		// sole owner — rename in place (or merge if target name exists)
		catID, err := categoryseed.SubscriptionsCategoryID(ctx, db, userID)
		if err != nil {
			return nil, err
		}
		existing, err := queries(db).GetSubcategoryByCategoryAndName(ctx, sqlcdb.GetSubcategoryByCategoryAndNameParams{
			CategoryID: catID, Name: newName, UserID: userID,
		})
		if err == nil && existing.ID != subID {
			return &existing.ID, nil
		}
		if err := queries(db).UpdateSubcategory(ctx, sqlcdb.UpdateSubcategoryParams{Name: newName, Icon: iconVal, ID: subID}); err != nil {
			return nil, err
		}
		return &subID, nil
	}
	// shared — split: clear link; next ensure creates/finds new name
	_, _ = queries(db).SetSubscriptionSubcategory(ctx, sqlcdb.SetSubscriptionSubcategoryParams{
		SubcategoryID: nil, UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()),
		ID: subscriptionID, UserID: userID,
	})
	return nil, nil
}

func validateInput(ctx context.Context, db *sql.DB, userID string, in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrInvalidName
	}
	if in.Amount <= 0 {
		return ErrInvalidAmount
	}
	if err := schedule.ValidatePeriodFields(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, true); err != nil {
		return err
	}
	if in.WebsiteURL != nil && strings.TrimSpace(*in.WebsiteURL) != "" {
		u, err := url.Parse(strings.TrimSpace(*in.WebsiteURL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return ErrInvalidURL
		}
	}
	acc, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: in.AccountID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if acc.Status != "active" {
		return ErrAccountArchived
	}
	return nil
}

func nextRun(in Input, tz string, ref time.Time) (string, error) {
	return schedule.NextRunAt(schedule.Input{
		Period: in.Period, Weekday: in.Weekday, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, TimeLocal: in.TimeLocal,
	}, tz, ref)
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

func iconOrDefault(icon *string) string {
	if icon == nil || strings.TrimSpace(*icon) == "" {
		return "subscription"
	}
	return strings.TrimSpace(*icon)
}
