package categoryrule

import (
	"context"
	"database/sql"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// SuggestDefaultAccount returns default_account_id from the first matching
// category_account rule (priority ASC, id ASC). If subcategoryID is nil,
// rules with subcategory_id NULL match any subcategory of the category; rules
// with a specific subcategory_id require an exact match.
func SuggestDefaultAccount(ctx context.Context, db *sql.DB, userID, categoryID string, subcategoryID *string) (*string, error) {
	rows, err := queries(db).ListCategoryRulesByUserAndKind(ctx, sqlcdb.ListCategoryRulesByUserAndKindParams{
		OwnerIds: []string{userID},
		Kind:     KindCategoryAccount,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.CategoryID != categoryID {
			continue
		}
		if row.SubcategoryID != nil {
			if subcategoryID == nil || *row.SubcategoryID != *subcategoryID {
				continue
			}
		}
		if row.DefaultAccountID != nil && *row.DefaultAccountID != "" {
			id := *row.DefaultAccountID
			return &id, nil
		}
	}
	return nil, nil
}
