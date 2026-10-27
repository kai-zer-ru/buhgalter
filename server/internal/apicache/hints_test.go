package apicache

import "testing"

func TestHintsForMutation_transaction(t *testing.T) {
	h := HintsForMutation("/api/v1/transactions")
	if len(h.Paths) == 0 {
		t.Fatal("expected hint paths")
	}
	if !contains(h.Paths, "/api/v1/transactions") || !contains(h.Paths, "/api/v1/dashboard") {
		t.Fatalf("paths=%v", h.Paths)
	}
	if !contains(h.Entities, "transaction") {
		t.Fatalf("entities=%v", h.Entities)
	}
}

func TestHintsForMutation_accountDetail(t *testing.T) {
	h := HintsForMutation("/api/v1/accounts/acc-1")
	if !contains(h.Paths, "/api/v1/accounts/acc-1") {
		t.Fatalf("missing concrete account path: %v", h.Paths)
	}
	if !contains(h.Paths, "/api/v1/accounts") {
		t.Fatalf("missing accounts list: %v", h.Paths)
	}
}

func TestHintsForMutation_transfers(t *testing.T) {
	h := HintsForMutation("/api/v1/transfers")
	wantPaths := []string{
		"/api/v1/transactions",
		"/api/v1/dashboard",
		"/api/v1/accounts",
		"/api/v1/accounts/summary",
		"/api/v1/stats",
		"/api/v1/budgets",
	}
	for _, p := range wantPaths {
		if !contains(h.Paths, p) {
			t.Fatalf("missing path %s in %v", p, h.Paths)
		}
	}
	if !contains(h.Entities, "transaction") || !contains(h.Entities, "account") {
		t.Fatalf("entities=%v", h.Entities)
	}
}

func TestHintsForMutation_merchants(t *testing.T) {
	h := HintsForMutation("/api/v1/merchants/id/merge")
	for _, p := range []string{
		"/api/v1/merchants",
		"/api/v1/ui/meta",
		"/api/v1/transactions",
		"/api/v1/transaction-templates",
	} {
		if !contains(h.Paths, p) {
			t.Fatalf("missing path %s in %v", p, h.Paths)
		}
	}
	if !contains(h.Entities, "merchant") || !contains(h.Entities, "transaction") {
		t.Fatalf("entities=%v", h.Entities)
	}
}

func TestHintsForMutation_household(t *testing.T) {
	h := HintsForMutation("/api/v1/households/me/invites")
	if !contains(h.Paths, "/api/v1/households/me") {
		t.Fatalf("paths=%v", h.Paths)
	}
	h2 := HintsForMutation("/api/v1/household-invites/accept")
	if !contains(h2.Entities, "household") {
		t.Fatalf("entities=%v", h2.Entities)
	}
}

func TestHintsForMutation_unknownIsCoarse(t *testing.T) {
	h := HintsForMutation("/api/v1/auth/logout")
	if len(h.Paths) != 0 {
		t.Fatalf("expected coarse (empty paths), got %v", h.Paths)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
