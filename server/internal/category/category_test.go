package category

import "testing"

func TestNormalizeCategoryOrderPinsSystemLast(t *testing.T) {
	existing := []Category{
		{ID: "a", Name: "Food", SortOrder: 1},
		{ID: "b", Name: "Transport", SortOrder: 2},
		{ID: "credit", Name: "Кредиты", SortOrder: 9998, IsSystem: true},
		{ID: "debt", Name: "Долги", SortOrder: 9999, IsSystem: true},
	}

	got, err := normalizeCategoryOrder(existing, []string{"debt", "b", "credit", "a"})
	if err != nil {
		t.Fatalf("normalizeCategoryOrder: %v", err)
	}
	want := []string{"b", "a", "credit", "debt"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestNormalizeCategoryOrderRequiresAllUserCategories(t *testing.T) {
	existing := []Category{
		{ID: "a", Name: "Food", SortOrder: 1},
		{ID: "debt", Name: "Долги", SortOrder: 9999, IsSystem: true},
	}
	_, err := normalizeCategoryOrder(existing, []string{"debt"})
	if err != ErrInvalidReorder {
		t.Fatalf("got %v want ErrInvalidReorder", err)
	}
}
