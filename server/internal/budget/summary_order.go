package budget

import (
	"sort"

	"github.com/kai-zer-ru/buhgalter/internal/money"
)

func enrichAndSortSummary(items []SummaryItem) []SummaryItem {
	var childrenPlanned, childrenSpent int64
	for i := range items {
		if items[i].Scope == ScopeCategory || items[i].Scope == ScopeSubcategory {
			childrenPlanned += items[i].Planned
			childrenSpent += items[i].Spent
		}
	}
	for i := range items {
		if items[i].Scope != ScopeAllExpense {
			continue
		}
		items[i].ChildrenPlanned = childrenPlanned
		items[i].ChildrenPlannedDisplay = money.FormatRubles(childrenPlanned)
		items[i].ChildrenSpent = childrenSpent
		items[i].ChildrenSpentDisplay = money.FormatRubles(childrenSpent)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Scope == ScopeAllExpense {
			return true
		}
		if items[j].Scope == ScopeAllExpense {
			return false
		}
		return items[i].Name < items[j].Name
	})
	return items
}
