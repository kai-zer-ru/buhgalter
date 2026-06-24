package importexport

import (
	"sort"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/bank"
)

// buildAccountMappings proposes mapping for every distinct account name in the file.
// Exact case-insensitive name match → existing; otherwise → create (bank if name matches a known bank).
func buildAccountMappings(
	fileAccounts map[string]struct{},
	existingByLower map[string]string,
	existingNames map[string]string,
	banks []bank.Bank,
) []AccountMappingSuggestion {
	names := make([]string, 0, len(fileAccounts))
	for name := range fileAccounts {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]AccountMappingSuggestion, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if id, ok := existingByLower[key]; ok {
			display := existingNames[key]
			out = append(out, AccountMappingSuggestion{
				FileName:    name,
				Mode:        "existing",
				AccountID:   &id,
				AccountName: &display,
			})
			continue
		}
		accType, bankID := suggestCreateAccount(name, banks)
		out = append(out, AccountMappingSuggestion{
			FileName:    name,
			Mode:        "create",
			AccountType: &accType,
			BankID:      bankID,
		})
	}
	return out
}

func collectFileAccounts(rows []MappedRow) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range rows {
		for _, name := range fileAccountNames(m) {
			name = strings.TrimSpace(name)
			if name != "" {
				out[name] = struct{}{}
			}
		}
	}
	return out
}

func fileAccountNames(m MappedRow) []string {
	switch m.CubuxType {
	case "Расходы":
		return []string{m.DebitAccount}
	case "Доходы":
		return []string{m.CreditAccount}
	case "Перевод":
		return []string{m.DebitAccount, m.CreditAccount}
	default:
		return nil
	}
}

func accountsToCreateFromMap(mappings []AccountMappingSuggestion) []string {
	out := make([]string, 0)
	for _, m := range mappings {
		if m.Mode == "create" {
			out = append(out, m.FileName)
		}
	}
	return out
}
