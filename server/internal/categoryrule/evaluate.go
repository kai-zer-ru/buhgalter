package categoryrule

import (
	"context"
	"database/sql"
	"strings"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type EvaluateInput struct {
	Description   *string
	CategoryID    *string
	SubcategoryID *string
}

type TextMatchRef struct {
	RuleID        string  `json:"rule_id"`
	Pattern       string  `json:"pattern"`
	CategoryID    string  `json:"category_id"`
	SubcategoryID *string `json:"subcategory_id,omitempty"`
	Priority      int     `json:"priority"`
}

type AccountSuggestion struct {
	RuleID    string `json:"rule_id"`
	AccountID string `json:"account_id"`
}

type EvaluateResult struct {
	TextMatches       []TextMatchRef     `json:"text_matches"`
	AccountSuggestion *AccountSuggestion `json:"account_suggestion"`
	Ambiguous         bool               `json:"ambiguous"`
}

func Evaluate(ctx context.Context, db *sql.DB, userID string, in EvaluateInput) (EvaluateResult, error) {
	return EvaluateOwned(ctx, db, []string{userID}, in)
}

func EvaluateOwned(ctx context.Context, db *sql.DB, ownerIDs []string, in EvaluateInput) (EvaluateResult, error) {
	out := EvaluateResult{TextMatches: []TextMatchRef{}}
	if len(ownerIDs) == 0 {
		return out, nil
	}

	textRows, err := queries(db).ListCategoryRulesByUserAndKind(ctx, sqlcdb.ListCategoryRulesByUserAndKindParams{
		OwnerIds: ownerIDs,
		Kind:     KindTextCategory,
	})
	if err != nil {
		return out, err
	}
	textRules := make([]Rule, 0, len(textRows))
	for _, row := range textRows {
		textRules = append(textRules, fromRow(row))
	}
	matches := matchTextRules(textRules, in.Description)
	for _, m := range matches {
		ref := TextMatchRef{
			RuleID:     m.ID,
			Pattern:    derefString(m.Pattern),
			CategoryID: m.CategoryID,
			Priority:   m.Priority,
		}
		if m.SubcategoryID != nil {
			ref.SubcategoryID = m.SubcategoryID
		}
		out.TextMatches = append(out.TextMatches, ref)
	}
	out.Ambiguous = len(matches) > 1

	catID := trimStringPtr(in.CategoryID)
	if catID != "" {
		sub := trimStringPtr(in.SubcategoryID)
		var subPtr *string
		if sub != "" {
			subPtr = &sub
		}
		accID, ruleID, err := suggestAccountFromRows(ctx, db, ownerIDs, catID, subPtr)
		if err != nil {
			return out, err
		}
		if accID != nil && ruleID != "" {
			out.AccountSuggestion = &AccountSuggestion{RuleID: ruleID, AccountID: *accID}
		}
	}

	return out, nil
}

// ApplySilentTextCategory sets category/subcategory when exactly one text rule matches.
func ApplySilentTextCategory(ctx context.Context, db *sql.DB, userID string, categoryID **string, subcategoryID **string, description *string) error {
	if categoryID == nil {
		return nil
	}
	if *categoryID != nil && strings.TrimSpace(**categoryID) != "" {
		return nil
	}
	rows, err := queries(db).ListCategoryRulesByUserAndKind(ctx, sqlcdb.ListCategoryRulesByUserAndKindParams{
		OwnerIds: []string{userID},
		Kind:     KindTextCategory,
	})
	if err != nil {
		return err
	}
	rules := make([]Rule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, fromRow(row))
	}
	matches := matchTextRules(rules, description)
	if len(matches) != 1 {
		return nil
	}
	w := matches[0]
	cid := w.CategoryID
	*categoryID = &cid
	if w.SubcategoryID != nil {
		sid := *w.SubcategoryID
		if subcategoryID != nil {
			*subcategoryID = &sid
		}
	}
	return nil
}

func matchTextRules(rules []Rule, description *string) []Rule {
	var out []Rule
	for _, r := range rules {
		if r.Pattern == nil || *r.Pattern == "" {
			continue
		}
		if textContainsPattern(description, *r.Pattern) {
			out = append(out, r)
		}
	}
	return out
}

func textContainsPattern(description *string, pattern string) bool {
	if description == nil {
		return false
	}
	desc := strings.TrimSpace(*description)
	if desc == "" {
		return false
	}
	pat := strings.TrimSpace(pattern)
	if pat == "" {
		return false
	}
	return strings.Contains(foldCase(desc), foldCase(pat))
}

func foldCase(s string) string {
	return strings.ToLower(s)
}

func suggestAccountFromRows(ctx context.Context, db *sql.DB, ownerIDs []string, categoryID string, subcategoryID *string) (*string, string, error) {
	rows, err := queries(db).ListCategoryRulesByUserAndKind(ctx, sqlcdb.ListCategoryRulesByUserAndKindParams{
		OwnerIds: ownerIDs,
		Kind:     KindCategoryAccount,
	})
	if err != nil {
		return nil, "", err
	}
	for _, row := range rows {
		if row.CategoryID != categoryID {
			continue
		}
		if row.SubcategoryID != nil {
			if subcategoryID == nil || *row.SubcategoryID != *subcategoryID {
				continue
			}
		}
		if row.DefaultAccountID != nil && *row.DefaultAccountID != "" {
			id := *row.DefaultAccountID
			return &id, row.ID, nil
		}
	}
	return nil, "", nil
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func trimStringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
