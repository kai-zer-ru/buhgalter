package receiptcategory

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type lineAssignment struct {
	categoryID    string
	subcategoryID string
}

// DisplayLabels applies receipt line aggregation rules for journal category cells:
// one category + different subcategories → category only;
// multiple categories → all category names;
// one category + one subcategory on all lines → category and subcategory.
func DisplayLabels(ctx context.Context, db *sql.DB, userID, receiptImportID string) (categoryName, categoryIcon, subcategoryName, subcategoryIcon *string) {
	if strings.TrimSpace(receiptImportID) == "" {
		return nil, nil, nil, nil
	}
	lines, err := collectLineAssignments(ctx, db, userID, receiptImportID)
	if err != nil || len(lines) == 0 {
		return nil, nil, nil, nil
	}

	catSet := map[string]struct{}{}
	subSet := map[string]struct{}{}
	for _, ln := range lines {
		catSet[ln.categoryID] = struct{}{}
		if sub := strings.TrimSpace(ln.subcategoryID); sub != "" {
			subSet[sub] = struct{}{}
		}
	}

	resolved := make([]resolvedLine, 0, len(lines))
	for _, ln := range lines {
		cat, err := category.GetByID(ctx, db, userID, ln.categoryID)
		if err != nil {
			continue
		}
		rl := resolvedLine{categoryName: cat.Name}
		subID := strings.TrimSpace(ln.subcategoryID)
		if subID != "" {
			sub, err := category.GetSubcategory(ctx, db, userID, subID)
			if err == nil {
				rl.subcategoryName = sub.Name
			}
		}
		resolved = append(resolved, rl)
	}
	if len(resolved) == 0 {
		return nil, nil, nil, nil
	}
	cn, sn := aggregateDisplay(resolved)
	if cn == "" {
		return nil, nil, nil, nil
	}
	name := cn
	var ci *string
	// Icon from the sole category when display shows one category label.
	if len(catSet) == 1 {
		var onlyCatID string
		for id := range catSet {
			onlyCatID = id
		}
		if cat, err := category.GetByID(ctx, db, userID, onlyCatID); err == nil && cat.Icon != "" {
			icon := cat.Icon
			ci = &icon
		}
	}
	if sn == nil {
		return &name, ci, nil, nil
	}
	var si *string
	if len(subSet) == 1 {
		var onlySubID string
		for id := range subSet {
			onlySubID = id
		}
		if sub, err := category.GetSubcategory(ctx, db, userID, onlySubID); err == nil && sub.Icon != "" {
			icon := sub.Icon
			si = &icon
		}
	}
	return &name, ci, sn, si
}

type resolvedLine struct {
	categoryName    string
	subcategoryName string
}

func aggregateDisplay(lines []resolvedLine) (categoryName string, subcategoryName *string) {
	if len(lines) == 0 {
		return "", nil
	}
	catNames := map[string]struct{}{}
	subNames := map[string]struct{}{}
	hasEmptySub := false
	for _, ln := range lines {
		catNames[ln.categoryName] = struct{}{}
		sub := strings.TrimSpace(ln.subcategoryName)
		if sub == "" {
			hasEmptySub = true
		} else {
			subNames[sub] = struct{}{}
		}
	}
	if len(catNames) > 1 {
		names := make([]string, 0, len(catNames))
		for n := range catNames {
			names = append(names, n)
		}
		sort.Strings(names)
		return strings.Join(names, ", "), nil
	}
	var onlyCat string
	for n := range catNames {
		onlyCat = n
	}
	if len(subNames) > 1 || (len(subNames) == 1 && hasEmptySub) {
		return onlyCat, nil
	}
	if len(subNames) == 1 {
		var onlySub string
		for n := range subNames {
			onlySub = n
		}
		return onlyCat, &onlySub
	}
	return onlyCat, nil
}

func collectLineAssignments(ctx context.Context, db *sql.DB, userID, receiptImportID string) ([]lineAssignment, error) {
	q := sqlcdb.New(db)
	rid := receiptImportID
	txRows, err := q.ListReceiptLinkedTransactionCategories(ctx, sqlcdb.ListReceiptLinkedTransactionCategoriesParams{
		UserID:          userID,
		ReceiptImportID: &rid,
	})
	if err != nil {
		return nil, err
	}
	itemRows, err := q.ListReceiptImportItemCategories(ctx, sqlcdb.ListReceiptImportItemCategoriesParams{
		UserID:          userID,
		ReceiptImportID: receiptImportID,
	})
	if err != nil {
		return nil, err
	}

	byIndex := map[int64]lineAssignment{}
	for _, row := range itemRows {
		if row.CategoryID == "" {
			continue
		}
		sub := ""
		if row.SubcategoryID != nil {
			sub = *row.SubcategoryID
		}
		byIndex[row.ItemIndex] = lineAssignment{categoryID: row.CategoryID, subcategoryID: sub}
	}
	for _, row := range txRows {
		if row.CategoryID == nil || *row.CategoryID == "" || row.ReceiptItemIndex == nil {
			continue
		}
		sub := ""
		if row.SubcategoryID != nil {
			sub = *row.SubcategoryID
		}
		byIndex[*row.ReceiptItemIndex] = lineAssignment{categoryID: *row.CategoryID, subcategoryID: sub}
	}
	if len(byIndex) == 0 {
		return nil, nil
	}
	out := make([]lineAssignment, 0, len(byIndex))
	for _, ln := range byIndex {
		out = append(out, ln)
	}
	return out, nil
}
