package importexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatCubuxAmount(t *testing.T) {
	tests := []struct {
		kopecks int64
		want    string
	}{
		{5000, "50.00"},
		{23797, "237.97"},
		{3102446, "31024.46"},
	}
	for _, tc := range tests {
		got := FormatCubuxAmount(tc.kopecks)
		if got != tc.want {
			t.Fatalf("%d kopecks: got %q want %q", tc.kopecks, got, tc.want)
		}
		if strings.Contains(got, "_") || strings.Contains(got, "₽") {
			t.Fatalf("%d kopecks: unexpected symbols in %q", tc.kopecks, got)
		}
	}
}

func TestParseCubuxAmount(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"50.00_-₽", 5000},
		{"31024.46_-₽", 3102446},
		{"1000.00", 100000},
		{"31 024,46 ₽", 3102446},
		{"31\u00a0024,46 ₽", 3102446},
		{"31\u202f024,46 ₽", 3102446},
		{"31.024,46 ₽", 3102446},
		{"31,024.46 ₽", 3102446},
	}
	for _, tc := range tests {
		got, err := ParseCubuxAmount(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: want %d got %d", tc.in, tc.want, got)
		}
	}
}

func TestParseCubuxDate(t *testing.T) {
	got, err := ParseCubuxDate("01.01.2025")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2025-01-01" {
		t.Fatalf("want 2025-01-01 got %s", got)
	}
}

func TestStripUTF8BOM(t *testing.T) {
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Тип,Дата")...)
	out := StripUTF8BOM(data)
	if string(out) != "Тип,Дата" {
		t.Fatalf("bom not stripped: %q", out)
	}
}

func TestDedupHash(t *testing.T) {
	h1 := DedupHash("2025-01-01", 5000, "Наличные", "Транспорт", "expense")
	h2 := DedupHash("2025-01-01", 5000, "наличные", "транспорт", "expense")
	if h1 != h2 {
		t.Fatal("dedup hash should be case-insensitive for names")
	}
}

func TestMapCubuxExpenseRow(t *testing.T) {
	headers := CubuxHeaders
	row := RawRow{
		RowNum: 2,
		Values: []string{
			"Расходы", "01.01.2025", "50.00_-₽", "RUB", "Наличные",
			"", "", "", "Транспорт", "Автобус", "", "", "User",
		},
	}
	m, err := MapCubuxRow(headers, row)
	if err != nil {
		t.Fatal(err)
	}
	if m.CubuxType != "Расходы" || m.DebitAmount != 5000 || m.DebitAccount != "Наличные" {
		t.Fatalf("unexpected mapped row: %+v", m)
	}
}

func TestMapCubuxTransferRow(t *testing.T) {
	headers := CubuxHeaders
	row := RawRow{
		RowNum: 21,
		Values: []string{
			"Перевод", "06.01.2025", "3680.00_-₽", "RUB", "Яндекс",
			"3680.00_-₽", "RUB", "Кредитка", "Перевод", "", "", "", "User",
		},
	}
	m, err := MapCubuxRow(headers, row)
	if err != nil {
		t.Fatal(err)
	}
	if m.CubuxType != "Перевод" || m.DebitAmount != 368000 || m.DebitAccount != "Яндекс" || m.CreditAccount != "Кредитка" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestMapCubuxIncomeRow(t *testing.T) {
	headers := CubuxHeaders
	row := RawRow{
		RowNum: 50,
		Values: []string{
			"Доходы", "12.01.2025", "", "", "",
			"1860.00_-₽", "RUB", "Яндекс", "Прочие доходы", "Авито", "", "", "User",
		},
	}
	m, err := MapCubuxRow(headers, row)
	if err != nil {
		t.Fatal(err)
	}
	if m.CreditAmount != 186000 || m.CreditAccount != "Яндекс" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestPreview1CSV(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "1.csv"))
	if err != nil {
		t.Skip("1.csv not in repo root")
	}
	table, err := ParseCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	mapped, errs := MapTable(table, ImportOptions{Preset: "cubux"})
	if len(mapped) < 1800 {
		t.Fatalf("expected ~1858 rows, got %d mapped, %d errors", len(mapped), len(errs))
	}
	report, _, _ := PreviewFromMapped(mapped)
	if report.ValidRows < 1800 {
		t.Fatalf("valid rows %d", report.ValidRows)
	}
}
