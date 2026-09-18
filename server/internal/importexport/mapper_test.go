package importexport

import (
	"testing"
	"time"
)

func TestApplyColumnMapCustom(t *testing.T) {
	headers := []string{"Type", "Date", "Amount", "Account", "Category"}
	row := RawRow{
		RowNum: 2,
		Values: []string{"expense", "2025-01-15", "100.50", "Cash", "Food"},
	}
	m := ColumnMap{
		ColType:         "Type",
		ColDate:         "Date",
		ColDebitAmount:  "Amount",
		ColDebitAccount: "Account",
		ColCategory:     "Category",
	}
	mapped, err := ApplyColumnMap(headers, row, m)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.DebitAccount != "Cash" || mapped.Category != "Food" {
		t.Fatalf("mapped: %+v", mapped)
	}
	if mapped.DebitAmount != 10050 {
		t.Fatalf("amount %d", mapped.DebitAmount)
	}
}

func TestMapTableCustomPreset(t *testing.T) {
	table := RawTable{
		Headers: []string{"Type", "Date", "Amount", "Account", "Category"},
		Rows: []RawRow{{
			RowNum: 2,
			Values: []string{"income", "2025-02-01", "200", "Bank", "Salary"},
		}},
	}
	mapped, errs := MapTable(table, ImportOptions{
		Preset: "custom",
		ColumnMap: ColumnMap{
			ColType:          "Type",
			ColDate:          "Date",
			ColCreditAmount:  "Amount",
			ColCreditAccount: "Account",
			ColCategory:      "Category",
		},
	})
	if len(errs) > 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(mapped) != 1 || mapped[0].CreditAmount != 20000 {
		t.Fatalf("mapped: %+v", mapped)
	}
}

func TestCollectFileAccountsAndCategories(t *testing.T) {
	rows := []MappedRow{
		{DebitAccount: "Cash", Category: "Food", CubuxType: "Расходы"},
		{CreditAccount: "Bank", Category: "Salary", CubuxType: "Доходы"},
	}
	accts := collectFileAccounts(rows)
	if len(accts) < 2 {
		t.Fatalf("accounts: %v", accts)
	}
	cats := collectFileCategories(rows)
	if len(cats) < 2 {
		t.Fatalf("categories: %v", cats)
	}
}

func TestMapBuhgalterRowExtraColumns(t *testing.T) {
	headers := BuhgalterHeaders
	row := RawRow{
		RowNum: 2,
		Values: []string{
			"Расходы", "18.09.2026", "150.00", "RUB", "Наличные",
			"", "", "", "Транспорт", "Автобус", "поездка", "", "User",
			"09:15:00", "Пятёрочка", "дом, еда",
		},
	}
	m, err := MapBuhgalterRow(headers, row)
	if err != nil {
		t.Fatal(err)
	}
	if !m.HasTime || m.Date.Hour() != 9 || m.Date.Minute() != 15 {
		t.Fatalf("time: %v hasTime=%v", m.Date, m.HasTime)
	}
	if m.Merchant != "Пятёрочка" {
		t.Fatalf("merchant %q", m.Merchant)
	}
	if len(m.Tags) != 2 || m.Tags[0] != "дом" || m.Tags[1] != "еда" {
		t.Fatalf("tags %#v", m.Tags)
	}
}

func TestMapTableBuhgalterPreset(t *testing.T) {
	table := RawTable{
		Headers: BuhgalterHeaders,
		Rows: []RawRow{{
			RowNum: 2,
			Values: []string{
				"Доходы", "01.02.2025", "", "", "",
				"200.00", "RUB", "Яндекс", "Зарплата", "", "", "", "User",
				"08:00:00", "", "работа",
			},
		}},
	}
	mapped, errs := MapTable(table, ImportOptions{Preset: "buhgalter"})
	if len(errs) > 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(mapped) != 1 || !mapped[0].HasTime || mapped[0].CreditAmount != 20000 {
		t.Fatalf("mapped: %+v", mapped)
	}
	if len(mapped[0].Tags) != 1 || mapped[0].Tags[0] != "работа" {
		t.Fatalf("tags %#v", mapped[0].Tags)
	}
}

func TestPreviewFromMappedCountsJournalRows(t *testing.T) {
	report, _, _ := PreviewFromMapped([]MappedRow{
		{RowNum: 1, CubuxType: "Расходы", DebitAccount: "Cash", DebitAmount: 100, Date: parseISODateForTest(t, "2025-01-01")},
		{RowNum: 2, CubuxType: "Доходы", CreditAccount: "Bank", CreditAmount: 200, Date: parseISODateForTest(t, "2025-01-02")},
		{RowNum: 3, CubuxType: "Перевод", DebitAccount: "Cash", CreditAccount: "Bank", DebitAmount: 300, Date: parseISODateForTest(t, "2025-01-03")},
	})
	if report.ValidRows != 3 {
		t.Fatalf("valid %d", report.ValidRows)
	}
	if report.TransferRows != 1 {
		t.Fatalf("transfers %d", report.TransferRows)
	}
	if report.ListRows != 4 {
		t.Fatalf("list %d", report.ListRows)
	}
}

func parseISODateForTest(t *testing.T, iso string) time.Time {
	t.Helper()
	d, err := parseISODate(iso)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
