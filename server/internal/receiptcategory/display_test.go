package receiptcategory

import "testing"

func TestAggregateDisplay_sameCategoryDifferentSubs(t *testing.T) {
	cat, sub := aggregateDisplay([]resolvedLine{
		{categoryName: "Еда", subcategoryName: "Молочное"},
		{categoryName: "Еда", subcategoryName: "Хлеб"},
	})
	if cat != "Еда" || sub != nil {
		t.Fatalf("want category only, got %q sub=%v", cat, sub)
	}
}

func TestAggregateDisplay_multipleCategories(t *testing.T) {
	cat, sub := aggregateDisplay([]resolvedLine{
		{categoryName: "Еда", subcategoryName: "Молочное"},
		{categoryName: "Быт", subcategoryName: ""},
	})
	if cat != "Быт, Еда" || sub != nil {
		t.Fatalf("want sorted category list, got %q sub=%v", cat, sub)
	}
}

func TestAggregateDisplay_sameCategorySameSub(t *testing.T) {
	cat, sub := aggregateDisplay([]resolvedLine{
		{categoryName: "Еда", subcategoryName: "Молочное"},
		{categoryName: "Еда", subcategoryName: "Молочное"},
	})
	if cat != "Еда" || sub == nil || *sub != "Молочное" {
		t.Fatalf("want category+sub, got %q sub=%v", cat, sub)
	}
}
