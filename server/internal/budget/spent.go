package budget

import (
	"context"
	"database/sql"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

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
