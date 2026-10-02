package categoryrule

import (
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/category"
)

func TestMatchTextCategoryAndSilentApply(t *testing.T) {
	ctx, handle, userID, cashID, _, catID, _ := seed(t)
	sqlDB := handle.DB()

	p1 := "кофе"
	_, err := Create(ctx, sqlDB, userID, Input{
		Kind:       KindTextCategory,
		CategoryID: catID,
		Pattern:    &p1,
	})
	if err != nil {
		t.Fatal(err)
	}
	p2 := "утром"
	_, err = Create(ctx, sqlDB, userID, Input{
		Kind:       KindTextCategory,
		CategoryID: catID,
		Pattern:    &p2,
	})
	if err != nil {
		t.Fatal(err)
	}

	desc := "Купил кофе утром"
	ev, err := Evaluate(ctx, sqlDB, userID, EvaluateInput{Description: &desc})
	if err != nil {
		t.Fatal(err)
	}
	if !ev.Ambiguous || len(ev.TextMatches) != 2 {
		t.Fatalf("expected ambiguous 2 matches, got %+v", ev)
	}

	_ = Delete(ctx, sqlDB, userID, ev.TextMatches[1].RuleID)

	ev, err = Evaluate(ctx, sqlDB, userID, EvaluateInput{Description: &desc})
	if err != nil || ev.Ambiguous || len(ev.TextMatches) != 1 {
		t.Fatalf("single match: %+v err=%v", ev, err)
	}

	catPtr := (*string)(nil)
	subPtr := (*string)(nil)
	if err := ApplySilentTextCategory(ctx, sqlDB, userID, &catPtr, &subPtr, &desc); err != nil {
		t.Fatal(err)
	}
	if catPtr == nil || *catPtr != catID {
		t.Fatalf("applied category=%v", catPtr)
	}

	_, err = Create(ctx, sqlDB, userID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		DefaultAccountID: &cashID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cid := catID
	ev, err = Evaluate(ctx, sqlDB, userID, EvaluateInput{CategoryID: &cid})
	if err != nil || ev.AccountSuggestion == nil || ev.AccountSuggestion.AccountID != cashID {
		t.Fatalf("account suggestion: %+v err=%v", ev, err)
	}
}

func TestEvaluateAccountSuggestionRequiresSubcategoryWhenRuleHasOne(t *testing.T) {
	ctx, handle, userID, cashID, _, catID, subID := seed(t)
	sqlDB := handle.DB()

	_, err := Create(ctx, sqlDB, userID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		SubcategoryID:    &subID,
		DefaultAccountID: &cashID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cid := catID
	ev, err := Evaluate(ctx, sqlDB, userID, EvaluateInput{CategoryID: &cid})
	if err != nil {
		t.Fatal(err)
	}
	if ev.AccountSuggestion != nil {
		t.Fatalf("category only should not match sub-specific rule: %+v", ev)
	}
	sid := subID
	ev, err = Evaluate(ctx, sqlDB, userID, EvaluateInput{CategoryID: &cid, SubcategoryID: &sid})
	if err != nil || ev.AccountSuggestion == nil || ev.AccountSuggestion.AccountID != cashID {
		t.Fatalf("with subcategory: %+v err=%v", ev, err)
	}
}

func TestApplySilentSkipsWhenCategorySet(t *testing.T) {
	ctx, handle, userID, _, _, catID, _ := seed(t)
	sqlDB := handle.DB()
	pat := "test"
	_, err := Create(ctx, sqlDB, userID, Input{Kind: KindTextCategory, CategoryID: catID, Pattern: &pat})
	if err != nil {
		t.Fatal(err)
	}
	other := "other-cat"
	cats, _ := category.ListByUser(ctx, sqlDB, userID, "expense")
	if len(cats) > 1 {
		other = cats[1].ID
	}
	otherCat := other
	catPtr := &otherCat
	desc := "test payment"
	if err := ApplySilentTextCategory(ctx, sqlDB, userID, &catPtr, nil, &desc); err != nil {
		t.Fatal(err)
	}
	if catPtr == nil || *catPtr != other {
		t.Fatalf("should not override explicit category")
	}
}
