package household

import (
	"strings"
	"testing"
)

func TestNormalizeInviteCode(t *testing.T) {
	code, err := generateInviteCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 9 {
		t.Fatalf("len=%d", len(code))
	}
	formatted := formatInviteCode(code)
	got, ok := normalizeInviteCode(formatted)
	if !ok || got != code {
		t.Fatalf("format roundtrip %q → %q ok=%v", formatted, got, ok)
	}
	got, ok = normalizeInviteCode(strings.ToLower(formatted))
	if !ok || got != code {
		t.Fatalf("lower roundtrip")
	}
	got, ok = normalizeInviteCode(" " + code[:3] + " " + code[3:])
	if !ok || got != code {
		t.Fatalf("spaces roundtrip %q", got)
	}
	if _, ok := normalizeInviteCode("ABC"); ok {
		t.Fatal("short accepted")
	}
	if _, ok := normalizeInviteCode("000-000-000"); ok {
		t.Fatal("invalid alphabet accepted")
	}
}
