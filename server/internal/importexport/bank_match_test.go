package importexport

import (
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/bank"
)

func testBanks() []bank.Bank {
	return []bank.Bank{
		{ID: "vtb", Name: "ВТБ"},
		{ID: "wbbank", Name: "WB Банк"},
		{ID: "yandex", Name: "Яндекс Банк"},
		{ID: "sberbank", Name: "Сбербанк"},
		{ID: "tinkoff", Name: "Т-Банк"},
	}
}

func TestMatchBank(t *testing.T) {
	banks := testBanks()
	cases := []struct {
		name   string
		wantID string
	}{
		{"ВТБ", "vtb"},
		{"втб", "vtb"},
		{"ВБ", "wbbank"},
		{"WB", "wbbank"},
		{"Яндекс", "yandex"},
		{"Яндекс Банк", "yandex"},
		{"Сбер", "sberbank"},
		{"Тинькофф", "tinkoff"},
		{"Наличные", ""},
		{"Кредитка", ""},
		{"Мой кошелёк", ""},
	}
	for _, tc := range cases {
		got := MatchBank(tc.name, banks)
		gotID := ""
		if got != nil {
			gotID = *got
		}
		if gotID != tc.wantID {
			t.Errorf("MatchBank(%q) = %q, want %q", tc.name, gotID, tc.wantID)
		}
	}
}

func TestMatchBankRequiresSeededBank(t *testing.T) {
	if got := MatchBank("ВТБ", nil); got != nil {
		t.Fatalf("expected nil without banks, got %v", got)
	}
}

func TestBuildAccountMappingsBankSuggestion(t *testing.T) {
	file := map[string]struct{}{
		"ВТБ":      {},
		"Кредитка": {},
	}
	mappings := buildAccountMappings(file, nil, nil, testBanks())
	byName := make(map[string]AccountMappingSuggestion)
	for _, m := range mappings {
		byName[m.FileName] = m
	}
	vtb := byName["ВТБ"]
	if vtb.Mode != "create" || vtb.AccountType == nil || *vtb.AccountType != "bank" {
		t.Fatalf("ВТБ: want create/bank, got %+v", vtb)
	}
	if vtb.BankID == nil || *vtb.BankID != "vtb" {
		t.Fatalf("ВТБ bank_id: got %+v", vtb.BankID)
	}
	credit := byName["Кредитка"]
	if credit.Mode != "create" || credit.AccountType == nil || *credit.AccountType != "cash" {
		t.Fatalf("Кредитка: want create/cash, got %+v", credit)
	}
}
