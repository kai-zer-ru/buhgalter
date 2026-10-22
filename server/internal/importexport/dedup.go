package importexport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// DedupKey uniquely identifies an imported operation.
type DedupKey struct {
	Date        string
	Time        string
	Amount      int64
	Account     string
	DestAccount string
	Category    string
	Subcategory string
	Type        string
	Description string
	Merchant    string
	Tags        string
	Commission  int64
}

// Hash returns a stable fingerprint for duplicate matching.
func (k DedupKey) Hash() string {
	parts := []string{
		strings.TrimSpace(k.Date),
		strings.TrimSpace(k.Time),
		fmt.Sprintf("%d", k.Amount),
		strings.ToLower(strings.TrimSpace(k.Account)),
		strings.ToLower(strings.TrimSpace(k.DestAccount)),
		strings.ToLower(strings.TrimSpace(k.Category)),
		strings.ToLower(strings.TrimSpace(k.Subcategory)),
		k.Type,
		strings.ToLower(strings.TrimSpace(k.Description)),
		strings.ToLower(strings.TrimSpace(k.Merchant)),
		normalizeTagKey(k.Tags),
		fmt.Sprintf("%d", k.Commission),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func normalizeTagKey(s string) string {
	names := parseTagList(s)
	if len(names) == 0 {
		return ""
	}
	lower := make([]string, len(names))
	for i, n := range names {
		lower[i] = strings.ToLower(n)
	}
	sort.Strings(lower)
	return strings.Join(lower, ",")
}

// DedupHash is a compatibility wrapper used by tests.
func DedupHash(date string, amount int64, account, category, txType string) string {
	return DedupKey{Date: date, Amount: amount, Account: account, Category: category, Type: txType, Time: "12:00:00"}.Hash()
}

// DedupBag matches file rows against existing DB occurrences (multiplicity-aware).
type DedupBag struct {
	remaining map[string]int
}

func NewDedupBag(existing []string) *DedupBag {
	b := &DedupBag{remaining: make(map[string]int, len(existing))}
	for _, h := range existing {
		b.remaining[h]++
	}
	return b
}

// SkipExisting reports whether this hash still has an unmatched DB row (should skip import).
func (b *DedupBag) SkipExisting(hash string) bool {
	n := b.remaining[hash]
	if n <= 0 {
		return false
	}
	b.remaining[hash] = n - 1
	return true
}
