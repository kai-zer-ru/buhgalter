package receipt

import (
	"context"
	"database/sql"

	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

// DetailItem is a receipt line with optional ledger category labels from linked transactions.
type DetailItem struct {
	Name            string  `json:"name"`
	Quantity        float64 `json:"quantity"`
	Price           int64   `json:"price"`
	Sum             int64   `json:"sum"`
	CategoryName    string  `json:"category_name,omitempty"`
	SubcategoryName string  `json:"subcategory_name,omitempty"`
}

type linkedCategory struct {
	CategoryName    string
	SubcategoryName string
}

func detailItemsFromPayload(items []fns.Item, byIndex map[int64]linkedCategory, attach linkedCategory) []DetailItem {
	out := make([]DetailItem, 0, len(items))
	for i, item := range items {
		di := DetailItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
			Sum:      item.Sum,
		}
		if lc, ok := byIndex[int64(i)]; ok {
			di.CategoryName = lc.CategoryName
			di.SubcategoryName = lc.SubcategoryName
		} else if attach.CategoryName != "" || attach.SubcategoryName != "" {
			di.CategoryName = attach.CategoryName
			di.SubcategoryName = attach.SubcategoryName
		}
		out = append(out, di)
	}
	return out
}

func loadLinkedCategories(ctx context.Context, db *sql.DB, userID, receiptImportID string) (map[int64]linkedCategory, linkedCategory) {
	rid := receiptImportID
	rows, err := queries(db).ListReceiptLinkedTransactions(ctx, sqlcdb.ListReceiptLinkedTransactionsParams{
		UserID:          userID,
		ReceiptImportID: &rid,
	})
	if err != nil {
		return nil, linkedCategory{}
	}
	byIndex := make(map[int64]linkedCategory)
	var attach linkedCategory
	for _, row := range rows {
		lc := linkedCategoryForTx(ctx, db, userID, row.ID)
		if row.ReceiptItemIndex != nil {
			byIndex[*row.ReceiptItemIndex] = lc
		} else if attach.CategoryName == "" && attach.SubcategoryName == "" {
			attach = lc
		}
	}
	itemRows, err := queries(db).ListReceiptImportItemCategories(ctx, sqlcdb.ListReceiptImportItemCategoriesParams{
		UserID:          userID,
		ReceiptImportID: receiptImportID,
	})
	if err == nil {
		for _, row := range itemRows {
			lc := linkedCategoryFromIDs(ctx, db, userID, row.CategoryID, row.SubcategoryID)
			byIndex[row.ItemIndex] = lc
		}
	}
	return byIndex, attach
}

func linkedCategoryFromIDs(ctx context.Context, db *sql.DB, userID, categoryID string, subcategoryID *string) linkedCategory {
	var lc linkedCategory
	if categoryID != "" {
		cat, err := category.GetByID(ctx, db, userID, categoryID)
		if err == nil {
			lc.CategoryName = cat.Name
		}
	}
	if subcategoryID != nil && *subcategoryID != "" {
		sub, err := category.GetSubcategory(ctx, db, userID, *subcategoryID)
		if err == nil {
			lc.SubcategoryName = sub.Name
		}
	}
	return lc
}

func linkedCategoryForTx(ctx context.Context, db *sql.DB, userID, txID string) linkedCategory {
	txRow, err := transaction.GetByID(ctx, db, userID, txID)
	if err != nil {
		return linkedCategory{}
	}
	var lc linkedCategory
	if txRow.CategoryID != nil && *txRow.CategoryID != "" {
		cat, err := category.GetByID(ctx, db, userID, *txRow.CategoryID)
		if err == nil {
			lc.CategoryName = cat.Name
		}
	}
	if txRow.SubcategoryID != nil && *txRow.SubcategoryID != "" {
		sub, err := category.GetSubcategory(ctx, db, userID, *txRow.SubcategoryID)
		if err == nil {
			lc.SubcategoryName = sub.Name
		}
	}
	return lc
}
