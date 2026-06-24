package importexport

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
)

func parseImportDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("пустая дата")
	}

	if iso, err := ParseCubuxDate(s); err == nil {
		return parseISODate(iso)
	}

	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		time.RFC3339,
		"02.01.2006 15:04",
		"02.01.2006 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	if t, ok := parseExcelSerialDate(s); ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("некорректная дата %q", s)
}

func parseExcelSerialDate(s string) (time.Time, bool) {
	clean := strings.TrimSpace(s)
	clean = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '\u00a0' || r == '\u202f' {
			return -1
		}
		return r
	}, clean)
	clean = strings.ReplaceAll(clean, ",", ".")
	if clean == "" {
		return time.Time{}, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return time.Time{}, false
	}
	// Serial day numbers in modern spreadsheets are comfortably above 1000.
	if v < 1000 {
		return time.Time{}, false
	}
	if t, err := excelize.ExcelDateToTime(v, false); err == nil {
		return t, true
	}
	if t, err := excelize.ExcelDateToTime(v, true); err == nil {
		return t, true
	}
	return time.Time{}, false
}
