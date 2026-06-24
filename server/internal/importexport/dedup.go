package importexport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// DedupHash builds a deduplication key: date|amount|account|category|type.
func DedupHash(date string, amount int64, account, category, txType string) string {
	parts := []string{
		date,
		fmt.Sprintf("%d", amount),
		strings.ToLower(strings.TrimSpace(account)),
		strings.ToLower(strings.TrimSpace(category)),
		txType,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

// DedupSet tracks seen hashes for in-file and DB deduplication.
type DedupSet struct {
	seen map[string]struct{}
}

func NewDedupSet(existing []string) *DedupSet {
	s := &DedupSet{seen: make(map[string]struct{}, len(existing))}
	for _, h := range existing {
		s.seen[h] = struct{}{}
	}
	return s
}

func (s *DedupSet) Has(hash string) bool {
	_, ok := s.seen[hash]
	return ok
}

func (s *DedupSet) Add(hash string) {
	s.seen[hash] = struct{}{}
}
