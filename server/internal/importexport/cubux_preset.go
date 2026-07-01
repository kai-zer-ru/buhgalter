package importexport

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kai-zer-ru/buhgalter/internal/money"
)

// StripUTF8BOM removes a leading UTF-8 BOM if present.
func StripUTF8BOM(data []byte) []byte {
	const bom = "\xef\xbb\xbf"
	if len(data) >= 3 && string(data[:3]) == bom {
		return data[3:]
	}
	return data
}

// ParseCubuxAmount parses Cubux amount strings like "50.00_-₽" or "31024.46_-₽".
func ParseCubuxAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("пустая сумма")
	}
	num := extractAmountNumber(s)
	if num == "" {
		return 0, fmt.Errorf("не удалось извлечь сумму из %q", s)
	}
	kopecks, err := money.ParseRubles(num)
	if err != nil {
		return 0, fmt.Errorf("не удалось распарсить сумму: %w", err)
	}
	if kopecks < 0 {
		kopecks = -kopecks
	}
	return kopecks, nil
}

func extractAmountNumber(s string) string {
	if idx := strings.Index(s, "_"); idx >= 0 {
		return normalizeAmountNumber(strings.TrimSpace(s[:idx]))
	}
	var b strings.Builder
	for i, r := range s {
		if r == '-' && i == 0 {
			b.WriteRune(r)
			continue
		}
		// XLSX exports often include thousand separators as spaces/NBSP.
		if (r == ' ' || r == '\u00a0' || r == '\u202f' || unicode.IsSpace(r)) && b.Len() > 0 {
			continue
		}
		if unicode.IsDigit(r) || r == '.' || r == ',' {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			break
		}
	}
	return normalizeAmountNumber(b.String())
}

func normalizeAmountNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	sign := ""
	if strings.HasPrefix(raw, "-") {
		sign = "-"
		raw = raw[1:]
	}

	// keep only digits and decimal/group separators
	clean := make([]rune, 0, len(raw))
	for _, r := range raw {
		if unicode.IsDigit(r) || r == '.' || r == ',' {
			clean = append(clean, r)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	normalized := string(clean)

	lastDot := strings.LastIndex(normalized, ".")
	lastComma := strings.LastIndex(normalized, ",")
	decPos := lastDot
	if lastComma > decPos {
		decPos = lastComma
	}
	if decPos < 0 {
		return sign + normalized
	}

	intPart := strings.NewReplacer(".", "", ",", "").Replace(normalized[:decPos])
	fracPart := strings.NewReplacer(".", "", ",", "").Replace(normalized[decPos+1:])
	// If there are more than 2 digits after "decimal", this separator is likely
	// a thousands separator, not a decimal one.
	if len(fracPart) > 2 {
		return sign + strings.NewReplacer(".", "", ",", "").Replace(normalized)
	}
	if fracPart == "" {
		return sign + intPart
	}
	return sign + intPart + "." + fracPart
}

// FormatCubuxAmount formats kopecks as a decimal amount for CSV/XLSX export.
func FormatCubuxAmount(kopecks int64) string {
	return money.FormatRubles(kopecks)
}

// ParseCubuxDate parses DD.MM.YYYY dates.
func ParseCubuxDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("пустая дата")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("ожидается DD.MM.YYYY, получено %q", s)
	}
	day, month, year := parts[0], parts[1], parts[2]
	if len(day) != 2 || len(month) != 2 || len(year) != 4 {
		return "", fmt.Errorf("некорректная дата %q", s)
	}
	for _, p := range parts {
		for _, r := range p {
			if !unicode.IsDigit(r) {
				return "", fmt.Errorf("некорректная дата %q", s)
			}
		}
	}
	return fmt.Sprintf("%s-%s-%s", year, month, day), nil
}

// NormalizeHeader trims BOM/spaces from CSV header cells.
func NormalizeHeader(h string) string {
	h = strings.TrimSpace(h)
	if h != "" {
		r, size := utf8.DecodeRuneInString(h)
		if r == '\ufeff' {
			h = h[size:]
		}
	}
	return strings.TrimSpace(h)
}
