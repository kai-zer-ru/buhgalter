package subscription

import (
	"strings"
	"unicode"
)

const (
	MatchSameAccount        = "same_account"
	MatchSameAmount         = "same_amount"
	MatchNormDescription    = "norm_description"
	MatchSimilarDescription = "similar_description"
	MatchAmountNear         = "amount_near"
	MatchQuerySubstring     = "query_substring"
	MatchEmptyDescAmount    = "empty_desc_amount"
)

// Norm normalizes description for comparison.
func Norm(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		switch r {
		case '.', ',', '·', '—', '–', '-':
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, min(ins, sub))
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// Similarity returns 0..1 based on Levenshtein distance of normalized strings.
func Similarity(a, b string) float64 {
	na, nb := Norm(a), Norm(b)
	if na == "" && nb == "" {
		return 1
	}
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	dist := levenshtein(na, nb)
	maxLen := max(len(na), len(nb))
	if maxLen == 0 {
		return 1
	}
	return 1 - float64(dist)/float64(maxLen)
}

type scoreInput struct {
	TxAccountID   string
	TxAmount      int64
	TxDescription *string
	SubAccountID  string
	SubAmount     int64
	RefText       string
	Query         string
}

func scoreCandidate(in scoreInput) (int, []string) {
	score := 0
	reasons := make([]string, 0, 6)
	txDesc := ""
	if in.TxDescription != nil {
		txDesc = *in.TxDescription
	}
	normTx := Norm(txDesc)
	normRef := Norm(in.RefText)

	if in.TxAccountID == in.SubAccountID {
		score += 25
		reasons = append(reasons, MatchSameAccount)
	}
	if in.TxAmount == in.SubAmount {
		score += 25
		reasons = append(reasons, MatchSameAmount)
	}

	textSignal := false
	if normRef != "" && normTx != "" {
		if normTx == normRef {
			score += 40
			reasons = append(reasons, MatchNormDescription)
			textSignal = true
		} else {
			sim := Similarity(txDesc, in.RefText)
			if sim > 0 {
				add := int(40 * sim)
				if add > 0 {
					score += add
					reasons = append(reasons, MatchSimilarDescription)
					textSignal = true
				}
			}
		}
	} else if normRef == "" && normTx == "" && in.TxAmount == in.SubAmount {
		score += 30
		reasons = append(reasons, MatchEmptyDescAmount)
	}

	q := Norm(in.Query)
	if q != "" && strings.Contains(normTx, q) {
		score += 15
		reasons = append(reasons, MatchQuerySubstring)
		textSignal = true
	} else if normRef != "" && strings.Contains(normTx, normRef) {
		score += 15
		reasons = append(reasons, MatchQuerySubstring)
		textSignal = true
	}

	if textSignal && in.SubAmount > 0 {
		diff := in.TxAmount - in.SubAmount
		if diff < 0 {
			diff = -diff
		}
		if diff*100 <= in.SubAmount*20 {
			score += 10
			reasons = append(reasons, MatchAmountNear)
		}
	}

	if score > 100 {
		score = 100
	}
	return score, reasons
}

func refText(name string, description *string, override *string) string {
	if override != nil && strings.TrimSpace(*override) != "" {
		return strings.TrimSpace(*override)
	}
	if description != nil && strings.TrimSpace(*description) != "" {
		return strings.TrimSpace(*description)
	}
	return strings.TrimSpace(name)
}
