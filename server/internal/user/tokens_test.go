package user

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAPIToken(t *testing.T) {
	raw, hash, prefix, err := generateAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, apiTokenPrefix) {
		t.Fatalf("expected prefix %q, got %q", apiTokenPrefix, raw)
	}
	if prefix != raw[:8] {
		t.Fatalf("expected prefix %q, got %q", raw[:8], prefix)
	}
	if hash == "" || hash == raw {
		t.Fatal("expected hashed token")
	}
}

func TestResolveTokenExpiryNeverExpires(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	dbVal, resp, perpetual, err := resolveTokenExpiry(true, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if dbVal != nil || resp != nil || !perpetual {
		t.Fatalf("expected perpetual nil expiry, got db=%v resp=%v perpetual=%v", dbVal, resp, perpetual)
	}
}

func TestResolveTokenExpiryDefaultMonth(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	dbVal, resp, perpetual, err := resolveTokenExpiry(false, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if perpetual || resp == nil {
		t.Fatal("expected default expiry")
	}
	expected := now.Add(defaultTokenLifetime).Format(time.RFC3339)
	if dbVal != expected || *resp != expected {
		t.Fatalf("expected %q, got db=%v resp=%v", expected, dbVal, resp)
	}
}

func TestResolveTokenExpiryExplicit(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	exp := "2026-12-31T00:00:00Z"
	dbVal, resp, perpetual, err := resolveTokenExpiry(false, &exp, now)
	if err != nil {
		t.Fatal(err)
	}
	if perpetual || resp == nil || *resp != exp || dbVal != exp {
		t.Fatalf("unexpected result db=%v resp=%v perpetual=%v", dbVal, resp, perpetual)
	}
}

func TestResolveTokenExpiryPastRejected(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	past := "2026-01-01T00:00:00Z"
	_, _, _, err := resolveTokenExpiry(false, &past, now)
	if err == nil {
		t.Fatal("expected error for past expiry")
	}
}

func TestResolveTokenExpiryInvalidFormat(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	bad := "not-a-date"
	_, _, _, err := resolveTokenExpiry(false, &bad, now)
	if err == nil {
		t.Fatal("expected error for invalid expiry")
	}
}
