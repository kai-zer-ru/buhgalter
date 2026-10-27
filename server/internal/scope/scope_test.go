package scope

import (
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

func TestOwnerIDs(t *testing.T) {
	solo := auth.Scope{UserID: "u1"}
	got := OwnerIDs(solo)
	if len(got) != 1 || got[0] != "u1" {
		t.Fatalf("solo: %v", got)
	}
	hh := auth.Scope{UserID: "u1", HouseholdID: "h", PrincipalID: "p"}
	got = OwnerIDs(hh)
	if len(got) != 2 || got[0] != "u1" || got[1] != "p" {
		t.Fatalf("household: %v", got)
	}
}

func TestFilterOwnerIDs(t *testing.T) {
	solo := auth.Scope{UserID: "u1"}
	if got := FilterOwnerIDs(solo, Household); len(got) != 0 {
		t.Fatalf("solo household view: %v", got)
	}
	hh := auth.Scope{UserID: "u1", PrincipalID: "p"}
	if got := FilterOwnerIDs(hh, Personal); len(got) != 1 || got[0] != "u1" {
		t.Fatalf("personal: %v", got)
	}
	if got := FilterOwnerIDs(hh, Household); len(got) != 1 || got[0] != "p" {
		t.Fatalf("household: %v", got)
	}
	if got := FilterOwnerIDs(hh, All); len(got) != 2 {
		t.Fatalf("all: %v", got)
	}
}

func TestPrincipalForCreate(t *testing.T) {
	solo := auth.Scope{UserID: "u1"}
	got, err := PrincipalForCreate(solo, "")
	if err != nil || got != "u1" {
		t.Fatalf("default personal: %q %v", got, err)
	}
	if _, err := PrincipalForCreate(solo, Household); err != ErrNoHousehold {
		t.Fatalf("solo household: %v", err)
	}
	hh := auth.Scope{UserID: "u1", PrincipalID: "p"}
	got, err = PrincipalForCreate(hh, Household)
	if err != nil || got != "p" {
		t.Fatalf("household: %q %v", got, err)
	}
}

func TestAccountScope(t *testing.T) {
	viewer := auth.Scope{UserID: "u1", PrincipalID: "p"}
	if AccountScope(viewer, "u1") != Personal {
		t.Fatal("personal")
	}
	if AccountScope(viewer, "p") != Household {
		t.Fatal("household")
	}
}
