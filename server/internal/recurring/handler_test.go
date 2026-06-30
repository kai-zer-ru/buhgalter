package recurring

import "testing"

func TestParseInputDefaultTimeLocal(t *testing.T) {
	in, err := parseInput(createRequest{
		Type:       "expense",
		Amount:     "100.00",
		AccountID:  "acc",
		CategoryID: "cat",
		Period:     "month",
		StartDate:  "2026-06-01 00:00:00",
		TimeLocal:  "",
	})
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}
	if in.TimeLocal != "08:00" {
		t.Fatalf("expected default time_local 08:00, got %q", in.TimeLocal)
	}
}

func TestParseInputExplicitTimeLocal(t *testing.T) {
	in, err := parseInput(createRequest{
		Type:       "expense",
		Amount:     "50.00",
		AccountID:  "acc",
		CategoryID: "cat",
		Period:     "month",
		StartDate:  "2026-06-01 00:00:00",
		TimeLocal:  "14:30",
	})
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}
	if in.TimeLocal != "14:30" {
		t.Fatalf("expected time_local 14:30, got %q", in.TimeLocal)
	}
}
