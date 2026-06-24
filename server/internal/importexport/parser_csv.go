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
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".xlsx"):
		return ParseXLSX(data)
	case strings.HasSuffix(lower, ".csv"):
		return ParseCSV(data)
	default:
		if len(data) > 2 && data[0] == 'P' && data[1] == 'K' {
			return ParseXLSX(data)
		}
		return ParseCSV(data)
	}
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
