package money

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseRubles converts a decimal ruble string (e.g. "1500.00") to kopecks.
func ParseRubles(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.ReplaceAll(s, " ", "")

	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 2)
	rubles, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}

	var kopecks int64
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 2 {
			return 0, fmt.Errorf("too many decimal places")
		}
		for len(frac) < 2 {
			frac += "0"
		}
		kopecks, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount: %w", err)
		}
	}

	total := rubles*100 + kopecks
	if negative {
		total = -total
	}
	return total, nil
}

// FormatRubles formats kopecks as a decimal ruble string with two fractional digits.
func FormatRubles(kopecks int64) string {
	negative := kopecks < 0
	if negative {
		kopecks = -kopecks
	}
	rubles := kopecks / 100
	frac := kopecks % 100
	s := fmt.Sprintf("%d.%02d", rubles, frac)
	if negative {
		s = "-" + s
	}
	return s
}

// ParseAmount accepts JSON amount as decimal string ("123.45") or integer kopecks (12345).
func ParseAmount(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return ParseRubles(asString)
	}
	var asInt int64
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return asInt, nil
	}
	var asFloat float64
	if err := json.Unmarshal(raw, &asFloat); err == nil {
		return int64(math.Round(asFloat * 100)), nil
	}
	return 0, fmt.Errorf("invalid amount")
}
