package importexport

import (
	"sort"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/category"
)

type FileSubcategory struct {
	CategoryName    string
	SubcategoryName string
	Type            string // expense | income
}

func subcategoryMapLookupKey(categoryName, catType, subName string) string {
	return strings.TrimSpace(categoryName) + "|" + catType + "|" + strings.TrimSpace(subName)
}

func collectFileSubcategories(rows []MappedRow) map[string]FileSubcategory {
	out := make(map[string]FileSubcategory)
	for _, m := range rows {
		switch m.CubuxType {
		case "Расходы":
			addFileSubcategory(out, m.Category, "expense", m.Subcategory)
		case "Доходы":
			addFileSubcategory(out, m.Category, "income", m.Subcategory)
		}
	}
	return out
}

func addFileSubcategory(out map[string]FileSubcategory, categoryName, catType, subName string) {
	categoryName = strings.TrimSpace(categoryName)
	subName = strings.TrimSpace(subName)
	if categoryName == "" || subName == "" {
		return
	}
	key := subcategoryMapLookupKey(categoryName, catType, subName)
	out[key] = FileSubcategory{
		CategoryName:    categoryName,
		SubcategoryName: subName,
		Type:            catType,
	}
}

func buildSubcategoryMappings(
	fileSubcategories map[string]FileSubcategory,
	categoryMap map[string]CategoryMapEntry,
	res *resolver,
) []SubcategoryMappingSuggestion {
	keys := make([]string, 0, len(fileSubcategories))
	for k := range fileSubcategories {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]SubcategoryMappingSuggestion, 0, len(keys))
	for _, k := range keys {
		fs := fileSubcategories[k]
		s := SubcategoryMappingSuggestion{
			FileCategory:    fs.CategoryName,
			FileSubcategory: fs.SubcategoryName,
			Type:            fs.Type,
			Mode:            "create",
		}
		catID := resolveMappedCategoryID(fs.CategoryName, fs.Type, categoryMap, res)
		if catID != "" {
			if sub, ok := findExistingSubcategoryByCategoryID(catID, fs.SubcategoryName, res.subByCategoryID); ok {
				s.Mode = "existing"
				s.SubcategoryID = &sub.ID
				s.SubcategoryName = &sub.Name
			}
		}
		out = append(out, s)
	}
	return out
}

func resolveMappedCategoryID(categoryName, catType string, categoryMap map[string]CategoryMapEntry, res *resolver) string {
	lookup := categoryMapLookupKey(categoryName, catType)
	if entry, ok := categoryMap[lookup]; ok {
		if entry.Mode == "existing" && entry.CategoryID != "" {
			return entry.CategoryID
		}
		if entry.Mode == "create" {
			return ""
		}
	}
	if id, _, ok := findExistingCategory(categoryName, catType, res.categories, res.catNames); ok {
		return id
	}
	return ""
}

func findExistingSubcategoryByCategoryID(categoryID, subName string, byCategory map[string]map[string]category.Subcategory) (category.Subcategory, bool) {
	m := byCategory[categoryID]
	if len(m) == 0 {
		return category.Subcategory{}, false
	}
	sub, ok := m[strings.ToLower(strings.TrimSpace(subName))]
	return sub, ok
}

