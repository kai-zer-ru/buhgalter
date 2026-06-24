package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type Category struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Type             string        `json:"type"`
	Icon             string        `json:"icon"`
	SortOrder        int           `json:"sort_order"`
	IsPrimary        bool          `json:"is_primary"`
	IsSystem         bool          `json:"is_system"`
	SubcategoryCount int           `json:"subcategory_count"`
	CreatedAt        string        `json:"created_at"`
	Subcategories    []Subcategory `json:"subcategories,omitempty"`
}

type Subcategory struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	SortOrder  int    `json:"sort_order"`
	CreatedAt  string `json:"created_at"`
}

var ErrNotFound = errors.New("category not found")
var ErrSubNotFound = errors.New("subcategory not found")
var ErrNameTaken = errors.New("category name already exists")
var ErrSubNameTaken = errors.New("subcategory name already exists")
var ErrInvalidName = errors.New("invalid category name")
var ErrInvalidReorder = errors.New("invalid category order")
var ErrSystemCategory = errors.New("system category is read-only")

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func categoryFromRow(
	id, name, catType, icon, createdAt string,
	sortOrder, isPrimary, isSystem, subCount int64,
) Category {
	return Category{
		ID:               id,
		Name:             name,
		Type:             catType,
		Icon:             icon,
		SortOrder:        int(sortOrder),
		IsPrimary:        isPrimary != 0,
		IsSystem:         isSystem != 0,
		SubcategoryCount: int(subCount),
		CreatedAt:        createdAt,
	}
}

func categoryFromGet(row sqlcdb.GetCategoryByIDRow) Category {
	return categoryFromRow(row.ID, row.Name, row.Type, row.Icon, row.CreatedAt, row.SortOrder, row.IsPrimary, row.IsSystem, row.SubCount)
}

func categoryFromList(row sqlcdb.ListCategoriesByUserRow) Category {
	return categoryFromRow(row.ID, row.Name, row.Type, row.Icon, row.CreatedAt, row.SortOrder, row.IsPrimary, row.IsSystem, row.SubCount)
}

func categoryFromListType(row sqlcdb.ListCategoriesByUserAndTypeRow) Category {
	return categoryFromRow(row.ID, row.Name, row.Type, row.Icon, row.CreatedAt, row.SortOrder, row.IsPrimary, row.IsSystem, row.SubCount)
}

func subcategoryFromRow(id, categoryID, name, icon, createdAt string, sortOrder int64) Subcategory {
	return Subcategory{
		ID:         id,
		CategoryID: categoryID,
		Name:       name,
		Icon:       icon,
		SortOrder:  int(sortOrder),
		CreatedAt:  createdAt,
	}
}

func maxSubSortOrder(ctx context.Context, q *sqlcdb.Queries, categoryID string) (int64, error) {
	v, err := q.MaxSubcategorySortOrder(ctx, categoryID)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, nil
	}
}

func maxSortOrder(ctx context.Context, q *sqlcdb.Queries, userID, catType string) (int64, error) {
	v, err := q.MaxCategorySortOrder(ctx, sqlcdb.MaxCategorySortOrderParams{
		UserID: userID,
		Type:   catType,
	})
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, nil
	}
}

func ListByUser(ctx context.Context, db *sql.DB, userID, catType string) ([]Category, error) {
	q := queries(db)
	if catType != "" {
		rows, err := q.ListCategoriesByUserAndType(ctx, sqlcdb.ListCategoriesByUserAndTypeParams{
			UserID: userID,
			Type:   catType,
		})
		if err != nil {
			return nil, err
		}
		out := make([]Category, 0, len(rows))
		for _, row := range rows {
			out = append(out, categoryFromListType(row))
		}
		return out, nil
	}

	rows, err := q.ListCategoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, categoryFromList(row))
	}
	return out, nil
}

func GetByID(ctx context.Context, db *sql.DB, userID, id string) (Category, error) {
	row, err := queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, err
	}
	return categoryFromGet(row), nil
}

func Create(ctx context.Context, db *sql.DB, userID, name, catType, icon string, sortOrder int) (Category, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return Category{}, ErrInvalidName
	}
	if catType != "income" && catType != "expense" {
		return Category{}, fmt.Errorf("invalid type")
	}
	if icon == "" {
		icon = "default"
	}
	if err := checkNameUnique(ctx, db, userID, name, catType, ""); err != nil {
		return Category{}, err
	}

	q := queries(db)
	if sortOrder <= 0 {
		max, err := maxSortOrder(ctx, q, userID, catType)
		if err != nil {
			return Category{}, err
		}
		sortOrder = int(max) + 1
	}

	count, err := q.CountCategoriesByType(ctx, sqlcdb.CountCategoriesByTypeParams{
		UserID: userID,
		Type:   catType,
	})
	if err != nil {
		return Category{}, err
	}
	isPrimary := int64(0)
	if count == 0 {
		isPrimary = 1
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Type:      catType,
		Icon:      icon,
		SortOrder: int64(sortOrder),
		IsPrimary: isPrimary,
		IsSystem:  0,
		CreatedAt: now,
	}); err != nil {
		return Category{}, err
	}
	return GetByID(ctx, db, userID, id)
}

func Update(ctx context.Context, db *sql.DB, userID, id, name, icon string, sortOrder *int) (Category, error) {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Category{}, err
	}
	if existing.IsSystem {
		return Category{}, ErrSystemCategory
	}
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return Category{}, ErrInvalidName
	}
	if err := checkNameUnique(ctx, db, userID, name, existing.Type, id); err != nil {
		return Category{}, err
	}
	if icon == "" {
		icon = existing.Icon
	}
	order := existing.SortOrder
	if sortOrder != nil {
		order = *sortOrder
	}
	if err := queries(db).UpdateCategory(ctx, sqlcdb.UpdateCategoryParams{
		Name:      name,
		Icon:      icon,
		SortOrder: int64(order),
		ID:        id,
		UserID:    userID,
	}); err != nil {
		return Category{}, err
	}
	return GetByID(ctx, db, userID, id)
}

func Reorder(ctx context.Context, db *sql.DB, userID, catType string, ids []string) error {
	if catType != "income" && catType != "expense" {
		return ErrInvalidReorder
	}
	existing, err := ListByUser(ctx, db, userID, catType)
	if err != nil {
		return err
	}
	if len(ids) != len(existing) {
		return ErrInvalidReorder
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return ErrInvalidReorder
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidReorder
		}
		seen[id] = struct{}{}
	}
	for _, c := range existing {
		if _, ok := seen[c.ID]; !ok {
			return ErrInvalidReorder
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	for i, id := range ids {
		if err := q.UpdateCategorySortOrder(ctx, sqlcdb.UpdateCategorySortOrderParams{
			SortOrder: int64(i + 1),
			ID:        id,
			UserID:    userID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func SetPrimary(ctx context.Context, db *sql.DB, userID, id string) (Category, error) {
	cat, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Category{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Category{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	if err := q.ClearPrimaryCategories(ctx, sqlcdb.ClearPrimaryCategoriesParams{
		UserID: userID,
		Type:   cat.Type,
	}); err != nil {
		return Category{}, err
	}
	if err := q.SetCategoryPrimary(ctx, sqlcdb.SetCategoryPrimaryParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return Category{}, err
	}
	if err := tx.Commit(); err != nil {
		return Category{}, err
	}
	return GetByID(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	cat, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return err
	}
	if cat.IsSystem {
		return ErrSystemCategory
	}
	wasPrimary := cat.IsPrimary
	catType := cat.Type

	n, err := queries(db).DeleteCategory(ctx, sqlcdb.DeleteCategoryParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}

	if !wasPrimary {
		return nil
	}

	remaining, err := ListByUser(ctx, db, userID, catType)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return nil
	}
	_, err = SetPrimary(ctx, db, userID, remaining[0].ID)
	return err
}

func checkNameUnique(ctx context.Context, db *sql.DB, userID, name, catType, excludeID string) error {
	q := queries(db)
	var n int64
	var err error
	if excludeID != "" {
		n, err = q.CountCategoriesByNameExcluding(ctx, sqlcdb.CountCategoriesByNameExcludingParams{
			UserID: userID,
			Name:   name,
			Type:   catType,
			ID:     excludeID,
		})
	} else {
		n, err = q.CountCategoriesByName(ctx, sqlcdb.CountCategoriesByNameParams{
			UserID: userID,
			Name:   name,
			Type:   catType,
		})
	}
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrNameTaken
	}
	return nil
}

func ListSubcategories(ctx context.Context, db *sql.DB, userID, categoryID string) ([]Subcategory, error) {
	if _, err := GetByID(ctx, db, userID, categoryID); err != nil {
		return nil, err
	}
	rows, err := queries(db).ListSubcategoriesByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	out := make([]Subcategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, subcategoryFromRow(row.ID, row.CategoryID, row.Name, row.Icon, row.CreatedAt, row.SortOrder))
	}
	return out, nil
}

func ReorderSubcategories(ctx context.Context, db *sql.DB, userID, categoryID string, ids []string) ([]Subcategory, error) {
	if _, err := GetByID(ctx, db, userID, categoryID); err != nil {
		return nil, err
	}
	existing, err := ListSubcategories(ctx, db, userID, categoryID)
	if err != nil {
		return nil, err
	}
	if len(ids) != len(existing) {
		return nil, ErrInvalidReorder
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, ErrInvalidReorder
		}
		if _, ok := seen[id]; ok {
			return nil, ErrInvalidReorder
		}
		seen[id] = struct{}{}
	}
	for _, s := range existing {
		if _, ok := seen[s.ID]; !ok {
			return nil, ErrInvalidReorder
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	for i, id := range ids {
		if err := q.UpdateSubcategorySortOrder(ctx, sqlcdb.UpdateSubcategorySortOrderParams{
			SortOrder: int64(i + 1),
			ID:        id,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ListSubcategories(ctx, db, userID, categoryID)
}

func CreateSubcategory(ctx context.Context, db *sql.DB, userID, categoryID, name, icon string) (Subcategory, error) {
	parent, err := GetByID(ctx, db, userID, categoryID)
	if err != nil {
		return Subcategory{}, err
	}
	if parent.IsSystem {
		return Subcategory{}, ErrSystemCategory
	}
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return Subcategory{}, ErrInvalidName
	}
	if icon == "" {
		icon = parent.Icon
	}
	if icon == "" {
		icon = "default"
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	max, err := maxSubSortOrder(ctx, q, categoryID)
	if err != nil {
		return Subcategory{}, err
	}
	if err := q.InsertSubcategory(ctx, sqlcdb.InsertSubcategoryParams{
		ID:         id,
		CategoryID: categoryID,
		Name:       name,
		Icon:       icon,
		SortOrder:  max + 1,
		CreatedAt:  now,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Subcategory{}, ErrSubNameTaken
		}
		return Subcategory{}, err
	}
	return GetSubcategory(ctx, db, userID, id)
}

func GetSubcategory(ctx context.Context, db *sql.DB, userID, id string) (Subcategory, error) {
	row, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Subcategory{}, ErrSubNotFound
	}
	if err != nil {
		return Subcategory{}, err
	}
	return subcategoryFromRow(row.ID, row.CategoryID, row.Name, row.Icon, row.CreatedAt, row.SortOrder), nil
}

func UpdateSubcategory(ctx context.Context, db *sql.DB, userID, id, name, icon string) (Subcategory, error) {
	existing, err := GetSubcategory(ctx, db, userID, id)
	if err != nil {
		return Subcategory{}, err
	}
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return Subcategory{}, ErrInvalidName
	}
	if icon == "" {
		icon = existing.Icon
	}
	if err := queries(db).UpdateSubcategory(ctx, sqlcdb.UpdateSubcategoryParams{Name: name, Icon: icon, ID: id}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Subcategory{}, ErrSubNameTaken
		}
		return Subcategory{}, err
	}
	return GetSubcategory(ctx, db, userID, id)
}

func DeleteSubcategory(ctx context.Context, db *sql.DB, userID, id string) error {
	if _, err := GetSubcategory(ctx, db, userID, id); err != nil {
		return err
	}
	n, err := queries(db).DeleteSubcategory(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSubNotFound
	}
	return nil
}
