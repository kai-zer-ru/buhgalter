package importexport

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseImportDateSupportsCommonFormats(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"01.01.2025", "2025-01-01"},
		{"2025-01-01", "2025-01-01"},
		{"2025-01-01 15:00:00", "2025-01-01"},
		{"2025-01-01T15:00:00Z", "2025-01-01"},
	}
	for _, tc := range tests {
		got, err := parseImportDate(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got.Format("2006-01-02") != tc.want {
			t.Fatalf("%q: want %s got %s", tc.in, tc.want, got.Format("2006-01-02"))
		}
	}
}

func TestParseImportDateSupportsExcelSerial(t *testing.T) {
	serial := "45929"
	expected, err := excelize.ExcelDateToTime(45929, false)
	if err != nil {
		t.Fatalf("excelize expected conversion: %v", err)
	}
	got, err := parseImportDate(serial)
	if err != nil {
		t.Fatalf("parseImportDate(%q): %v", serial, err)
	}
	if got.Format("2006-01-02") != expected.Format("2006-01-02") {
		t.Fatalf("want %s got %s", expected.Format("2006-01-02"), got.Format("2006-01-02"))
	}
}
