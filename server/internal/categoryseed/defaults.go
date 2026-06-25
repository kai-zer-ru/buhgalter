package categoryseed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const DebtCategoryName = "Долги"
const CreditCategoryName = "Кредиты"
const CommissionCategoryName = "Комиссия"

type defaultCategory struct {
	Type      string
	Name      string
	Icon      string
	Sort      int
	IsPrimary bool
	IsSystem  bool
}

// DefaultCount is the number of categories seeded for a new user (including system).
const DefaultCount = 11

var defaultCategories = []defaultCategory{
	{Type: "expense", Name: "Транспорт", Icon: "transport", Sort: 1, IsPrimary: true},
	{Type: "expense", Name: "Магазины", Icon: "food", Sort: 2},
	{Type: "expense", Name: "Связь", Icon: "phone", Sort: 3},
	{Type: "expense", Name: "Здоровье", Icon: "health", Sort: 4},
	{Type: "expense", Name: "Разное", Icon: "default", Sort: 5},
	{Type: "income", Name: "Зарплата", Icon: "salary", Sort: 1, IsPrimary: true},
	{Type: "income", Name: "Прочие доходы", Icon: "default", Sort: 2},
}

var systemCategories = []defaultCategory{
	{Type: "expense", Name: CommissionCategoryName, Icon: "percent", Sort: 9997, IsSystem: true},
	{Type: "expense", Name: CreditCategoryName, Icon: "loan", Sort: 9998, IsSystem: true},
	{Type: "expense", Name: DebtCategoryName, Icon: "loan", Sort: 9999, IsSystem: true},
	{Type: "income", Name: DebtCategoryName, Icon: "loan", Sort: 9999, IsSystem: true},
}

// SeedDefaults inserts default income/expense categories for a new user.
func SeedDefaults(ctx context.Context, db sqlcdb.DBTX, userID string) error {
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range defaultCategories {
		if err := insertCategory(ctx, q, userID, c, now); err != nil {
			return fmt.Errorf("seed category %q: %w", c.Name, err)
		}
	}
	return EnsureSystemCategories(ctx, db, userID)
}

// EnsureSystemCategories creates or marks system «Долги» categories for a user.
func EnsureSystemCategories(ctx context.Context, db sqlcdb.DBTX, userID string) error {
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range systemCategories {
		row, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
			UserID: userID, Name: c.Name, Type: c.Type,
		})
		if err == nil {
			if row.IsSystem == 0 {
				if err := q.SetCategorySystem(ctx, sqlcdb.SetCategorySystemParams{
					IsSystem: 1, ID: row.ID, UserID: userID,
				}); err != nil {
					return err
				}
			}
			if row.Icon != c.Icon {
				if err := q.UpdateSystemCategoryIcon(ctx, sqlcdb.UpdateSystemCategoryIconParams{
					Icon: c.Icon, ID: row.ID, UserID: userID,
				}); err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := insertCategory(ctx, q, userID, c, now); err != nil {
			return fmt.Errorf("seed system category %q: %w", c.Name, err)
		}
	}
	return nil
}

// BackfillSystemCategories ensures system categories for all existing users.
func BackfillSystemCategories(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT id FROM users`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		if err := EnsureSystemCategories(ctx, db, userID); err != nil {
			return err
		}
	}
	return rows.Err()
}

// DebtCategoryID returns the system «Долги» category id for the given type.
func DebtCategoryID(ctx context.Context, db sqlcdb.DBTX, userID, catType string) (string, error) {
	if err := EnsureSystemCategories(ctx, db, userID); err != nil {
		return "", err
	}
	row, err := sqlcdb.New(db).GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: DebtCategoryName, Type: catType,
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// CommissionCategoryID returns the system expense «Комиссия» category id.
func CommissionCategoryID(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	if err := EnsureSystemCategories(ctx, db, userID); err != nil {
		return "", err
	}
	row, err := sqlcdb.New(db).GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: CommissionCategoryName, Type: "expense",
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// CreditCategoryID returns the system expense «Кредиты» category id.
func CreditCategoryID(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	if err := EnsureSystemCategories(ctx, db, userID); err != nil {
		return "", err
	}
	row, err := sqlcdb.New(db).GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: CreditCategoryName, Type: "expense",
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func insertCategory(ctx context.Context, q *sqlcdb.Queries, userID string, c defaultCategory, now string) error {
	isPrimary := int64(0)
	if c.IsPrimary {
		isPrimary = 1
	}
	isSystem := int64(0)
	if c.IsSystem {
		isSystem = 1
	}
	return q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      c.Name,
		Type:      c.Type,
		Icon:      c.Icon,
		SortOrder: int64(c.Sort),
		IsPrimary: isPrimary,
		IsSystem:  isSystem,
		CreatedAt: now,
	})
}
