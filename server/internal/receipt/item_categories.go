package receipt

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func validateAttachItems(ctx context.Context, db *sql.DB, userID string, receipt fns.ParsedReceipt, items []ImportItemInput) error {
	if len(receipt.Items) == 0 {
		return ErrInvalidItem
	}
	if len(items) != len(receipt.Items) {
		return fmt.Errorf("%w: item count mismatch", ErrInvalidItem)
	}
	seen := map[int]struct{}{}
	for _, itemIn := range items {
		if !itemIn.Include {
			return fmt.Errorf("%w: all receipt lines required", ErrInvalidItem)
		}
		if itemIn.Index < 0 || itemIn.Index >= len(receipt.Items) {
			return ErrInvalidItem
		}
		if _, dup := seen[itemIn.Index]; dup {
			return ErrInvalidItem
		}
		seen[itemIn.Index] = struct{}{}
		if itemIn.CategoryID == nil || strings.TrimSpace(*itemIn.CategoryID) == "" {
			return fmt.Errorf("%w: category required", ErrInvalidItem)
		}
		if err := validateExpenseCategory(ctx, db, userID, itemIn.CategoryID); err != nil {
			return err
		}
	}
	for i := range receipt.Items {
		if _, ok := seen[i]; !ok {
			return fmt.Errorf("%w: missing item index %d", ErrInvalidItem, i)
		}
	}
	return nil
}

func saveReceiptItemCategories(
	ctx context.Context,
	_ sqlcdb.DBTX,
	q *sqlcdb.Queries,
	userID, receiptID string,
	receipt fns.ParsedReceipt,
	items []ImportItemInput,
) error {
	for _, itemIn := range items {
		if !itemIn.Include {
			continue
		}
		if itemIn.Index < 0 || itemIn.Index >= len(receipt.Items) {
			return ErrInvalidItem
		}
		if err := q.UpsertReceiptImportItemCategory(ctx, sqlcdb.UpsertReceiptImportItemCategoryParams{
			ReceiptImportID: receiptID,
			ItemIndex:       int64(itemIn.Index),
			UserID:          userID,
			CategoryID:      strings.TrimSpace(*itemIn.CategoryID),
			SubcategoryID:   itemIn.SubcategoryID,
		}); err != nil {
			return err
		}
	}
	return nil
}
