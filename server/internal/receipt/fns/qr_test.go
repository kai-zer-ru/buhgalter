package fns

import (
	"strings"
	"testing"
	"time"
)

func TestParseQR_valid(t *testing.T) {
	t.Parallel()
	raw := "t=20260115T1030&s=150.00&fn=7288940700123456&i=12345&fp=9876543210&n=1"
	p, err := ParseQR(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.FiscalDriveNumber != "7288940700123456" || p.FiscalDocumentNumber != "12345" || p.FiscalSign != "9876543210" {
		t.Fatalf("unexpected fiscal fields: %+v", p)
	}
	if p.TotalSumKopecks != 15000 {
		t.Fatalf("expected 15000 kopecks, got %d", p.TotalSumKopecks)
	}
	if p.OperationType != "1" {
		t.Fatalf("expected operation type 1, got %q", p.OperationType)
	}
	want := time.Date(2026, 1, 15, 10, 30, 0, 0, moscowLocation())
	if !p.DateTime.Equal(want) {
		t.Fatalf("expected %v, got %v", want, p.DateTime)
	}
}

func TestParseQR_withURLPrefix(t *testing.T) {
	t.Parallel()
	raw := "https://example.test/?t=20260115T103000&s=10,50&fn=1&i=2&fp=3"
	p, err := ParseQR(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.TotalSumKopecks != 1050 {
		t.Fatalf("expected 1050 kopecks, got %d", p.TotalSumKopecks)
	}
}

func TestParseQR_invalid(t *testing.T) {
	t.Parallel()
	cases := []string{"", "not-a-qr", "t=bad&s=1&fn=1&i=1&fp=1", "t=20260115T1030&s=x&fn=1&i=1&fp=1"}
	for _, raw := range cases {
		if _, err := ParseQR(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestCheckURL_containsFiscalParams(t *testing.T) {
	t.Parallel()
	p := QRParams{
		DateTime:             time.Date(2026, 1, 15, 10, 30, 0, 0, moscowLocation()),
		TotalSumKopecks:      15000,
		FiscalDriveNumber:    "fn",
		FiscalDocumentNumber: "i",
		FiscalSign:           "fp",
		OperationType:        "1",
	}
	u := CheckURL(p)
	if !strings.Contains(u, "proverkacheka.nalog.ru") {
		t.Fatalf("unexpected url: %s", u)
	}
	for _, part := range []string{"fn=fn", "i=i", "fp=fp", "s=150.00"} {
		if !strings.Contains(u, part) {
			t.Fatalf("url missing %q: %s", part, u)
		}
	}
}
