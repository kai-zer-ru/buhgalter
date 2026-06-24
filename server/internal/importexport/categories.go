package importexport

import (
	"sort"
	"strings"
)

// FileCategory is a category name from the import file with transaction type.
type FileCategory struct {
	Name string
	Type string // expense | income
}

func categoryMapLookupKey(name, catType string) string {
	return strings.TrimSpace(name) + "|" + catType
}

func collectFileCategories(rows []MappedRow) map[string]FileCategory {
	out := make(map[string]FileCategory)
	for _, m := range rows {
		switch m.CubuxType {
		case "Расходы":
			addFileCategory(out, m.Category, "expense")
		case "Доходы":
			addFileCategory(out, m.Category, "income")
		}
	}
	return out
}

func addFileCategory(out map[string]FileCategory, name, catType string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	key := categoryMapLookupKey(name, catType)
	out[key] = FileCategory{Name: name, Type: catType}
}

// buildCategoryMappings proposes mapping for every distinct category in the file.
func buildCategoryMappings(
	fileCategories map[string]FileCategory,
	existing map[string]string,
	existingNames map[string]string,
) []CategoryMappingSuggestion {
	keys := make([]string, 0, len(fileCategories))
	for k := range fileCategories {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]CategoryMappingSuggestion, 0, len(keys))
	for _, key := range keys {
		fc := fileCategories[key]
		if id, display, ok := findExistingCategory(fc.Name, fc.Type, existing, existingNames); ok {
			out = append(out, CategoryMappingSuggestion{
				FileName:     fc.Name,
				Type:         fc.Type,
				Mode:         "existing",
				CategoryID:   &id,
				CategoryName: &display,
			})
			continue
		}
		out = append(out, CategoryMappingSuggestion{
			FileName: fc.Name,
			Type:     fc.Type,
			Mode:     "create",
		})
	}
	return out
}

func findExistingCategory(name, catType string, existing, existingNames map[string]string) (id, display string, ok bool) {
	key := catKey(name, catType)
	if id, ok = existing[key]; ok {
		display = existingNames[key]
		if display == "" {
			display = name
		}
		return id, display, true
	}
	for k, cid := range existing {
		parts := strings.SplitN(k, "|", 2)
		if len(parts) == 2 && parts[1] == catType && strings.EqualFold(parts[0], name) {
			display = existingNames[k]
			if display == "" {
				display = name
			}
			return cid, display, true
		}
	}
	return "", "", false
}

func categoriesToCreateFromMap(mappings []CategoryMappingSuggestion) []string {
	out := make([]string, 0)
	for _, m := range mappings {
		if m.Mode == "create" {
			out = append(out, m.FileName+" ("+m.Type+")")
		}
	}
	return out
}
