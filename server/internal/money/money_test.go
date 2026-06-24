package money

import (
	"encoding/json"
	"testing"
)

func TestParseFormatRubles(t *testing.T) {
	cases := []struct {
		in  string
		out int64
	}{
		{"1500.00", 150000},
		{"0", 0},
		{"0.50", 50},
		{"-10.25", -1025},
		{"1 500,00", 150000},
		{"", 0},
		{"5", 500},
		{"5.5", 550},
	}
	for _, c := range cases {
		got, err := ParseRubles(c.in)
		if err != nil {
			t.Fatalf("ParseRubles(%q): %v", c.in, err)
		}
		if got != c.out {
			t.Fatalf("ParseRubles(%q) = %d, want %d", c.in, got, c.out)
		}
		if FormatRubles(got) != FormatRubles(c.out) {
			t.Fatalf("round-trip %q", c.in)
		}
	}
}

func TestParseRublesErrors(t *testing.T) {
	bad := []string{"abc", "1.2.3", "1.234"}
	for _, in := range bad {
		if _, err := ParseRubles(in); err == nil {
			t.Fatalf("ParseRubles(%q) expected error", in)
		}
	}
	if _, err := ParseRubles("1.999"); err == nil {
		t.Fatal("expected too many decimal places")
	}
}

func TestFormatRublesNegative(t *testing.T) {
	if got := FormatRubles(-105); got != "-1.05" {
		t.Fatalf("got %q", got)
	}
}

func TestParseAmount(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{`"10.50"`, 1050},
		{`1050`, 1050},
		{`10.5`, 1050},
		{`null`, 0},
		{``, 0},
	}
	for _, c := range cases {
		got, err := ParseAmount(json.RawMessage(c.raw))
		if err != nil {
			t.Fatalf("ParseAmount(%s): %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseAmount(%s) = %d, want %d", c.raw, got, c.want)
		}
	}
	if _, err := ParseAmount(json.RawMessage(`"nope"`)); err == nil {
		t.Fatal("expected invalid amount")
	}
	if _, err := ParseAmount(json.RawMessage(`{"x":1}`)); err == nil {
		t.Fatal("expected invalid amount for object")
	}
}
