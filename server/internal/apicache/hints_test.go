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
