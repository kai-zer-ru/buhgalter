package apicache

import "strings"

// InvalidateHints tells live clients which GET paths / entities to soft-reload.
// Empty Paths means coarse invalidate (full user cache bust on the client).
type InvalidateHints struct {
	Paths    []string
	Entities []string
}

// HintsForMutation maps a successful write path to realtime invalidate hints.
func HintsForMutation(apiPath string) InvalidateHints {
	path := strings.Split(apiPath, "?")[0]
	switch {
	case strings.HasPrefix(path, "/api/v1/transactions"):
		return InvalidateHints{
			Paths: []string{
				"/api/v1/transactions",
				"/api/v1/dashboard",
				"/api/v1/accounts",
				"/api/v1/accounts/summary",
				"/api/v1/stats",
				"/api/v1/budgets",
			},
			Entities: []string{"transaction", "account", "budget"},
		}
	case strings.HasPrefix(path, "/api/v1/accounts"):
		hints := InvalidateHints{
			Paths: []string{
				"/api/v1/accounts",
				"/api/v1/dashboard",
				"/api/v1/accounts/summary",
			},
			Entities: []string{"account"},
		}
		// Keep the concrete account URL so an open account page refetches.
		if path != "/api/v1/accounts" && path != "/api/v1/accounts/summary" {
			hints.Paths = append(hints.Paths, path)
		}
		return hints
	case strings.HasPrefix(path, "/api/v1/budgets"):
		return InvalidateHints{
			Paths:    []string{"/api/v1/budgets", "/api/v1/dashboard"},
			Entities: []string{"budget"},
		}
	case strings.HasPrefix(path, "/api/v1/credits"):
		return InvalidateHints{
			Paths: []string{
				"/api/v1/credits",
				"/api/v1/dashboard",
				"/api/v1/accounts",
				"/api/v1/accounts/summary",
				"/api/v1/transactions",
			},
			Entities: []string{"credit", "account", "transaction"},
		}
	case strings.HasPrefix(path, "/api/v1/debtors") || strings.HasPrefix(path, "/api/v1/debts"):
		return InvalidateHints{
			Paths: []string{
				"/api/v1/debtors",
				"/api/v1/dashboard",
				"/api/v1/accounts",
				"/api/v1/accounts/summary",
				"/api/v1/transactions",
			},
			Entities: []string{"debt", "account", "transaction"},
		}
	case strings.HasPrefix(path, "/api/v1/subscriptions"):
		return InvalidateHints{
			Paths: []string{
				"/api/v1/subscriptions",
				"/api/v1/dashboard",
				"/api/v1/accounts",
				"/api/v1/accounts/summary",
			},
			Entities: []string{"subscription", "account"},
		}
	case strings.HasPrefix(path, "/api/v1/recurring-operations"):
		return InvalidateHints{
			Paths: []string{
				"/api/v1/recurring-operations",
				"/api/v1/dashboard",
				"/api/v1/accounts",
				"/api/v1/transactions",
			},
			Entities: []string{"recurring", "account", "transaction"},
		}
	case strings.HasPrefix(path, "/api/v1/categories") || strings.HasPrefix(path, "/api/v1/subcategories"):
		return InvalidateHints{
			Paths:    []string{"/api/v1/categories", "/api/v1/ui/meta"},
			Entities: []string{"category"},
		}
	case strings.HasPrefix(path, "/api/v1/merchants"):
		return InvalidateHints{
			Paths:    []string{"/api/v1/merchants"},
			Entities: []string{"merchant"},
		}
	case strings.HasPrefix(path, "/api/v1/tags"):
		return InvalidateHints{
			Paths:    []string{"/api/v1/tags"},
			Entities: []string{"tag"},
		}
	case strings.HasPrefix(path, "/api/v1/transaction-templates"):
		return InvalidateHints{
			Paths:    []string{"/api/v1/transaction-templates", "/api/v1/dashboard"},
			Entities: []string{"transaction_template"},
		}
	case strings.HasPrefix(path, "/api/v1/import"):
		return LedgerHints()
	case strings.HasPrefix(path, "/api/v1/user/data"):
		return LedgerHints()
	default:
		// Unknown write — coarse invalidate (empty Paths).
		return InvalidateHints{}
	}
}

// LedgerHints covers bulk ledger changes (import, reset, scheduler).
func LedgerHints() InvalidateHints {
	return InvalidateHints{
		Paths: []string{
			"/api/v1/dashboard",
			"/api/v1/accounts",
			"/api/v1/accounts/summary",
			"/api/v1/transactions",
			"/api/v1/budgets",
			"/api/v1/credits",
			"/api/v1/debtors",
			"/api/v1/subscriptions",
			"/api/v1/recurring-operations",
			"/api/v1/stats",
			"/api/v1/transaction-templates",
			"/api/v1/ui/meta",
		},
		Entities: []string{
			"transaction", "account", "budget", "credit", "debt",
			"subscription", "recurring", "category",
		},
	}
}
