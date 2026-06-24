package importexport

import "testing"

func TestBuildCategoryMappings(t *testing.T) {
	existing := map[string]string{
		"транспорт|expense": "id-transport",
		"прочие доходы|income": "id-income",
	}
	names := map[string]string{
		"транспорт|expense": "Транспорт",
		"прочие доходы|income": "Прочие доходы",
	}
	file := map[string]FileCategory{
		"Транспорт|expense": {Name: "Транспорт", Type: "expense"},
		"Связь|expense":     {Name: "Связь", Type: "expense"},
		"Прочие доходы|income": {Name: "Прочие доходы", Type: "income"},
	}
	mappings := buildCategoryMappings(file, existing, names)
	if len(mappings) != 3 {
		t.Fatalf("want 3 mappings, got %d", len(mappings))
	}
	byKey := make(map[string]CategoryMappingSuggestion)
	for _, m := range mappings {
		byKey[m.FileName+"|"+m.Type] = m
	}
	if byKey["Транспорт|expense"].Mode != "existing" {
		t.Fatal("Транспорт should map to existing")
	}
	if byKey["Связь|expense"].Mode != "create" {
		t.Fatal("Связь should default to create")
	}
	if byKey["Прочие доходы|income"].Mode != "existing" {
		t.Fatal("Прочие доходы should map to existing")
	}
}
