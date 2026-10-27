package household

import (
	"context"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func copyOwnerCatalog(ctx context.Context, q *sqlcdb.Queries, ownerID, principalID string) error {
	cats, err := q.ListCategoriesByUser(ctx, []string{ownerID})
	if err != nil {
		return err
	}
	idMap := make(map[string]string, len(cats))
	for _, c := range cats {
		newID := uuid.NewString()
		idMap[c.ID] = newID
		if err := q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
			ID:        newID,
			UserID:    principalID,
			Name:      c.Name,
			Type:      c.Type,
			Icon:      c.Icon,
			SortOrder: c.SortOrder,
			IsPrimary: c.IsPrimary,
			IsSystem:  c.IsSystem,
			CreatedAt: c.CreatedAt,
		}); err != nil {
			return err
		}
	}
	subs, err := q.ListSubcategoriesByUser(ctx, []string{ownerID})
	if err != nil {
		return err
	}
	for _, s := range subs {
		newCatID, ok := idMap[s.CategoryID]
		if !ok {
			continue
		}
		if err := q.InsertSubcategory(ctx, sqlcdb.InsertSubcategoryParams{
			ID:         uuid.NewString(),
			CategoryID: newCatID,
			Name:       s.Name,
			Icon:       s.Icon,
			SortOrder:  s.SortOrder,
			CreatedAt:  s.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}
