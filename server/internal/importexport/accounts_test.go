package importexport

import "testing"

func TestBuildAccountMappings(t *testing.T) {
	existing := map[string]string{
		"наличные": "id-cash",
		"яндекс":   "id-ya",
	}
	names := map[string]string{
		"наличные": "Наличные",
		"яндекс":   "Яндекс",
	}
	file := map[string]struct{}{
		"Наличные":  {},
		"Яндекс":    {},
		"Кредитка":  {},
	}
	mappings := buildAccountMappings(file, existing, names, nil)
	if len(mappings) != 3 {
		t.Fatalf("want 3 mappings, got %d", len(mappings))
	}
	byName := make(map[string]AccountMappingSuggestion)
	for _, m := range mappings {
		byName[m.FileName] = m
	}
	if byName["Наличные"].Mode != "existing" || byName["Наличные"].AccountID == nil {
		t.Fatal("Наличные should map to existing")
	}
	if byName["Яндекс"].Mode != "existing" {
		t.Fatal("Яндекс should map to existing")
	}
	if byName["Кредитка"].Mode != "create" {
		t.Fatal("Кредитка should default to create")
	}
}
