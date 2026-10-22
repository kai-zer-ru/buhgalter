package categoryseed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const SubscriptionsLegacyName = "Подписки (старые)"

var subscriptionsSystemCategory = defaultCategory{
	Type: "expense", Name: SubscriptionsCategoryName, Icon: "subscription", Sort: 9997, IsSystem: true,
}

// EnsureSubscriptionsCategory ensures system expense «Подписки».
// If a user category with that name already exists, it is renamed to «Подписки (старые)»
// (or «Подписки (старые) N» on collision) and a new system category is created.
func EnsureSubscriptionsCategory(ctx context.Context, db sqlcdb.DBTX, userID string) error {
	q := sqlcdb.New(db)
	row, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: SubscriptionsCategoryName, Type: "expense",
	})
	if err == nil {
		if row.IsSystem == 1 {
			if row.Icon != subscriptionsSystemCategory.Icon {
				if err := q.UpdateSystemCategoryIcon(ctx, sqlcdb.UpdateSystemCategoryIconParams{
					Icon: subscriptionsSystemCategory.Icon, ID: row.ID, UserID: userID,
				}); err != nil {
					return err
				}
			}
			return nil
		}
		legacyName, err := uniqueSubscriptionsLegacyName(ctx, q, userID)
		if err != nil {
			return err
		}
		if err := q.UpdateCategory(ctx, sqlcdb.UpdateCategoryParams{
			Name: legacyName, Icon: row.Icon, SortOrder: row.SortOrder, ID: row.ID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("rename user category %q to %q: %w", SubscriptionsCategoryName, legacyName, err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: SubscriptionsCategoryName, Type: "expense",
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return insertCategory(ctx, q, userID, subscriptionsSystemCategory, now)
}

func uniqueSubscriptionsLegacyName(ctx context.Context, q *sqlcdb.Queries, userID string) (string, error) {
	n, err := q.CountCategoriesByName(ctx, sqlcdb.CountCategoriesByNameParams{
		UserID: userID, Name: SubscriptionsLegacyName, Type: "expense",
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return SubscriptionsLegacyName, nil
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s %d", SubscriptionsLegacyName, i)
		n, err := q.CountCategoriesByName(ctx, sqlcdb.CountCategoriesByNameParams{
			UserID: userID, Name: candidate, Type: "expense",
		})
		if err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot allocate legacy name for %q", SubscriptionsCategoryName)
}
