package stats

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"sort"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var ErrInvalidDate = errors.New("invalid stats date")

type Filters struct {
	From          string
	To            string
	Type          string
	AccountID     string
	CategoryID    string
	Kind          string
	Search        string
	IncludeFuture bool
}

type Summary struct {
	IncomeTotal      int64 `json:"income_total"`
	ExpenseTotal     int64 `json:"expense_total"`
	BalanceDelta     int64 `json:"balance_delta"`
	TransactionCount int64 `json:"transaction_count"`
}

type CategoryItem struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Icon         string  `json:"icon"`
	Type         string  `json:"type"`
	Total        int64   `json:"total"`
	Percentage   float64 `json:"percentage"`
	Count        int64   `json:"count"`
}

type SubcategoryItem struct {
	CategoryID      string  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	CategoryIcon    string  `json:"category_icon"`
	SubcategoryID   string  `json:"subcategory_id"`
	SubcategoryName string  `json:"subcategory_name"`
	Total           int64   `json:"total"`
	Percentage      float64 `json:"percentage"`
	Count           int64   `json:"count"`
}

type PeriodItem struct {
	Period  string `json:"period"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

type ContextSummary struct {
	Summary
	Scope           string `json:"scope"`
	ScopeID         string `json:"scope_id,omitempty"`
	LentTotal       *int64 `json:"lent_total,omitempty"`
	BorrowedTotal   *int64 `json:"borrowed_total,omitempty"`
	PaidTotal       *int64 `json:"paid_total,omitempty"`
	PaymentCount    *int64 `json:"payment_count,omitempty"`
	RemainingAmount *int64 `json:"remaining_amount,omitempty"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func (s *Service) Summary(ctx context.Context, userID string, f Filters, factualOnly bool) (Summary, error) {
	if err := transaction.ActivateDueFutureTransactions(ctx, s.db, userID); err != nil {
		return Summary{}, err
	}
	f, err := normalizeFilters(f, factualOnly)
	if err != nil {
		return Summary{}, err
	}
	var income, expense, count int64
	if f.AccountID != "" {
		row, err := queries(s.db).StatsSummaryAccount(ctx, toAccountSummaryParams(userID, f))
		if err != nil {
			return Summary{}, err
		}
		income, expense, count = row.IncomeTotal, row.ExpenseTotal, row.TransactionCount
	} else {
		row, err := queries(s.db).StatsSummary(ctx, toSummaryParams(userID, f))
		if err != nil {
			return Summary{}, err
		}
		income, expense, count = row.IncomeTotal, row.ExpenseTotal, row.TransactionCount
	}
	return Summary{
		IncomeTotal:      income,
		ExpenseTotal:     expense,
		BalanceDelta:     income - expense,
		TransactionCount: count,
	}, nil
}

func (s *Service) ByCategory(ctx context.Context, userID string, f Filters) ([]CategoryItem, error) {
	f, err := normalizeFilters(f, true)
	if err != nil {
		return nil, err
	}
	rows, err := queries(s.db).StatsByCategory(ctx, toCategoryParams(userID, f))
	if err != nil {
		return nil, err
	}
	totalsByType := map[string]int64{}
	for _, row := range rows {
		totalsByType[row.CategoryType] += row.Total
	}
	out := make([]CategoryItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, CategoryItem{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Icon:         row.CategoryIcon,
			Type:         row.CategoryType,
			Total:        row.Total,
			Percentage:   percentage(totalsByType[row.CategoryType], row.Total),
			Count:        row.TxCount,
		})
	}
	return out, nil
}

func (s *Service) BySubcategory(ctx context.Context, userID string, f Filters) ([]SubcategoryItem, error) {
	f, err := normalizeFilters(f, true)
	if err != nil {
		return nil, err
	}
	rows, err := queries(s.db).StatsBySubcategory(ctx, toSubcategoryParams(userID, f))
	if err != nil {
		return nil, err
	}
	var total int64
	for _, row := range rows {
		total += row.Total
	}
	out := make([]SubcategoryItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, SubcategoryItem{
			CategoryID:      row.CategoryID,
			CategoryName:    row.CategoryName,
			CategoryIcon:    row.CategoryIcon,
			SubcategoryID:   row.SubcategoryID,
			SubcategoryName: row.SubcategoryName,
			Total:           row.Total,
			Percentage:      percentage(total, row.Total),
			Count:           row.TxCount,
		})
	}
	return out, nil
}

func (s *Service) ByPeriod(ctx context.Context, userID string, groupBy string, f Filters) ([]PeriodItem, error) {
	f, err := normalizeFilters(f, true)
	if err != nil {
		return nil, err
	}
	tz, err := queries(s.db).GetUserTimezone(ctx, userID)
	if err != nil {
		return nil, err
	}
	if tz == "" {
		tz = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	rows, err := queries(s.db).StatsPeriodRows(ctx, toPeriodParams(userID, f))
	if err != nil {
		return nil, err
	}
	m := map[string]*PeriodItem{}
	for _, row := range rows {
		ts, err := timeutil.ParseUTC(row.TransactionDate)
		if err != nil {
			return nil, ErrInvalidDate
		}
		key := periodKey(ts.In(loc), groupBy)
		item, ok := m[key]
		if !ok {
			item = &PeriodItem{Period: key}
			m[key] = item
		}
		switch row.Type {
		case "income":
			item.Income += row.Amount
		case "expense":
			item.Expense += row.Amount
		}
	}
	out := make([]PeriodItem, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Period > out[j].Period
	})
	return out, nil
}

func (s *Service) ContextDefault(ctx context.Context, userID string, f Filters) (ContextSummary, error) {
	summary, err := s.Summary(ctx, userID, f, contextFactualOnly(f))
	if err != nil {
		return ContextSummary{}, err
	}
	return ContextSummary{Summary: summary, Scope: "all"}, nil
}

func (s *Service) ContextAccount(ctx context.Context, userID, accountID string, f Filters) (ContextSummary, error) {
	f.AccountID = accountID
	summary, err := s.Summary(ctx, userID, f, contextFactualOnly(f))
	if err != nil {
		return ContextSummary{}, err
	}
	return ContextSummary{Summary: summary, Scope: "account", ScopeID: accountID}, nil
}

func (s *Service) ContextDebtor(ctx context.Context, userID, debtorID string, f Filters) (ContextSummary, error) {
	f, err := normalizeFilters(f, false)
	if err != nil {
		return ContextSummary{}, err
	}
	row, err := queries(s.db).StatsContextDebtor(ctx, sqlcdb.StatsContextDebtorParams{
		UserID:            userID,
		UserID_2:          userID,
		DebtorID:          debtorID,
		Column4:           f.Kind,
		Kind:              f.Kind,
		Column6:           f.From,
		TransactionDate:   f.From,
		Column8:           f.To,
		TransactionDate_2: f.To,
	})
	if err != nil {
		return ContextSummary{}, err
	}
	lent := row.LentTotal
	borrowed := row.BorrowedTotal
	return ContextSummary{
		Summary: Summary{
			IncomeTotal:      row.IncomeTotal,
			ExpenseTotal:     row.ExpenseTotal,
			BalanceDelta:     row.IncomeTotal - row.ExpenseTotal,
			TransactionCount: row.TransactionCount,
		},
		Scope:         "debtor",
		ScopeID:       debtorID,
		LentTotal:     &lent,
		BorrowedTotal: &borrowed,
	}, nil
}

func (s *Service) ContextCredit(ctx context.Context, userID, creditID string, f Filters) (ContextSummary, error) {
	f, err := normalizeFilters(f, false)
	if err != nil {
		return ContextSummary{}, err
	}
	row, err := queries(s.db).StatsContextCredit(ctx, sqlcdb.StatsContextCreditParams{
		UserID:            userID,
		UserID_2:          userID,
		ID:                creditID,
		Column4:           f.Kind,
		Kind:              f.Kind,
		Column6:           f.From,
		TransactionDate:   f.From,
		Column8:           f.To,
		TransactionDate_2: f.To,
	})
	if err != nil {
		return ContextSummary{}, err
	}
	paidRow, err := queries(s.db).StatsContextCreditPaid(ctx, sqlcdb.StatsContextCreditPaidParams{
		UserID:         userID,
		ID:             creditID,
		Column3:        f.From,
		PaymentDate:    f.From,
		Column5:        f.To,
		PaymentDate_2:  f.To,
	})
	if err != nil {
		return ContextSummary{}, err
	}
	remainingRow, err := queries(s.db).StatsContextCreditRemaining(ctx, sqlcdb.StatsContextCreditRemainingParams{
		ID:     creditID,
		UserID: userID,
	})
	if err != nil {
		return ContextSummary{}, err
	}
	remaining := remainingRow.PrincipalAmount - remainingRow.PaidAmount
	paid := paidRow.PaidTotal
	paymentCount := paidRow.PaymentCount
	return ContextSummary{
		Summary: Summary{
			IncomeTotal:      row.IncomeTotal,
			ExpenseTotal:     row.ExpenseTotal,
			BalanceDelta:     row.IncomeTotal - row.ExpenseTotal,
			TransactionCount: row.TransactionCount,
		},
		Scope:           "credit",
		ScopeID:         creditID,
		PaidTotal:       &paid,
		PaymentCount:    &paymentCount,
		RemainingAmount: &remaining,
	}, nil
}

func (s *Service) ContextDebts(ctx context.Context, userID string, f Filters) (ContextSummary, error) {
	f, err := normalizeFilters(f, false)
	if err != nil {
		return ContextSummary{}, err
	}
	row, err := queries(s.db).StatsContextDebts(ctx, sqlcdb.StatsContextDebtsParams{
		UserID:            userID,
		UserID_2:          userID,
		Column3:           f.Kind,
		Kind:              f.Kind,
		Column5:           f.From,
		TransactionDate:   f.From,
		Column7:           f.To,
		TransactionDate_2: f.To,
	})
	if err != nil {
		return ContextSummary{}, err
	}
	return ContextSummary{
		Summary: Summary{
			IncomeTotal:      row.IncomeTotal,
			ExpenseTotal:     row.ExpenseTotal,
			BalanceDelta:     row.IncomeTotal - row.ExpenseTotal,
			TransactionCount: row.TransactionCount,
		},
		Scope: "debts",
	}, nil
}

func percentage(total, value int64) float64 {
	if total <= 0 {
		return 0
	}
	v := float64(value) * 100 / float64(total)
	return math.Round(v*10) / 10
}

// contextFactualOnly mirrors transaction list filters: «плановые» → only future, otherwise factual.
func contextFactualOnly(f Filters) bool {
	if f.IncludeFuture || f.Kind == "future" {
		return false
	}
	return true
}

func normalizeFilters(f Filters, factualOnly bool) (Filters, error) {
	if f.From != "" {
		if _, err := timeutil.ParseUTC(f.From); err != nil {
			return Filters{}, ErrInvalidDate
		}
	}
	if f.To != "" {
		if _, err := timeutil.ParseUTC(f.To); err != nil {
			return Filters{}, ErrInvalidDate
		}
	}
	if factualOnly && !f.IncludeFuture {
		if f.Kind == "" {
			f.Kind = "manual"
		}
		now := timeutil.FormatUTC(timeutil.NowUTC())
		if f.To == "" || f.To > now {
			f.To = now
		}
	}
	return f, nil
}

func periodKey(t time.Time, groupBy string) string {
	switch groupBy {
	case "week":
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := t.AddDate(0, 0, -(weekday - 1))
		return weekStart.Format("2006-01-02")
	case "month":
		monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		return monthStart.Format("2006-01-02")
	default:
		return t.Format("2006-01-02")
	}
}

func toSummaryParams(userID string, f Filters) sqlcdb.StatsSummaryParams {
	search := strPtr(f.Search)
	categoryID := strPtr(f.CategoryID)
	return sqlcdb.StatsSummaryParams{
		UserID:            userID,
		Column2:           f.AccountID,
		AccountID:         f.AccountID,
		Column4:           f.CategoryID,
		CategoryID:        categoryID,
		Column6:           f.Type,
		Type:              f.Type,
		Column8:           f.Kind,
		Kind:              f.Kind,
		Column10:          f.From,
		TransactionDate:   f.From,
		Column12:          f.To,
		TransactionDate_2: f.To,
		Column14:          f.Search,
		Column15:          search,
	}
}

func toAccountSummaryParams(userID string, f Filters) sqlcdb.StatsSummaryAccountParams {
	search := strPtr(f.Search)
	categoryID := strPtr(f.CategoryID)
	return sqlcdb.StatsSummaryAccountParams{
		UserID:            userID,
		AccountID:         f.AccountID,
		Column3:           f.CategoryID,
		CategoryID:        categoryID,
		Column5:           f.Type,
		Type:              f.Type,
		Column7:           f.Kind,
		Kind:              f.Kind,
		Column9:           f.From,
		TransactionDate:   f.From,
		Column11:          f.To,
		TransactionDate_2: f.To,
		Column13:          f.Search,
		Column14:          search,
	}
}

func toCategoryParams(userID string, f Filters) sqlcdb.StatsByCategoryParams {
	p := toSummaryParams(userID, f)
	return sqlcdb.StatsByCategoryParams(p)
}

func toSubcategoryParams(userID string, f Filters) sqlcdb.StatsBySubcategoryParams {
	p := toSummaryParams(userID, f)
	return sqlcdb.StatsBySubcategoryParams(p)
}

func toPeriodParams(userID string, f Filters) sqlcdb.StatsPeriodRowsParams {
	p := toSummaryParams(userID, f)
	return sqlcdb.StatsPeriodRowsParams(p)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
