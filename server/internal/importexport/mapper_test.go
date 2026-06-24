package importexport

import (
	"testing"
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
