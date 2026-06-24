package importexport

import "testing"

func TestCollectFileSubcategories(t *testing.T) {
	rows := []MappedRow{
		{CubuxType: "Расходы", Category: "Food", Subcategory: "Lunch"},
		{CubuxType: "Доходы", Category: "Salary", Subcategory: "Bonus"},
	}
	subs := collectFileSubcategories(rows)
	if len(subs) != 2 {
		t.Fatalf("subs %d", len(subs))
	}
}

func TestSubcategoryMapLookupKey(t *testing.T) {
	key := subcategoryMapLookupKey(" Food ", "expense", " Lunch ")
	if key == "" {
		t.Fatal("expected key")
	}
}
