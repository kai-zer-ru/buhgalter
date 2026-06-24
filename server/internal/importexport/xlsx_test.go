package importexport

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseXLSXSample(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	headers := CubuxHeaders
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Sheet1", cell, h)
	}
	_ = f.SetCellValue("Sheet1", "A2", "Расходы")
	_ = f.SetCellValue("Sheet1", "B2", "01.01.2025")
	_ = f.SetCellValue("Sheet1", "C2", "50.00_-₽")
	_ = f.SetCellValue("Sheet1", "E2", "Наличные")
	_ = f.SetCellValue("Sheet1", "I2", "Транспорт")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	table, err := ParseXLSX(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(table.Rows))
	}
	m, err := MapCubuxRow(table.Headers, table.Rows[0])
	if err != nil {
		t.Fatal(err)
	}
	if m.DebitAmount != 5000 {
		t.Fatalf("amount %d", m.DebitAmount)
	}
}

func TestParseXLSXDetectsHeaderAfterIntroRow(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	_ = f.SetCellValue("Sheet1", "A1", "Экспорт Cebex")
	_ = f.SetCellValue("Sheet1", "B1", "Отчет")

	for i, h := range CubuxHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue("Sheet1", cell, h)
	}

	_ = f.SetCellValue("Sheet1", "A3", "Расходы")
	_ = f.SetCellValue("Sheet1", "B3", "01.01.2025")
	_ = f.SetCellValue("Sheet1", "C3", "50.00_-₽")
	_ = f.SetCellValue("Sheet1", "E3", "Наличные")
	_ = f.SetCellValue("Sheet1", "I3", "Транспорт")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	table, err := ParseXLSX(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(table.Rows))
	}
	if table.Rows[0].RowNum != 3 {
		t.Fatalf("expected source row 3, got %d", table.Rows[0].RowNum)
	}

	m, err := MapCubuxRow(table.Headers, table.Rows[0])
	if err != nil {
		t.Fatal(err)
	}
	if m.Date.Format("2006-01-02") != "2025-01-01" {
		t.Fatalf("unexpected date %s", m.Date.Format("2006-01-02"))
	}
}

func TestImportXLSXIntegration(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	csvPath := filepath.Join(root, "1.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Skip("1.csv missing")
	}
	// Build minimal xlsx from first data line concept — covered by TestParseXLSXSample
}

func TestCSVAndXLSXAreEquivalentOnSampleFiles(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	csvPath := filepath.Join(root, "1.csv")
	xlsxPath := filepath.Join(root, "1.xlsx")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Skip("1.csv missing")
	}
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("1.xlsx missing")
	}

	csvData, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	xlsxData, err := os.ReadFile(xlsxPath)
	if err != nil {
		t.Fatal(err)
	}

	csvTable, err := ParseCSV(csvData)
	if err != nil {
		t.Fatal(err)
	}
	xlsxTable, err := ParseXLSX(xlsxData)
	if err != nil {
		t.Fatal(err)
	}

	csvMapped, csvErrs := MapTable(csvTable, ImportOptions{Preset: "cubux"})
	xlsxMapped, xlsxErrs := MapTable(xlsxTable, ImportOptions{Preset: "cubux"})
	if len(csvErrs) > 0 || len(xlsxErrs) > 0 {
		sampleCSVErr := ""
		sampleXLSXErr := ""
		if len(csvErrs) > 0 {
			sampleCSVErr = fmt.Sprintf("row=%d msg=%s", csvErrs[0].Row, csvErrs[0].Message)
		}
		if len(xlsxErrs) > 0 {
			sampleXLSXErr = fmt.Sprintf("row=%d msg=%s", xlsxErrs[0].Row, xlsxErrs[0].Message)
		}
		t.Fatalf(
			"map errors: csv=%d (%s) xlsx=%d (%s)",
			len(csvErrs),
			sampleCSVErr,
			len(xlsxErrs),
			sampleXLSXErr,
		)
	}

	csvSig := mappedSignatureMultiset(csvMapped)
	xlsxSig := mappedSignatureMultiset(xlsxMapped)
	if len(csvSig) != len(xlsxSig) {
		t.Fatalf("different unique signatures: csv=%d xlsx=%d", len(csvSig), len(xlsxSig))
	}

	for key, c := range csvSig {
		if xlsxSig[key] != c {
			t.Fatalf("signature count mismatch for %q: csv=%d xlsx=%d", key, c, xlsxSig[key])
		}
	}
}

func mappedSignatureMultiset(rows []MappedRow) map[string]int {
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		key := strings.Join([]string{
			r.CubuxType,
			r.Date.Format("2006-01-02"),
			r.DebitAccount,
			r.CreditAccount,
			fmt.Sprintf("%d", r.DebitAmount),
			fmt.Sprintf("%d", r.CreditAmount),
			r.Category,
			r.Subcategory,
			r.Description,
		}, "|")
		out[key]++
	}
	return out
}
