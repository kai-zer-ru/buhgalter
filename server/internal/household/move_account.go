package household

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

var ErrLinkedPersonal = errors.New("share linked personal")

type ShareLink struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type LinkedPersonalError struct {
	Links []ShareLink
}

func (e *LinkedPersonalError) Error() string { return ErrLinkedPersonal.Error() }

func (e *LinkedPersonalError) Unwrap() error { return ErrLinkedPersonal }

type MovedCounts struct {
	Transactions  int64 `json:"transactions"`
	Credits       int64 `json:"credits"`
	Subscriptions int64 `json:"subscriptions"`
	Recurring     int64 `json:"recurring"`
	Budgets       int64 `json:"budgets"`
	Debts         int64 `json:"debts"`
	Templates     int64 `json:"templates"`
}

type MoveOpts struct {
	OriginalOwner *string
	AllowDeleted  bool
}

// MoveAccount reassigns an account and related rows from fromOwner to toOwner.
// Caller owns the transaction (db is typically *sql.Tx).
func MoveAccount(ctx context.Context, db sqlcdb.DBTX, accountID, fromOwner, toOwner string, opts MoveOpts) (MovedCounts, error) {
	var moved MovedCounts
	if accountID == "" || fromOwner == "" || toOwner == "" || fromOwner == toOwner {
		return moved, sql.ErrNoRows
	}
	q := sqlcdb.New(db)

	acc, err := q.GetAccountPaymentLink(ctx, sqlcdb.GetAccountPaymentLinkParams{ID: accountID, UserID: fromOwner})
	if errors.Is(err, sql.ErrNoRows) {
		return moved, sql.ErrNoRows
	}
	if err != nil {
		return moved, err
	}
	deleted := acc.Status == "deleted"
	if deleted && !opts.AllowDeleted {
		return moved, sql.ErrNoRows
	}

	if !deleted {
		if links, err := linkedPersonal(ctx, q, acc, accountID, fromOwner); err != nil {
			return moved, err
		} else if len(links) > 0 {
			return moved, &LinkedPersonalError{Links: links}
		}
	}

	wasPrimary := !deleted && acc.IsPrimary == 1 && acc.Status == "active"
	now := time.Now().UTC().Format(time.RFC3339)

	rm := newRemapper(db, q, fromOwner, toOwner, now)
	if err := rm.remapTransactions(ctx, accountID); err != nil {
		return moved, err
	}
	if !deleted {
		if err := rm.remapBudgets(ctx, accountID); err != nil {
			return moved, err
		}
		if err := rm.remapSubscriptions(ctx, accountID); err != nil {
			return moved, err
		}
		if err := rm.remapRecurring(ctx, accountID); err != nil {
			return moved, err
		}
		if err := rm.remapTemplates(ctx, accountID); err != nil {
			return moved, err
		}
		if err := rm.remapDebts(ctx, accountID); err != nil {
			return moved, err
		}
	}

	n, err := q.ReassignTransactionsByAccount(ctx, sqlcdb.ReassignTransactionsByAccountParams{
		UserID: toOwner, UpdatedAt: now, AccountID: accountID, UserID_2: fromOwner,
	})
	if err != nil {
		return moved, err
	}
	moved.Transactions = n

	if !deleted {
		if n, err = q.ReassignCreditsByDebitAccount(ctx, sqlcdb.ReassignCreditsByDebitAccountParams{
			UserID: toOwner, UpdatedAt: now, DebitAccountID: accountID, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Credits = n

		if n, err = q.ReassignSubscriptionsByAccount(ctx, sqlcdb.ReassignSubscriptionsByAccountParams{
			UserID: toOwner, UpdatedAt: now, AccountID: accountID, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Subscriptions = n

		if n, err = q.ReassignRecurringByAccount(ctx, sqlcdb.ReassignRecurringByAccountParams{
			UserID: toOwner, UpdatedAt: now, AccountID: accountID, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Recurring = n

		accPtr := accountID
		if n, err = q.ReassignTemplatesByAccount(ctx, sqlcdb.ReassignTemplatesByAccountParams{
			UserID: toOwner, UpdatedAt: now, AccountID: &accPtr, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Templates = n

		if n, err = q.ReassignBudgetsByAccount(ctx, sqlcdb.ReassignBudgetsByAccountParams{
			UserID: toOwner, UpdatedAt: now, AccountID: &accPtr, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Budgets = n

		if n, err = q.ReassignDebtsByOpenAccount(ctx, sqlcdb.ReassignDebtsByOpenAccountParams{
			UserID: toOwner, AccountID: accountID, UserID_2: fromOwner,
		}); err != nil {
			return moved, err
		}
		moved.Debts = n

		if _, err = q.ReassignReceiptImportsByAccount(ctx, sqlcdb.ReassignReceiptImportsByAccountParams{
			UserID: toOwner, UserID_2: fromOwner, AccountID: accountID,
		}); err != nil {
			return moved, err
		}

		accID := accountID
		if err := q.ClearCategoryRuleDefaultAccount(ctx, sqlcdb.ClearCategoryRuleDefaultAccountParams{
			DefaultAccountID: &accID, UserID: fromOwner,
		}); err != nil {
			return moved, err
		}
	}

	nAcc, err := q.ReassignAccountOwner(ctx, sqlcdb.ReassignAccountOwnerParams{
		UserID: toOwner, OriginalOwnerUserID: opts.OriginalOwner, UpdatedAt: now, ID: accountID, UserID_2: fromOwner,
	})
	if err != nil {
		return moved, err
	}
	if nAcc == 0 {
		return moved, sql.ErrNoRows
	}

	if wasPrimary {
		if err := q.ClearAccountPrimaryFlag(ctx, sqlcdb.ClearAccountPrimaryFlagParams{ID: accountID, UserID: toOwner}); err != nil {
			return moved, err
		}
		next, err := q.FirstActiveAccountID(ctx, fromOwner)
		if err == nil && next != "" {
			if err := q.SetAccountPrimary(ctx, sqlcdb.SetAccountPrimaryParams{ID: next, UserID: fromOwner}); err != nil {
				return moved, err
			}
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return moved, err
		}
	}

	return moved, nil
}

func linkedPersonal(ctx context.Context, q *sqlcdb.Queries, acc sqlcdb.GetAccountPaymentLinkRow, accountID, fromOwner string) ([]ShareLink, error) {
	var links []ShareLink
	if acc.PaymentAccountID != nil && strings.TrimSpace(*acc.PaymentAccountID) != "" && *acc.PaymentAccountID != accountID {
		oid, err := q.GetAccountOwnerID(ctx, *acc.PaymentAccountID)
		if err == nil && oid == fromOwner {
			links = append(links, ShareLink{Kind: "payment_account", ID: *acc.PaymentAccountID})
		}
	}
	if acc.AutoTopupEnabled == 1 && acc.AutoTopupSourceAccountID != nil && *acc.AutoTopupSourceAccountID != "" && *acc.AutoTopupSourceAccountID != accountID {
		oid, err := q.GetAccountOwnerID(ctx, *acc.AutoTopupSourceAccountID)
		if err == nil && oid == fromOwner {
			links = append(links, ShareLink{Kind: "auto_topup_source", ID: *acc.AutoTopupSourceAccountID})
		}
	}
	payAccs, err := q.ListCreditPaymentAccountIDs(ctx, sqlcdb.ListCreditPaymentAccountIDsParams{
		DebitAccountID: accountID, UserID: fromOwner, AccountID: accountID,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range payAccs {
		oid, err := q.GetAccountOwnerID(ctx, id)
		if err == nil && oid == fromOwner {
			links = append(links, ShareLink{Kind: "credit_payment_account", ID: id})
		}
	}
	settle, err := q.ListDebtSettleAccountIDs(ctx, sqlcdb.ListDebtSettleAccountIDsParams{
		AccountID: accountID, UserID: fromOwner, AccountID_2: accountID,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range settle {
		oid, err := q.GetAccountOwnerID(ctx, id)
		if err == nil && oid == fromOwner {
			links = append(links, ShareLink{Kind: "debt_settle_account", ID: id})
		}
	}
	return links, nil
}

type remapper struct {
	db        sqlcdb.DBTX
	q         *sqlcdb.Queries
	from, to  string
	now       string
	cats      map[string]string
	subs      map[string]string
	merchants map[string]string
	tags      map[string]string
	stores    map[string]string
	debtors   map[string]string
}

func newRemapper(db sqlcdb.DBTX, q *sqlcdb.Queries, from, to, now string) *remapper {
	return &remapper{
		db: db, q: q, from: from, to: to, now: now,
		cats: map[string]string{}, subs: map[string]string{}, merchants: map[string]string{},
		tags: map[string]string{}, stores: map[string]string{}, debtors: map[string]string{},
	}
}

func (rm *remapper) remapTransactions(ctx context.Context, accountID string) error {
	rows, err := rm.q.ListTransactionCatalogRefsByAccount(ctx, sqlcdb.ListTransactionCatalogRefsByAccountParams{
		AccountID: accountID, UserID: rm.from,
	})
	if err != nil {
		return err
	}
	owners := []string{rm.from}
	for _, row := range rows {
		cat, err := rm.mapCategory(ctx, row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		merch, err := rm.mapMerchant(ctx, row.MerchantID)
		if err != nil {
			return err
		}
		storeID, err := rm.mapStore(ctx, row.StoreID)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateTransactionCatalogRefs(ctx, sqlcdb.UpdateTransactionCatalogRefsParams{
			CategoryID: cat, SubcategoryID: sub, MerchantID: merch, StoreID: storeID,
			UpdatedAt: rm.now, ID: row.ID, UserID: rm.from,
		}); err != nil {
			return err
		}
		tagIDs, err := rm.q.ListTransactionTagIDs(ctx, row.ID)
		if err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			mapped, err := rm.mapTag(ctx, tagID)
			if err != nil {
				return err
			}
			if mapped != nil && *mapped != tagID {
				if err := rm.q.ReplaceTransactionTag(ctx, sqlcdb.ReplaceTransactionTagParams{
					TagID: *mapped, TransactionID: row.ID, TagID_2: tagID,
				}); err != nil {
					return err
				}
			}
		}
		_ = owners
	}
	return nil
}

func (rm *remapper) remapBudgets(ctx context.Context, accountID string) error {
	accPtr := accountID
	rows, err := rm.q.ListBudgetsCatalogByAccount(ctx, sqlcdb.ListBudgetsCatalogByAccountParams{AccountID: &accPtr, UserID: rm.from})
	if err != nil {
		return err
	}
	for _, row := range rows {
		cat, err := rm.mapCategory(ctx, row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateBudgetCatalog(ctx, sqlcdb.UpdateBudgetCatalogParams{
			CategoryID: cat, SubcategoryID: sub, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (rm *remapper) remapSubscriptions(ctx context.Context, accountID string) error {
	rows, err := rm.q.ListSubscriptionsCatalogByAccount(ctx, sqlcdb.ListSubscriptionsCatalogByAccountParams{AccountID: accountID, UserID: rm.from})
	if err != nil {
		return err
	}
	for _, row := range rows {
		sub, err := rm.mapSub(ctx, row.SubcategoryID, nil)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateSubscriptionCatalog(ctx, sqlcdb.UpdateSubscriptionCatalogParams{
			SubcategoryID: sub, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (rm *remapper) remapRecurring(ctx context.Context, accountID string) error {
	rows, err := rm.q.ListRecurringCatalogByAccount(ctx, sqlcdb.ListRecurringCatalogByAccountParams{AccountID: accountID, UserID: rm.from})
	if err != nil {
		return err
	}
	for _, row := range rows {
		cat, err := rm.mapCategory(ctx, &row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		var catID string
		if cat != nil {
			catID = *cat
		} else {
			catID = row.CategoryID
		}
		if err := rm.q.UpdateRecurringCatalog(ctx, sqlcdb.UpdateRecurringCatalogParams{
			CategoryID: catID, SubcategoryID: sub, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (rm *remapper) remapTemplates(ctx context.Context, accountID string) error {
	accPtr := accountID
	rows, err := rm.q.ListTemplatesCatalogByAccount(ctx, sqlcdb.ListTemplatesCatalogByAccountParams{AccountID: &accPtr, UserID: rm.from})
	if err != nil {
		return err
	}
	for _, row := range rows {
		cat, err := rm.mapCategory(ctx, row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		merch, err := rm.mapMerchant(ctx, row.MerchantID)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateTemplateCatalog(ctx, sqlcdb.UpdateTemplateCatalogParams{
			CategoryID: cat, SubcategoryID: sub, MerchantID: merch, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (rm *remapper) remapDebts(ctx context.Context, accountID string) error {
	rows, err := rm.q.ListDebtsOpenedOnAccount(ctx, sqlcdb.ListDebtsOpenedOnAccountParams{AccountID: accountID, UserID: rm.from})
	if err != nil {
		return err
	}
	for _, row := range rows {
		id, err := rm.mapDebtor(ctx, row.DebtorID)
		if err != nil {
			return err
		}
		if id != "" && id != row.DebtorID {
			if err := rm.q.UpdateDebtDebtor(ctx, sqlcdb.UpdateDebtDebtorParams{DebtorID: id, ID: row.ID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rm *remapper) remapAccountless(ctx context.Context) error {
	budgets, err := rm.q.ListBudgetsWithoutAccount(ctx, rm.from)
	if err != nil {
		return err
	}
	for _, row := range budgets {
		cat, err := rm.mapCategory(ctx, row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateBudgetCatalog(ctx, sqlcdb.UpdateBudgetCatalogParams{
			CategoryID: cat, SubcategoryID: sub, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	if err := rm.q.ReassignBudgetsWithoutAccount(ctx, sqlcdb.ReassignBudgetsWithoutAccountParams{
		UserID: rm.to, UpdatedAt: rm.now, UserID_2: rm.from,
	}); err != nil {
		return err
	}

	rules, err := rm.q.ListCategoryRulesByOwner(ctx, rm.from)
	if err != nil {
		return err
	}
	for _, row := range rules {
		cat, err := rm.mapCategory(ctx, &row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		var catID string
		if cat != nil {
			catID = *cat
		} else {
			catID = row.CategoryID
		}
		if err := rm.q.UpdateCategoryRuleCatalog(ctx, sqlcdb.UpdateCategoryRuleCatalogParams{
			CategoryID: catID, SubcategoryID: sub, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	if err := rm.q.ReassignCategoryRulesOwner(ctx, sqlcdb.ReassignCategoryRulesOwnerParams{
		UserID: rm.to, UpdatedAt: rm.now, UserID_2: rm.from,
	}); err != nil {
		return err
	}

	tmpls, err := rm.q.ListTemplatesWithoutAccount(ctx, rm.from)
	if err != nil {
		return err
	}
	for _, row := range tmpls {
		cat, err := rm.mapCategory(ctx, row.CategoryID)
		if err != nil {
			return err
		}
		sub, err := rm.mapSub(ctx, row.SubcategoryID, cat)
		if err != nil {
			return err
		}
		merch, err := rm.mapMerchant(ctx, row.MerchantID)
		if err != nil {
			return err
		}
		if err := rm.q.UpdateTemplateCatalog(ctx, sqlcdb.UpdateTemplateCatalogParams{
			CategoryID: cat, SubcategoryID: sub, MerchantID: merch, ID: row.ID,
		}); err != nil {
			return err
		}
	}
	if err := rm.q.ReassignTemplatesWithoutAccount(ctx, sqlcdb.ReassignTemplatesWithoutAccountParams{
		UserID: rm.to, UpdatedAt: rm.now, UserID_2: rm.from,
	}); err != nil {
		return err
	}

	debts, err := rm.q.ListDebtsWithoutTransactions(ctx, rm.from)
	if err != nil {
		return err
	}
	for _, row := range debts {
		id, err := rm.mapDebtor(ctx, row.DebtorID)
		if err != nil {
			return err
		}
		if id != "" && id != row.DebtorID {
			if err := rm.q.UpdateDebtDebtor(ctx, sqlcdb.UpdateDebtDebtorParams{DebtorID: id, ID: row.ID}); err != nil {
				return err
			}
		}
		if err := rm.q.ReassignDebtOwner(ctx, sqlcdb.ReassignDebtOwnerParams{
			UserID: rm.to, ID: row.ID, UserID_2: rm.from,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (rm *remapper) mapCategory(ctx context.Context, id *string) (*string, error) {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil, nil
	}
	srcID := *id
	if mapped, ok := rm.cats[srcID]; ok {
		return &mapped, nil
	}
	src, err := rm.q.GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return id, nil
	}
	if err != nil {
		return nil, err
	}
	var dest string
	if src.IsSystem == 1 {
		dest, err = systemCategoryID(ctx, rm.db, rm.to, src.Name, src.Type)
		if err != nil {
			return nil, err
		}
	} else {
		row, err := rm.q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
			UserID: rm.to, Name: src.Name, Type: src.Type,
		})
		if err == nil {
			dest = row.ID
		} else if errors.Is(err, sql.ErrNoRows) {
			dest = uuid.NewString()
			sortV, _ := rm.q.MaxCategorySortOrder(ctx, sqlcdb.MaxCategorySortOrderParams{UserID: rm.to, Type: src.Type})
			sort := int64FromAny(sortV) + 1
			if err := rm.q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
				ID: dest, UserID: rm.to, Name: src.Name, Type: src.Type, Icon: src.Icon,
				SortOrder: sort, IsPrimary: 0, IsSystem: 0, CreatedAt: rm.now,
			}); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	rm.cats[srcID] = dest
	return &dest, nil
}

func (rm *remapper) mapSub(ctx context.Context, id *string, destCat *string) (*string, error) {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil, nil
	}
	srcID := *id
	if mapped, ok := rm.subs[srcID]; ok {
		return &mapped, nil
	}
	src, err := rm.q.GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return id, nil
	}
	if err != nil {
		return nil, err
	}
	parent := destCat
	if parent == nil {
		mapped, err := rm.mapCategory(ctx, &src.CategoryID)
		if err != nil {
			return nil, err
		}
		parent = mapped
	}
	if parent == nil {
		return id, nil
	}
	subs, err := rm.q.ListSubcategoriesByCategory(ctx, *parent)
	if err != nil {
		return nil, err
	}
	for _, s := range subs {
		if strings.EqualFold(s.Name, src.Name) {
			rm.subs[srcID] = s.ID
			return &s.ID, nil
		}
	}
	dest := uuid.NewString()
	sortV, _ := rm.q.MaxSubcategorySortOrder(ctx, *parent)
	if err := rm.q.InsertSubcategory(ctx, sqlcdb.InsertSubcategoryParams{
		ID: dest, CategoryID: *parent, Name: src.Name, Icon: src.Icon,
		SortOrder: int64FromAny(sortV) + 1, CreatedAt: rm.now,
	}); err != nil {
		return nil, err
	}
	rm.subs[srcID] = dest
	return &dest, nil
}

func (rm *remapper) mapMerchant(ctx context.Context, id *string) (*string, error) {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil, nil
	}
	srcID := *id
	if mapped, ok := rm.merchants[srcID]; ok {
		return &mapped, nil
	}
	src, err := rm.q.GetMerchantByID(ctx, sqlcdb.GetMerchantByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return id, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := rm.q.GetMerchantByName(ctx, sqlcdb.GetMerchantByNameParams{UserID: rm.to, Name: src.Name})
	if err == nil {
		rm.merchants[srcID] = row.ID
		return &row.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	dest := uuid.NewString()
	if err := rm.q.InsertMerchant(ctx, sqlcdb.InsertMerchantParams{
		ID: dest, UserID: rm.to, Name: src.Name, Icon: src.Icon, CreatedAt: rm.now,
	}); err != nil {
		return nil, err
	}
	rm.merchants[srcID] = dest
	return &dest, nil
}

func (rm *remapper) mapTag(ctx context.Context, srcID string) (*string, error) {
	if srcID == "" {
		return nil, nil
	}
	if mapped, ok := rm.tags[srcID]; ok {
		return &mapped, nil
	}
	src, err := rm.q.GetTagByID(ctx, sqlcdb.GetTagByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return &srcID, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := rm.q.GetTagByName(ctx, sqlcdb.GetTagByNameParams{UserID: rm.to, Name: src.Name})
	if err == nil {
		rm.tags[srcID] = row.ID
		return &row.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	dest := uuid.NewString()
	if err := rm.q.InsertTag(ctx, sqlcdb.InsertTagParams{
		ID: dest, UserID: rm.to, Name: src.Name, CreatedAt: rm.now,
	}); err != nil {
		return nil, err
	}
	rm.tags[srcID] = dest
	return &dest, nil
}

func (rm *remapper) mapStore(ctx context.Context, id *string) (*string, error) {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil, nil
	}
	srcID := *id
	if mapped, ok := rm.stores[srcID]; ok {
		return &mapped, nil
	}
	src, err := rm.q.GetStoreByID(ctx, sqlcdb.GetStoreByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return id, nil
	}
	if err != nil {
		return nil, err
	}
	norm := strings.ToLower(strings.TrimSpace(src.Name))
	row, err := rm.q.GetStoreByNormalizedName(ctx, sqlcdb.GetStoreByNormalizedNameParams{UserID: rm.to, NameNormalized: norm})
	if err == nil {
		rm.stores[srcID] = row.ID
		return &row.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	dest := uuid.NewString()
	if err := rm.q.InsertStore(ctx, sqlcdb.InsertStoreParams{
		ID: dest, UserID: rm.to, Name: src.Name, NameNormalized: norm, CreatedAt: rm.now,
	}); err != nil {
		return nil, err
	}
	rm.stores[srcID] = dest
	return &dest, nil
}

func (rm *remapper) mapDebtor(ctx context.Context, srcID string) (string, error) {
	if srcID == "" {
		return "", nil
	}
	if mapped, ok := rm.debtors[srcID]; ok {
		return mapped, nil
	}
	src, err := rm.q.GetDebtorByID(ctx, sqlcdb.GetDebtorByIDParams{ID: srcID, OwnerIds: []string{rm.from}})
	if errors.Is(err, sql.ErrNoRows) {
		return srcID, nil
	}
	if err != nil {
		return "", err
	}
	row, err := rm.q.GetDebtorByName(ctx, sqlcdb.GetDebtorByNameParams{UserID: rm.to, Name: src.Name})
	if err == nil {
		rm.debtors[srcID] = row.ID
		return row.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	dest := uuid.NewString()
	if err := rm.q.InsertDebtor(ctx, sqlcdb.InsertDebtorParams{
		ID: dest, UserID: rm.to, Name: src.Name, CreatedAt: rm.now,
	}); err != nil {
		return "", err
	}
	rm.debtors[srcID] = dest
	return dest, nil
}

func systemCategoryID(ctx context.Context, db sqlcdb.DBTX, userID, name, catType string) (string, error) {
	q := sqlcdb.New(db)
	switch {
	case name == categoryseed.TransferCategoryName:
		return categoryseed.TransferCategoryID(ctx, db, userID)
	case name == categoryseed.CommissionCategoryName:
		return categoryseed.CommissionCategoryID(ctx, db, userID)
	case name == categoryseed.CreditCategoryName && catType == "expense":
		return categoryseed.CreditCategoryID(ctx, db, userID)
	case name == categoryseed.CreditCategoryName && catType == "income":
		return categoryseed.CreditIncomeCategoryID(ctx, db, userID)
	case name == categoryseed.DebtCategoryName:
		return categoryseed.DebtCategoryID(ctx, db, userID, catType)
	case name == categoryseed.SubscriptionsCategoryName:
		return categoryseed.SubscriptionsCategoryID(ctx, db, userID)
	default:
		row, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
			UserID: userID, Name: name, Type: catType,
		})
		if err != nil {
			return "", fmt.Errorf("system category %s: %w", name, err)
		}
		return row.ID, nil
	}
}

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
