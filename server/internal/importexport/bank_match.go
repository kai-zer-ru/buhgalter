package importexport

import (
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/bank"
)

// bankAliases maps normalized abbreviations from Cubux exports to bank IDs.
var bankAliases = map[string]string{
	"втб":            "vtb",
	"сбер":           "sberbank",
	"сбербанк":       "sberbank",
	"тинькофф":       "tinkoff",
	"тинькоф":        "tinkoff",
	"т-банк":         "tinkoff",
	"tinkoff":        "tinkoff",
	"альфа":          "alfabank",
	"альфа-банк":     "alfabank",
	"альфабанк":      "alfabank",
	"вб":             "wbbank",
	"wb":             "wbbank",
	"wildberries":    "wbbank",
	"яндекс":         "yandex",
	"озон":           "ozon",
	"озон банк":      "ozon",
	"газпром":        "gazprombank",
	"газпромбанк":    "gazprombank",
	"райффайзен":     "raiffeisen",
	"росбанк":        "rosbank",
	"мкб":            "mkb",
	"рсхб":           "rshb",
	"россельхозбанк": "rshb",
	"открытие":       "open",
	"совкомбанк":     "sovcombank",
	"псб":            "psb",
	"уралсиб":        "uralsib",
	"хоум кредит":    "homecredit",
	"home credit":    "homecredit",
	"отп":            "otpbank",
	"отп банк":       "otpbank",
	"атб":            "atb",
}

func normalizeBankQuery(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "«", "")
	s = strings.ReplaceAll(s, "»", "")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "–", " ")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSuffix(s, " банк")
	s = strings.TrimSuffix(s, " bank")
	return strings.TrimSpace(s)
}

func isCashLikeName(q string) bool {
	switch q {
	case "наличные", "наличка", "cash", "кошелёк", "кошелек":
		return true
	default:
		return false
	}
}

func bankIDIfKnown(id string, banks []bank.Bank) *string {
	for _, b := range banks {
		if b.ID == id {
			out := b.ID
			return &out
		}
	}
	return nil
}

// MatchBank returns a bank ID when the file account name looks like a known bank.
func MatchBank(name string, banks []bank.Bank) *string {
	if len(banks) == 0 {
		return nil
	}
	q := normalizeBankQuery(name)
	if q == "" || isCashLikeName(q) {
		return nil
	}
	if id, ok := bankAliases[q]; ok {
		return bankIDIfKnown(id, banks)
	}
	for _, b := range banks {
		if strings.EqualFold(strings.TrimSpace(b.Name), strings.TrimSpace(name)) {
			id := b.ID
			return &id
		}
		bn := normalizeBankQuery(b.Name)
		if q == bn || strings.EqualFold(b.ID, q) {
			id := b.ID
			return &id
		}
	}
	var matches []string
	for _, b := range banks {
		bn := normalizeBankQuery(b.Name)
		if len(q) < 2 || len(bn) < 2 {
			continue
		}
		if strings.HasPrefix(bn, q) || strings.HasPrefix(q, bn) {
			matches = append(matches, b.ID)
		}
	}
	if len(matches) == 1 {
		id := matches[0]
		return &id
	}
	return nil
}

func suggestCreateAccount(name string, banks []bank.Bank) (accType string, bankID *string) {
	if id := MatchBank(name, banks); id != nil {
		return "bank", id
	}
	return "cash", nil
}
