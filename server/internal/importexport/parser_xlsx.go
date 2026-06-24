package importexport

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParseXLSX reads the first worksheet into a RawTable.
func ParseXLSX(data []byte) (RawTable, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return RawTable{}, fmt.Errorf("xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return RawTable{}, fmt.Errorf("xlsx: нет листов")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return RawTable{}, fmt.Errorf("xlsx: %w", err)
	}
	if len(rows) == 0 {
		return RawTable{}, fmt.Errorf("файл пуст")
	}

	headerRow := detectHeaderRow(rows)
	headers := make([]string, len(rows[headerRow]))
	for i, h := range rows[headerRow] {
		headers[i] = NormalizeHeader(h)
	}

	out := make([]RawRow, 0, len(rows)-headerRow-1)
	for i := headerRow + 1; i < len(rows); i++ {
		line := rows[i]
		if isEmptyRow(line) {
			continue
		}
		out = append(out, RawRow{RowNum: i + 1, Values: line})
	}
	return RawTable{Headers: headers, Rows: out}, nil
}

func detectHeaderRow(rows [][]string) int {
	if len(rows) == 0 {
		return 0
	}

	expected := make(map[string]struct{}, len(CubuxHeaders))
	for _, h := range CubuxHeaders {
		expected[strings.ToLower(NormalizeHeader(h))] = struct{}{}
	}

	bestIdx := 0
	bestScore := -1
	requiredBest := false

	limit := len(rows)
	if limit > 30 {
		limit = 30
	}
	for i := 0; i < limit; i++ {
		row := rows[i]
		if isEmptyRow(row) {
			continue
		}

		score := 0
		hasType := false
		hasDate := false
		seen := make(map[string]struct{}, len(row))
		for _, c := range row {
			key := strings.ToLower(NormalizeHeader(c))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if _, ok := expected[key]; ok {
				score++
			}
			if key == strings.ToLower("Тип") {
				hasType = true
			}
			if key == strings.ToLower("Дата") {
				hasDate = true
			}
		}

		required := hasType && hasDate
		if score > bestScore || (score == bestScore && required && !requiredBest) {
			bestScore = score
			bestIdx = i
			requiredBest = required
		}
	}

	return bestIdx
}

// CubuxHeaders is the fixed Cubux CSV header row.
var CubuxHeaders = []string{
	"Тип", "Дата", "Сумма списания", "Валюта списания", "Счет списания",
	"Сумма пополнения", "Валюта назначения", "Счет пополнения",
	"Категория", "Subcategory", "Описание", "Проект", "Пользователь",
}

func cubuxFieldIndex(headers []string) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[strings.ToLower(NormalizeHeader(h))] = i
	}
	return idx
}

func cellAt(row RawRow, idx map[string]int, names ...string) string {
	for _, name := range names {
		if i, ok := idx[strings.ToLower(name)]; ok && i < len(row.Values) {
			return strings.TrimSpace(row.Values[i])
		}
	}
	return ""
}

// MapCubuxRow maps a raw row using Cubux column headers.
func MapCubuxRow(headers []string, row RawRow) (MappedRow, error) {
	idx := cubuxFieldIndex(headers)
	m := MappedRow{
		RowNum:         row.RowNum,
		CubuxType:      cellAt(row, idx, "Тип"),
		DebitAccount:   cellAt(row, idx, "Счет списания"),
		CreditAccount:  cellAt(row, idx, "Счет пополнения"),
		Category:       cellAt(row, idx, "Категория"),
		Subcategory:    cellAt(row, idx, "Subcategory"),
		Description:    cellAt(row, idx, "Описание"),
		Project:        cellAt(row, idx, "Проект"),
		User:           cellAt(row, idx, "Пользователь"),
		DebitCurrency:  cellAt(row, idx, "Валюта списания"),
		CreditCurrency: cellAt(row, idx, "Валюта назначения"),
	}

	dateStr := cellAt(row, idx, "Дата")
	if dateStr == "" {
		return m, fmt.Errorf("не указана дата")
	}
	parsed, err := parseImportDate(dateStr)
	if err != nil {
		return m, err
	}
	m.Date = parsed

	debitAmt := cellAt(row, idx, "Сумма списания")
	creditAmt := cellAt(row, idx, "Сумма пополнения")

	switch strings.TrimSpace(m.CubuxType) {
	case "Расходы":
		if debitAmt == "" {
			return m, fmt.Errorf("не указана сумма списания")
		}
		m.DebitAmount, err = ParseCubuxAmount(debitAmt)
		if err != nil {
			return m, err
		}
	case "Доходы":
		if creditAmt == "" {
			return m, fmt.Errorf("не указана сумма пополнения")
		}
		m.CreditAmount, err = ParseCubuxAmount(creditAmt)
		if err != nil {
			return m, err
		}
	case "Перевод":
		if debitAmt == "" {
			return m, fmt.Errorf("не указана сумма перевода")
		}
		m.DebitAmount, err = ParseCubuxAmount(debitAmt)
		if err != nil {
			return m, err
		}
		if m.DebitAccount == "" || m.CreditAccount == "" {
			return m, fmt.Errorf("для перевода нужны оба счёта")
		}
	default:
		return m, fmt.Errorf("неизвестный тип: %s", m.CubuxType)
	}
	return m, nil
}
