package money

import (
	"errors"
	"math/big"
	"sort"
	"strings"
)

// Currency is one allowlisted ISO 4217 code with its display symbol.
type Currency struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

// DefaultCurrency is used when a stored currency code is empty.
const DefaultCurrency = "RUB"

// currencies is the single allowlist for profiles, accounts, notifications and GET /currencies.
var currencies = []Currency{
	{Code: "RUB", Symbol: "₽"},
	{Code: "USD", Symbol: "$"},
	{Code: "EUR", Symbol: "€"},
	{Code: "CNY", Symbol: "¥"},
	{Code: "KZT", Symbol: "₸"},
	{Code: "BYN", Symbol: "Br"},
	{Code: "TRY", Symbol: "₺"},
	{Code: "GBP", Symbol: "£"},
}

// Currencies returns a copy of the allowlist in display order.
func Currencies() []Currency {
	out := make([]Currency, len(currencies))
	copy(out, currencies)
	return out
}

// Normalize trims and upper-cases a currency code.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Valid reports whether code (exact, already normalized) is in the allowlist.
func Valid(code string) bool {
	for _, c := range currencies {
		if c.Code == code {
			return true
		}
	}
	return false
}

// Symbol returns the display symbol for code; an unknown code is returned as is.
func Symbol(code string) string {
	for _, c := range currencies {
		if c.Code == code {
			return c.Symbol
		}
	}
	return code
}

// ErrCurrencyMismatch is returned when a link would join entities of different currencies.
var ErrCurrencyMismatch = errors.New("currency mismatch")

// ErrInvalidCurrency is returned when a code is not in the allowlist.
var ErrInvalidCurrency = errors.New("invalid currency")

// ProfileOrDefault returns a valid currency code, falling back to DefaultCurrency.
func ProfileOrDefault(code string) string {
	c := Normalize(code)
	if Valid(c) {
		return c
	}
	return DefaultCurrency
}

// ParseQueryCurrency treats empty as the profile currency; unknown codes are invalid.
func ParseQueryCurrency(raw, profile string) (string, error) {
	c := Normalize(raw)
	if c == "" {
		return ProfileOrDefault(profile), nil
	}
	if !Valid(c) {
		return "", ErrInvalidCurrency
	}
	return c, nil
}

// OrderWithProfileFirst returns unique codes with profile first (even if absent
// from codes), then the rest A–Z. Empty strings are dropped.
func OrderWithProfileFirst(profile string, codes []string) []string {
	profile = ProfileOrDefault(profile)
	seen := map[string]struct{}{profile: {}}
	rest := make([]string, 0, len(codes))
	for _, raw := range codes {
		c := Normalize(raw)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		rest = append(rest, c)
	}
	sort.Strings(rest)
	return append([]string{profile}, rest...)
}

// OrderPresentWithProfileFirst orders codes that actually appear: profile first
// if present, then A–Z. Unlike OrderWithProfileFirst it does not insert a missing profile.
func OrderPresentWithProfileFirst(profile string, codes []string) []string {
	profile = Normalize(profile)
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(codes))
	hasProfile := false
	for _, raw := range codes {
		c := Normalize(raw)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		if c == profile {
			hasProfile = true
			continue
		}
		uniq = append(uniq, c)
	}
	sort.Strings(uniq)
	if hasProfile {
		return append([]string{profile}, uniq...)
	}
	return uniq
}

// Rate returns toAmount/amount as an informational string with 6 fractional digits.
func Rate(amount, toAmount int64) string {
	if amount == 0 {
		return ""
	}
	return new(big.Rat).SetFrac64(toAmount, amount).FloatString(6)
}
