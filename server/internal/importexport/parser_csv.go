package importexport

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ParseCSV reads CSV data into a RawTable. First row is headers.
func ParseCSV(data []byte) (RawTable, error) {
	data = StripUTF8BOM(data)
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = ','
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	all, err := r.ReadAll()
	if err != nil {
		return RawTable{}, fmt.Errorf("csv: %w", err)
	}
	if len(all) == 0 {
		return RawTable{}, fmt.Errorf("файл пуст")
	}

	headers := make([]string, len(all[0]))
	for i, h := range all[0] {
		headers[i] = NormalizeHeader(h)
	}

	rows := make([]RawRow, 0, len(all)-1)
	for i := 1; i < len(all); i++ {
		line := all[i]
		if isEmptyRow(line) {
			continue
		}
		rows = append(rows, RawRow{RowNum: i + 1, Values: line})
	}
	return RawTable{Headers: headers, Rows: rows}, nil
}

func isEmptyRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// ParseFile detects format by extension and parses CSV or XLSX.
func ParseFile(filename string, data []byte) (RawTable, error) {
	table, _, err := ParseImportFile(filename, data)
	return table, err
}

func ParseImportFile(filename string, data []byte) (RawTable, *NativeFile, error) {
	data = decodeImportText(data)
	if IsNativeCSV(data) {
		return nativeImportTables(data)
	}
	lower := strings.ToLower(filename)
	var table RawTable
	var err error
	switch {
	case strings.HasSuffix(lower, ".xlsx"):
		table, err = ParseXLSX(data)
	case strings.HasSuffix(lower, ".csv"):
		table, err = ParseCSV(data)
	default:
		if len(data) > 2 && data[0] == 'P' && data[1] == 'K' {
			table, err = ParseXLSX(data)
		} else {
			table, err = ParseCSV(data)
		}
	}
	if err != nil {
		return RawTable{}, nil, err
	}
	if looksLikeNativeTable(table) {
		nf, nerr := nativeFileFromFlatTable(table)
		if nerr != nil {
			return RawTable{}, nil, nerr
		}
		return nativeTxTable(nf)
	}
	return table, nil, nil
}

func nativeImportTables(data []byte) (RawTable, *NativeFile, error) {
	nf, err := ParseNativeCSV(data)
	if err != nil {
		return RawTable{}, nil, err
	}
	return nativeTxTable(nf)
}

func nativeTxTable(nf NativeFile) (RawTable, *NativeFile, error) {
	tx, ok := nf.Sections[sectionTx]
	if !ok || len(tx.Headers) == 0 {
		return RawTable{}, &nf, fmt.Errorf("в файле нет секции transactions")
	}
	return tx, &nf, nil
}

func looksLikeNativeTable(table RawTable) bool {
	if len(table.Headers) > 0 && strings.EqualFold(strings.Trim(table.Headers[0], "\"'"), nativeMagic) {
		return true
	}
	for _, row := range table.Rows {
		if len(row.Values) == 0 {
			continue
		}
		cell0 := strings.TrimSpace(row.Values[0])
		if strings.EqualFold(strings.Trim(cell0, "\"'"), nativeMagic) || strings.EqualFold(cell0, "#SECTION") {
			return true
		}
	}
	return false
}

func nativeFileFromFlatTable(table RawTable) (NativeFile, error) {
	all := make([][]string, 0, 1+len(table.Rows))
	if len(table.Headers) > 0 {
		all = append(all, table.Headers)
	}
	for _, row := range table.Rows {
		all = append(all, row.Values)
	}
	return parseNativeRecords(all)
}

// ReadAll reads the upload body with a size cap.
func ReadAll(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(r, maxBytes+1)); err != nil {
		return nil, err
	}
	if int64(buf.Len()) > maxBytes {
		return nil, fmt.Errorf("файл слишком большой")
	}
	return buf.Bytes(), nil
}
