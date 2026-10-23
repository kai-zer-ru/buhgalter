package transaction

import (
	"context"
	"database/sql"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/receiptcategory"
)

func enrichReceiptCategoryDisplay(ctx context.Context, db *sql.DB, userID string, tx *Transaction) {
	if tx == nil || tx.ReceiptImportID == nil || strings.TrimSpace(*tx.ReceiptImportID) == "" {
		return
	}
	cn, ci, sn, si := receiptcategory.DisplayLabels(ctx, db, userID, *tx.ReceiptImportID)
	if cn != nil {
		tx.CategoryName = cn
		tx.CategoryIcon = ci
		tx.SubcategoryName = sn
		tx.SubcategoryIcon = si
	}
}

func enrichReceiptCategoryDisplayBatch(ctx context.Context, db *sql.DB, userID string, items []Transaction) {
	cache := map[string]struct {
		cn, ci, sn, si *string
	}{}
	for i := range items {
		rid := items[i].ReceiptImportID
		if rid == nil || strings.TrimSpace(*rid) == "" {
			continue
		}
		key := *rid
		cached, ok := cache[key]
		if !ok {
			cn, ci, sn, si := receiptcategory.DisplayLabels(ctx, db, userID, key)
			cached = struct {
				cn, ci, sn, si *string
			}{cn, ci, sn, si}
			cache[key] = cached
		}
		if cached.cn != nil {
			items[i].CategoryName = cached.cn
			items[i].CategoryIcon = cached.ci
			items[i].SubcategoryName = cached.sn
			items[i].SubcategoryIcon = cached.si
		}
	}
}
