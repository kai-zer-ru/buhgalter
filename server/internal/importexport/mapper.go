package importexport

import (
	"fmt"
	"strings"
	"time"
)

func parseISODate(iso string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return time.Time{}, fmt.Errorf("некорректная дата %q", iso)
	}
	return t, nil
}

// ApplyColumnMap maps a raw row using user-defined column_map.
func ApplyColumnMap(headers []string, row RawRow, m ColumnMap) (MappedRow, error) {
	idx := cubuxFieldIndex(headers)
	get := func(field string) string {
		src, ok := m[field]
		if !ok || src == "" {
			return ""
		}
		return cellAt(row, idx, src)
	}

	out := MappedRow{
		RowNum:         row.RowNum,
		CubuxType:      get(ColType),
		DebitAccount:   get(ColDebitAccount),
		CreditAccount:  get(ColCreditAccount),
		Category:       get(ColCategory),
		Subcategory:    get(ColSubcategory),
		Description:    get(ColDescription),
		Project:        get(ColProject),
		User:           get(ColUser),
		DebitCurrency:  get(ColDebitCurrency),
		CreditCurrency: get(ColCreditCurrency),
	}

	dateStr := get(ColDate)
	if dateStr == "" {
		return out, fmt.Errorf("не указана дата")
	}
	var err error
	out.Date, err = parseImportDate(dateStr)
	if err != nil {
		return out, err
	}

	txType := strings.ToLower(strings.TrimSpace(out.CubuxType))
	switch txType {
	case "расходы", "expense", "расход":
		out.CubuxType = "Расходы"
		amt := get(ColDebitAmount)
		if amt == "" {
			return out, fmt.Errorf("не указана сумма")
		}
		out.DebitAmount, err = ParseCubuxAmount(amt)
	case "доходы", "income", "доход":
		out.CubuxType = "Доходы"
		amt := get(ColCreditAmount)
		if amt == "" {
			return out, fmt.Errorf("не указана сумма")
		}
		out.CreditAmount, err = ParseCubuxAmount(amt)
	case "перевод", "transfer":
		out.CubuxType = "Перевод"
		amt := get(ColDebitAmount)
		if amt == "" {
			amt = get(ColCreditAmount)
		}
		if amt == "" {
			return out, fmt.Errorf("не указана сумма")
		}
		out.DebitAmount, err = ParseCubuxAmount(amt)
	default:
		return out, fmt.Errorf("неизвестный тип: %s", out.CubuxType)
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

// MapTable applies preset or custom mapping to all rows.
func MapTable(table RawTable, opts ImportOptions) ([]MappedRow, []RowError) {
	var mapped []MappedRow
	var errs []RowError
	for _, row := range table.Rows {
		var m MappedRow
		var err error
		switch opts.Preset {
		case "custom":
			m, err = ApplyColumnMap(table.Headers, row, opts.ColumnMap)
		default:
			m, err = MapCubuxRow(table.Headers, row)
		}
		if err != nil {
			errs = append(errs, RowError{Row: row.RowNum, Message: err.Error()})
			continue
		}
		mapped = append(mapped, m)
	}
	return mapped, errs
}

// PreviewFromMapped builds preview items and entity lists without DB writes.
func PreviewFromMapped(rows []MappedRow) (Report, map[string]struct{}, map[string]struct{}) {
	accounts := make(map[string]struct{})
	categories := make(map[string]struct{})
	report := Report{
		TotalRows: len(rows),
		Errors:    make([]RowError, 0),
		Preview:   make([]PreviewItem, 0, min(50, len(rows))),
	}

	for _, m := range rows {
		item, accts, cats, err := mappedToPreview(m)
		if err != nil {
			report.Errors = append(report.Errors, RowError{Row: m.RowNum, Message: err.Error()})
			continue
		}
		report.ValidRows++
		for _, a := range accts {
			accounts[a] = struct{}{}
		}
		for _, c := range cats {
			categories[c] = struct{}{}
		}
		if len(report.Preview) < 50 {
			report.Preview = append(report.Preview, item)
		}
	}
	return report, accounts, categories
}

func mappedToPreview(m MappedRow) (PreviewItem, []string, []string, error) {
	date := m.Date.Format("2006-01-02")
	switch m.CubuxType {
	case "Расходы":
		if m.DebitAccount == "" {
			return PreviewItem{}, nil, nil, fmt.Errorf("не указан счёт списания")
		}
		return PreviewItem{
			Row: m.RowNum, Action: ActionCreateExpense,
			Account: m.DebitAccount, Amount: m.DebitAmount,
			Category: m.Category, Subcategory: m.Subcategory,
			Date: date, Description: m.Description,
		}, []string{m.DebitAccount}, categoryNames(m, "expense"), nil
	case "Доходы":
		if m.CreditAccount == "" {
			return PreviewItem{}, nil, nil, fmt.Errorf("не указан счёт пополнения")
		}
		return PreviewItem{
			Row: m.RowNum, Action: ActionCreateIncome,
			Account: m.CreditAccount, Amount: m.CreditAmount,
			Category: m.Category, Subcategory: m.Subcategory,
			Date: date, Description: m.Description,
		}, []string{m.CreditAccount}, categoryNames(m, "income"), nil
	case "Перевод":
		return PreviewItem{
			Row: m.RowNum, Action: ActionCreateTransfer,
			Account: m.DebitAccount, ToAccount: m.CreditAccount,
			Amount: m.DebitAmount, Date: date, Description: m.Description,
		}, []string{m.DebitAccount, m.CreditAccount}, nil, nil
	default:
		return PreviewItem{}, nil, nil, fmt.Errorf("неизвестный тип")
	}
}

func categoryNames(m MappedRow, txType string) []string {
	if strings.TrimSpace(m.Category) == "" {
		return nil
	}
	return []string{m.Category + " (" + txType + ")"}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
