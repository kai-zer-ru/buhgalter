package budget

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
)

// SpentPreview is the fact amount for a budget scope without an existing budget row.
type SpentPreview struct {
	Spent        int64  `json:"spent"`
	SpentDisplay string `json:"spent_display"`
}

// SpentPreviewInput selects scope/month filters for PreviewSpent.
type SpentPreviewInput struct {
	Month         string
	Scope         string
	CategoryID    *string
	SubcategoryID *string
	AccountID     *string
}

// PreviewSpent returns expense total for the given scope in a calendar month
// (same filters as budget summary spent).
func PreviewSpent(ctx context.Context, db *sql.DB, userID string, in SpentPreviewInput) (SpentPreview, error) {
	month, err := resolveMonth(ctx, db, userID, in.Month)
	if err != nil {
		return SpentPreview{}, err
	}
	scopeIn := Input{
		Scope:         in.Scope,
		CategoryID:    in.CategoryID,
		SubcategoryID: in.SubcategoryID,
		AccountID:     in.AccountID,
		Month:         month,
		IsActive:      true,
	}
	scopeIn, err = resolveScopeRefs(ctx, db, userID, scopeIn)
	if err != nil {
		return SpentPreview{}, err
	}
	if err := validateSpentPreviewInput(ctx, db, userID, scopeIn); err != nil {
		return SpentPreview{}, err
	}
	periodStart, periodEnd, err := MonthBounds(ctx, db, userID, month)
	if err != nil {
		return SpentPreview{}, err
	}
	spent, err := spentForBudget(
		ctx, db, userID, scopeIn.Scope,
		scopeIn.CategoryID, scopeIn.SubcategoryID, scopeIn.AccountID,
		periodStart, periodEnd,
	)
	if err != nil {
		return SpentPreview{}, err
	}
	return SpentPreview{
		Spent:        spent,
		SpentDisplay: money.FormatRubles(spent),
	}, nil
}

func validateSpentPreviewInput(ctx context.Context, db *sql.DB, userID string, in Input) error {
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

func spentForBudget(
	ctx context.Context,
	db *sql.DB,
	userID string,
	scope string,
	categoryID, subcategoryID, accountID *string,
	periodStart, periodEndExclusive string,
) (int64, error) {
	accCol := ""
	accVal := ""
	if accountID != nil && *accountID != "" {
		accCol = *accountID
		accVal = *accountID
	}
	catCol := ""
	var catPtr *string
	subCol := ""
	var subPtr *string
	switch scope {
	case ScopeCategory:
		if categoryID != nil && *categoryID != "" {
			catCol = *categoryID
			catPtr = categoryID
		}
	case ScopeSubcategory:
		if subcategoryID != nil && *subcategoryID != "" {
			subCol = *subcategoryID
			subPtr = subcategoryID
		}
	case ScopeAllExpense:
	default:
		return 0, ErrInvalidScope
	}
	return queries(db).BudgetSpent(ctx, sqlcdb.BudgetSpentParams{
		UserID:            userID,
		Column2:           accCol,
		AccountID:         accVal,
		Column4:           catCol,
		CategoryID:        catPtr,
		Column6:           subCol,
		SubcategoryID:     subPtr,
		TransactionDate:   periodStart,
		TransactionDate_2: periodEndExclusive,
	})
}
