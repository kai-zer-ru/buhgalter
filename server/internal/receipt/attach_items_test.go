package receipt

import (
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func attachItemsForReceipt(receipt fns.ParsedReceipt, categoryID string) []ImportItemInput {
	items := make([]ImportItemInput, len(receipt.Items))
	for i := range receipt.Items {
		cat := categoryID
		items[i] = ImportItemInput{Index: i, CategoryID: &cat, Include: true}
	}
	return items
}
