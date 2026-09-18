package importexport

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/budget"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
	"github.com/kai-zer-ru/buhgalter/internal/transactiontemplate"
)

type nativeAccountMeta struct {
	Name, Type, Bank, PaymentAccount, AutoTopupSource, Status        string
	InitialBalance, CreditLimit, AutoTopupThreshold, AutoTopupTarget int64
	IsPrimary, AutoTopupEnabled, hasLimit                            bool
}

func (r *resolver) loadNativeMeta(nf *NativeFile) {
	if nf == nil {
		return
	}
	r.nativeAccounts = make(map[string]nativeAccountMeta)
	table, ok := nf.Sections[sectionAccounts]
	if !ok {
		return
	}
	for _, row := range table.Rows {
		name := nativeCell(table, row, "name")
		if name == "" {
			continue
		}
		initial, _ := parseOptionalAmount(nativeCell(table, row, "initial_balance"))
		limit, err := parseOptionalAmount(nativeCell(table, row, "credit_limit"))
		thr, _ := parseOptionalAmount(nativeCell(table, row, "auto_topup_threshold"))
		tgt, _ := parseOptionalAmount(nativeCell(table, row, "auto_topup_target"))
		meta := nativeAccountMeta{
			Name: name, Type: nativeCell(table, row, "type"), Bank: nativeCell(table, row, "bank"),
			PaymentAccount: nativeCell(table, row, "payment_account"), AutoTopupSource: nativeCell(table, row, "auto_topup_source"),
			Status: nativeCell(table, row, "status"), InitialBalance: initial, CreditLimit: limit,
			AutoTopupThreshold: thr, AutoTopupTarget: tgt,
			IsPrimary:        parseCSVBool(nativeCell(table, row, "is_primary")),
			AutoTopupEnabled: parseCSVBool(nativeCell(table, row, "auto_topup_enabled")),
			hasLimit:         err == nil && strings.TrimSpace(nativeCell(table, row, "credit_limit")) != "",
		}
		r.nativeAccounts[strings.ToLower(name)] = meta
	}
}

func enrichAccountMappings(mappings []AccountMappingSuggestion, native map[string]nativeAccountMeta, banks []bank.Bank) []AccountMappingSuggestion {
	if len(native) == 0 {
		return mappings
	}
	out := append([]AccountMappingSuggestion(nil), mappings...)
	for i, m := range out {
		meta, ok := native[strings.ToLower(m.FileName)]
		if !ok || m.Mode != "create" {
			continue
		}
		if meta.Type != "" {
			t := meta.Type
			out[i].AccountType = &t
		}
		if meta.Bank != "" {
			out[i].BankID = MatchBank(meta.Bank, banks)
		}
		if meta.hasLimit {
			lim := FormatCubuxAmount(meta.CreditLimit)
			out[i].CreditLimit = &lim
		}
		if meta.InitialBalance != 0 {
			ib := FormatCubuxAmount(meta.InitialBalance)
			out[i].InitialBalance = &ib
		}
	}
	seen := make(map[string]struct{}, len(out))
	for _, m := range out {
		seen[strings.ToLower(m.FileName)] = struct{}{}
	}
	for _, meta := range native {
		if _, ok := seen[strings.ToLower(meta.Name)]; ok {
			continue
		}
		t := meta.Type
		if t == "" {
			t = "cash"
		}
		sug := AccountMappingSuggestion{FileName: meta.Name, Mode: "create", AccountType: &t}
		if meta.Bank != "" {
			sug.BankID = MatchBank(meta.Bank, banks)
		}
		if meta.hasLimit {
			lim := FormatCubuxAmount(meta.CreditLimit)
			sug.CreditLimit = &lim
		}
		if meta.InitialBalance != 0 {
			ib := FormatCubuxAmount(meta.InitialBalance)
			sug.InitialBalance = &ib
		}
		out = append(out, sug)
	}
	return out
}

func (r *resolver) ensureNativeAccounts(ctx context.Context) error {
	if len(r.nativeAccounts) == 0 {
		return nil
	}
	for _, want := range []string{"cash", "bank", "credit_card"} {
		for _, meta := range r.nativeAccounts {
			typ := meta.Type
			if typ == "" {
				typ = "cash"
			}
			if typ != want {
				continue
			}
			if _, err := r.resolveAccount(ctx, meta.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resolver) finalizeNativeAccounts(ctx context.Context) error {
	if r.dryRun || len(r.nativeAccounts) == 0 {
		return nil
	}
	for _, meta := range r.nativeAccounts {
		id, ok := r.fileAccounts[strings.ToLower(meta.Name)]
		if !ok || id == "" {
			continue
		}
		acc, err := account.GetByID(ctx, r.db, r.userID, id)
		if err != nil {
			return err
		}
		var paymentID *string
		if meta.PaymentAccount != "" {
			if pid, ok := r.fileAccounts[strings.ToLower(meta.PaymentAccount)]; ok && pid != "" {
				paymentID = &pid
			}
		}
		if paymentID != nil || meta.AutoTopupEnabled {
			in := account.UpdateInput{Name: acc.Name, BankID: acc.BankID, PaymentAccountID: paymentID}
			if meta.AutoTopupEnabled {
				src := ""
				if meta.AutoTopupSource != "" {
					if sid, ok := r.fileAccounts[strings.ToLower(meta.AutoTopupSource)]; ok {
						src = sid
					}
				}
				in.AutoTopup = &account.AutoTopupInput{
					Enabled: true, Threshold: meta.AutoTopupThreshold, Target: meta.AutoTopupTarget, SourceAccountID: src,
				}
			}
			_, _ = account.Update(ctx, r.db, r.userID, id, in)
		}
		if meta.IsPrimary && acc.Type != "credit_card" {
			_, _ = account.SetPrimary(ctx, r.db, r.userID, id)
		}
		if meta.Status == "archived" {
			_, _ = account.SetStatus(ctx, r.db, r.userID, id, "archived")
		}
	}
	return nil
}

func (r *resolver) importNativeCatalogs(ctx context.Context, nf *NativeFile) error {
	if nf == nil || r.dryRun {
		return nil
	}
	if err := importNativeCategories(ctx, r, nf); err != nil {
		return err
	}
	if err := importNativeSubcategories(ctx, r, nf); err != nil {
		return err
	}
	if table, ok := nf.Sections[sectionMerchants]; ok {
		for _, row := range table.Rows {
			name := nativeCell(table, row, "name")
			if name == "" {
				continue
			}
			if _, err := merchant.Create(ctx, r.db, r.userID, name, nativeCell(table, row, "icon")); err != nil && err != merchant.ErrNameTaken {
				return err
			}
		}
	}
	if table, ok := nf.Sections[sectionTags]; ok {
		for _, row := range table.Rows {
			name := nativeCell(table, row, "name")
			if name == "" {
				continue
			}
			if _, err := tag.Create(ctx, r.db, r.userID, name); err != nil && err != tag.ErrNameTaken {
				return err
			}
		}
	}
	return nil
}

func importNativeCategories(ctx context.Context, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionCategories]
	if !ok {
		return nil
	}
	for _, row := range table.Rows {
		name := nativeCell(table, row, "name")
		catType := nativeCell(table, row, "type")
		icon := nativeCell(table, row, "icon")
		if name == "" || catType == "" {
			continue
		}
		if parseCSVBool(nativeCell(table, row, "is_system")) {
			if id, ok := r.categories[catKey(name, catType)]; ok && icon != "" {
				_ = sqlcdb.New(r.db).UpdateSystemCategoryIcon(ctx, sqlcdb.UpdateSystemCategoryIconParams{
					Icon: icon, ID: id, UserID: r.userID,
				})
			}
			continue
		}
		if _, ok := r.categories[catKey(name, catType)]; ok {
			continue
		}
		sortOrder, _ := strconv.Atoi(nativeCell(table, row, "sort_order"))
		created, err := category.Create(ctx, r.db, r.userID, name, catType, icon, sortOrder)
		if err != nil {
			if err == category.ErrNameTaken {
				continue
			}
			return err
		}
		r.categories[catKey(created.Name, catType)] = created.ID
		r.catNames[catKey(created.Name, catType)] = created.Name
		r.catIsSystem[created.ID] = created.IsSystem
	}
	return nil
}

func importNativeSubcategories(ctx context.Context, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionSubcats]
	if !ok {
		return nil
	}
	q := sqlcdb.New(r.db)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range table.Rows {
		catName := nativeCell(table, row, "category")
		catType := nativeCell(table, row, "category_type")
		name := nativeCell(table, row, "name")
		icon := nativeCell(table, row, "icon")
		if catName == "" || name == "" {
			continue
		}
		catID, ok := r.categories[catKey(catName, catType)]
		if !ok {
			continue
		}
		if _, exists := r.subcategories[subKey(catName, name)]; exists {
			continue
		}
		if r.catIsSystem[catID] {
			id := uuid.NewString()
			if icon == "" {
				icon = "default"
			}
			if err := q.InsertSubcategory(ctx, sqlcdb.InsertSubcategoryParams{
				ID: id, CategoryID: catID, Name: name, Icon: icon, SortOrder: 0, CreatedAt: now,
			}); err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					continue
				}
				return err
			}
			r.subcategories[subKey(catName, name)] = id
			r.rememberSubcategory(catID, catName, name, id)
			continue
		}
		created, err := category.CreateSubcategory(ctx, r.db, r.userID, catID, name, icon)
		if err != nil {
			if err == category.ErrSubNameTaken || err == category.ErrSystemCategory {
				continue
			}
			return err
		}
		r.subcategories[subKey(catName, name)] = created.ID
		r.rememberSubcategory(catID, catName, name, created.ID)
	}
	return nil
}

func (r *resolver) rememberSubcategory(catID, catName, name, id string) {
	r.subcategories[subKey(catName, name)] = id
	byCategory, ok := r.subByCategoryID[catID]
	if !ok {
		byCategory = make(map[string]category.Subcategory)
		r.subByCategoryID[catID] = byCategory
	}
	byCategory[strings.ToLower(strings.TrimSpace(name))] = category.Subcategory{ID: id, Name: name, CategoryID: catID}
}

func importNativeEntities(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile, report *Report) {
	if nf == nil || r.dryRun {
		return
	}
	steps := []struct {
		name string
		fn   func() error
	}{
		{"profile", func() error { return importNativeProfile(ctx, db, r, nf) }},
		{"credits", func() error { return importNativeCredits(ctx, db, r, nf) }},
		{"debts", func() error { return importNativeDebts(ctx, db, r, nf) }},
		{"subscriptions", func() error { return importNativeSubscriptions(ctx, db, r, nf) }},
		{"recurring", func() error { return importNativeRecurring(ctx, db, r, nf) }},
		{"budgets", func() error { return importNativeBudgets(ctx, db, r, nf) }},
		{"templates", func() error { return importNativeTemplates(ctx, db, r, nf) }},
	}
	for _, step := range steps {
		if err := step.fn(); err != nil {
			report.Logs = append(report.Logs, step.name+": "+err.Error())
			report.Errors = append(report.Errors, RowError{Message: step.name + ": " + err.Error()})
		}
	}
}

func importNativeProfile(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionProfile]
	if !ok || len(table.Rows) == 0 {
		return nil
	}
	row := table.Rows[0]
	tz := strings.TrimSpace(nativeCell(table, row, "timezone"))
	if tz == "" {
		return nil
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("timezone %q: %w", tz, err)
	}
	u, err := auth.LoadUser(ctx, db, r.userID)
	if err != nil {
		return err
	}
	lang := orDefault(nativeCell(table, row, "language"), u.Language)
	currency := orDefault(nativeCell(table, row, "currency"), u.Currency)
	return auth.UpdateUserProfile(ctx, db, r.userID, u.DisplayName, lang, currency, tz, u.Theme)
}

func remapTx(r *resolver, old string) *string {
	old = strings.TrimSpace(old)
	if old == "" {
		return nil
	}
	if id, ok := r.txIDMap[old]; ok {
		return &id
	}
	return nil
}

func accountIDByName(r *resolver, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty account")
	}
	if id, ok := r.fileAccounts[strings.ToLower(name)]; ok && id != "" {
		return id, nil
	}
	if id, ok := r.accounts[strings.ToLower(name)]; ok {
		return id, nil
	}
	return "", fmt.Errorf("account %q not found", name)
}

func importNativeCredits(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionCredits]
	if !ok {
		return nil
	}
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	idMap := make(map[string]string)
	for _, row := range table.Rows {
		oldID := nativeCell(table, row, "id")
		acct, err := accountIDByName(r, nativeCell(table, row, "debit_account"))
		if err != nil {
			return err
		}
		principal, err := parseOptionalAmount(nativeCell(table, row, "principal"))
		if err != nil {
			return err
		}
		down, _ := parseOptionalAmount(nativeCell(table, row, "down_payment"))
		paid, _ := parseOptionalAmount(nativeCell(table, row, "paid_amount"))
		monthly, _ := parseOptionalAmount(nativeCell(table, row, "monthly_payment"))
		term, _ := strconv.ParseInt(nativeCell(table, row, "term_months"), 10, 64)
		name := strings.TrimSpace(nativeCell(table, row, "name"))
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		var bankID *string
		if bn := nativeCell(table, row, "bank"); bn != "" {
			bankID = MatchBank(bn, r.banks)
		}
		prop := parseOptionalIntAmount(nativeCell(table, row, "property_price"))
		newID := uuid.NewString()
		debitTime := strings.TrimSpace(nativeCell(table, row, "debit_time_local"))
		var debitTimePtr *string
		if debitTime != "" {
			debitTimePtr = &debitTime
		}
		closed := strings.TrimSpace(nativeCell(table, row, "closed_at"))
		var closedPtr *string
		if closed != "" {
			closedPtr = &closed
		}
		status := orDefault(nativeCell(table, row, "status"), "active")
		if err := q.InsertCredit(ctx, sqlcdb.InsertCreditParams{
			ID: newID, UserID: r.userID, Name: namePtr, CreditKind: orDefault(nativeCell(table, row, "kind"), "consumer"),
			PrincipalAmount: principal, PropertyPrice: prop, DownPayment: down,
			DownPaymentAffectsBalance: boolToInt(parseCSVBool(nativeCell(table, row, "down_payment_affects_balance"))),
			DownPaymentTransactionID:  remapTx(r, nativeCell(table, row, "down_payment_tx_id")),
			PrincipalAffectsBalance:   boolToInt(parseCSVBool(nativeCell(table, row, "principal_affects_balance"))),
			PrincipalTransactionID:    remapTx(r, nativeCell(table, row, "principal_tx_id")),
			IssueDate:                 nativeCell(table, row, "issue_date"), TermMonths: term,
			InterestRate:    parseOptionalFloat(nativeCell(table, row, "interest_rate")),
			PaymentInterval: orDefault(nativeCell(table, row, "payment_interval"), "month"),
			PaidAmount:      paid, MonthlyPayment: monthly, DebitAccountID: acct,
			DebitTimeLocal: debitTimePtr, BankID: bankID,
			BankIDLocked:       boolToInt(parseCSVBool(nativeCell(table, row, "bank_locked"))),
			AddedRetroactively: boolToInt(parseCSVBool(nativeCell(table, row, "added_retroactively"))),
			RecordedAt:         orDefault(nativeCell(table, row, "recorded_at"), now),
			Status:             status, ClosedAt: closedPtr, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		if oldID != "" {
			idMap[oldID] = newID
		}
	}
	pays, ok := nf.Sections[sectionCreditPays]
	if !ok {
		return nil
	}
	for _, row := range pays.Rows {
		creditID, ok := idMap[nativeCell(pays, row, "credit_id")]
		if !ok {
			continue
		}
		amt, err := parseOptionalAmount(nativeCell(pays, row, "amount"))
		if err != nil {
			return err
		}
		if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
			ID: uuid.NewString(), CreditID: creditID, TransactionID: remapTx(r, nativeCell(pays, row, "transaction_id")),
			Amount: amt, PaymentDate: nativeCell(pays, row, "payment_date"),
			Kind:             orDefault(nativeCell(pays, row, "kind"), "scheduled"),
			IsApplied:        boolToInt(parseCSVBool(nativeCell(pays, row, "is_applied"))),
			ExcludeFromStats: boolToInt(parseCSVBool(nativeCell(pays, row, "exclude_from_stats"))),
			CreatedAt:        now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func importNativeDebts(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	debtorMap := map[string]string{}
	if table, ok := nf.Sections[sectionDebtors]; ok {
		for _, row := range table.Rows {
			oldID := nativeCell(table, row, "id")
			name := nativeCell(table, row, "name")
			if name == "" {
				continue
			}
			newID := uuid.NewString()
			if err := q.InsertDebtor(ctx, sqlcdb.InsertDebtorParams{
				ID: newID, UserID: r.userID, Name: name, CreatedAt: now,
			}); err != nil {
				if !strings.Contains(err.Error(), "UNIQUE") {
					return err
				}
				existing, err := q.GetDebtorByName(ctx, sqlcdb.GetDebtorByNameParams{UserID: r.userID, Name: name})
				if err != nil {
					return err
				}
				newID = existing.ID
			}
			if oldID != "" {
				debtorMap[oldID] = newID
			}
			debtorMap[strings.ToLower(name)] = newID
		}
	}
	debtMap := map[string]string{}
	if table, ok := nf.Sections[sectionDebts]; ok {
		for _, row := range table.Rows {
			oldID := nativeCell(table, row, "id")
			debtorID := debtorMap[nativeCell(table, row, "debtor_id")]
			if debtorID == "" {
				debtorID = debtorMap[strings.ToLower(nativeCell(table, row, "debtor"))]
			}
			if debtorID == "" {
				continue
			}
			amt, err := parseOptionalAmount(nativeCell(table, row, "amount"))
			if err != nil {
				return err
			}
			newID := uuid.NewString()
			if err := q.InsertDebt(ctx, sqlcdb.InsertDebtParams{
				ID: newID, UserID: r.userID, DebtorID: debtorID, Direction: nativeCell(table, row, "direction"),
				Amount: amt, AffectsBalance: boolToInt(parseCSVBool(nativeCell(table, row, "affects_balance"))),
				DebtDate: nativeCell(table, row, "debt_date"), DueDate: nativeCell(table, row, "due_date"),
				Description: strPtr(nativeCell(table, row, "description")), TransactionID: remapTx(r, nativeCell(table, row, "transaction_id")),
				IsSettled: boolToInt(parseCSVBool(nativeCell(table, row, "is_settled"))),
				SettledAt: strPtr(nativeCell(table, row, "settled_at")), CreatedAt: now,
			}); err != nil {
				return err
			}
			if oldID != "" {
				debtMap[oldID] = newID
			}
		}
	}
	if table, ok := nf.Sections[sectionDebtLinks]; ok {
		for _, row := range table.Rows {
			debtID := debtMap[nativeCell(table, row, "debt_id")]
			txID := remapTx(r, nativeCell(table, row, "transaction_id"))
			if debtID == "" || txID == nil {
				continue
			}
			_ = q.InsertDebtTransactionLink(ctx, sqlcdb.InsertDebtTransactionLinkParams{
				DebtID: debtID, TransactionID: *txID, Role: orDefault(nativeCell(table, row, "role"), "open"),
			})
		}
	}
	return nil
}

func importNativeSubscriptions(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionSubs]
	if !ok {
		return nil
	}
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	nameToID := map[string]string{}
	for _, row := range table.Rows {
		acct, err := accountIDByName(r, nativeCell(table, row, "account"))
		if err != nil {
			return err
		}
		amt, err := parseOptionalAmount(nativeCell(table, row, "amount"))
		if err != nil {
			return err
		}
		name := nativeCell(table, row, "name")
		var subID *string
		if sn := nativeCell(table, row, "subcategory"); sn != "" {
			if id, ok := r.subcategories[subKey("Подписки", sn)]; ok {
				subID = &id
			}
		}
		upcoming := nativeCell(table, row, "upcoming_run_ats")
		if upcoming == "" {
			upcoming = "[]"
		}
		newID := uuid.NewString()
		if err := q.InsertSubscription(ctx, sqlcdb.InsertSubscriptionParams{
			ID: newID, UserID: r.userID, Name: name, Description: strPtr(nativeCell(table, row, "description")),
			Icon: strPtr(nativeCell(table, row, "icon")), WebsiteUrl: strPtr(nativeCell(table, row, "website_url")),
			Amount: amt, AccountID: acct, SubcategoryID: subID, Period: nativeCell(table, row, "period"),
			Weekday: parseOptionalInt(nativeCell(table, row, "weekday")), DayOfMonth: parseOptionalInt(nativeCell(table, row, "day_of_month")),
			StartDate: nativeCell(table, row, "start_date"), TimeLocal: orDefault(nativeCell(table, row, "time_local"), "08:00"),
			NextRunAt: nativeCell(table, row, "next_run_at"), UpcomingRunAts: upcoming, LastRunAt: strPtr(nativeCell(table, row, "last_run_at")),
			Active: boolToInt(parseCSVBool(nativeCell(table, row, "active"))), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		nameToID[strings.ToLower(name)] = newID
	}
	for newTx, subName := range r.txSubName {
		subID, ok := nameToID[strings.ToLower(subName)]
		if !ok {
			continue
		}
		_ = q.SetTransactionSubscriptionID(ctx, sqlcdb.SetTransactionSubscriptionIDParams{
			SubscriptionID: &subID, UpdatedAt: now, ID: newTx, UserID: r.userID,
		})
	}
	return nil
}

func importNativeRecurring(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionRecurring]
	if !ok {
		return nil
	}
	q := sqlcdb.New(db)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range table.Rows {
		acct, err := accountIDByName(r, nativeCell(table, row, "account"))
		if err != nil {
			return err
		}
		catType := nativeCell(table, row, "type")
		catID, ok := r.categories[catKey(nativeCell(table, row, "category"), catType)]
		if !ok {
			continue
		}
		amt, err := parseOptionalAmount(nativeCell(table, row, "amount"))
		if err != nil {
			return err
		}
		var subID *string
		if sn := nativeCell(table, row, "subcategory"); sn != "" {
			if id, ok := r.subcategories[subKey(nativeCell(table, row, "category"), sn)]; ok {
				subID = &id
			}
		}
		if err := q.InsertRecurringOperation(ctx, sqlcdb.InsertRecurringOperationParams{
			ID: uuid.NewString(), UserID: r.userID, Type: catType, Amount: amt,
			Description: strPtr(nativeCell(table, row, "description")), AccountID: acct, CategoryID: catID, SubcategoryID: subID,
			Period: nativeCell(table, row, "period"), Weekday: parseOptionalInt(nativeCell(table, row, "weekday")),
			DayOfMonth: parseOptionalInt(nativeCell(table, row, "day_of_month")),
			StartDate:  nativeCell(table, row, "start_date"), TimeLocal: orDefault(nativeCell(table, row, "time_local"), "08:00"),
			NextRunAt: nativeCell(table, row, "next_run_at"), LastRunAt: strPtr(nativeCell(table, row, "last_run_at")),
			Active: boolToInt(parseCSVBool(nativeCell(table, row, "active"))), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func importNativeBudgets(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionBudgets]
	if !ok {
		return nil
	}
	for _, row := range table.Rows {
		amt, err := parseOptionalAmount(nativeCell(table, row, "amount"))
		if err != nil {
			return err
		}
		in := budget.Input{
			Name: nativeCell(table, row, "name"), Scope: nativeCell(table, row, "scope"),
			Amount: amt, Month: nativeCell(table, row, "month"),
			CopyForward:    parseCSVBool(nativeCell(table, row, "copy_forward")),
			AlertAtPercent: 90, IsActive: parseCSVBool(nativeCell(table, row, "is_active")),
		}
		if v := parseOptionalInt(nativeCell(table, row, "alert_at_percent")); v != nil {
			in.AlertAtPercent = *v
		}
		if cn := nativeCell(table, row, "category"); cn != "" {
			if id, ok := r.categories[catKey(cn, "expense")]; ok {
				in.CategoryID = &id
			} else if id, ok := r.categories[catKey(cn, "income")]; ok {
				in.CategoryID = &id
			}
		}
		if sn := nativeCell(table, row, "subcategory"); sn != "" {
			if id, ok := r.subcategories[subKey(nativeCell(table, row, "category"), sn)]; ok {
				in.SubcategoryID = &id
			}
		}
		if an := nativeCell(table, row, "account"); an != "" {
			if id, err := accountIDByName(r, an); err == nil {
				in.AccountID = &id
			}
		}
		if _, err := budget.Create(ctx, db, r.userID, in); err != nil {
			continue
		}
	}
	return nil
}

func importNativeTemplates(ctx context.Context, db *sql.DB, r *resolver, nf *NativeFile) error {
	table, ok := nf.Sections[sectionTemplates]
	if !ok {
		return nil
	}
	for _, row := range table.Rows {
		in := transactiontemplate.Input{
			Name: nativeCell(table, row, "name"), Type: nativeCell(table, row, "type"),
			Description: strPtr(nativeCell(table, row, "description")),
			Icon:        strPtr(nativeCell(table, row, "icon")),
		}
		if an := nativeCell(table, row, "account"); an != "" {
			if id, err := accountIDByName(r, an); err == nil {
				in.AccountID = &id
			}
		}
		if an := nativeCell(table, row, "to_account"); an != "" {
			if id, err := accountIDByName(r, an); err == nil {
				in.ToAccountID = &id
			}
		}
		catType := in.Type
		if catType == "transfer" {
			catType = "expense"
		}
		if cn := nativeCell(table, row, "category"); cn != "" {
			if id, ok := r.categories[catKey(cn, catType)]; ok {
				in.CategoryID = &id
			}
		}
		if sn := nativeCell(table, row, "subcategory"); sn != "" {
			if id, ok := r.subcategories[subKey(nativeCell(table, row, "category"), sn)]; ok {
				in.SubcategoryID = &id
			}
		}
		if am := strings.TrimSpace(nativeCell(table, row, "amount")); am != "" {
			if v, err := ParseCubuxAmount(am); err == nil {
				in.Amount = &v
			}
		}
		if mn := nativeCell(table, row, "merchant"); mn != "" {
			if m, err := merchant.Create(ctx, db, r.userID, mn, "default"); err == nil {
				in.MerchantID = &m.ID
			} else {
				list, _ := merchant.List(ctx, db, r.userID)
				for i := range list {
					if strings.EqualFold(list[i].Name, mn) {
						id := list[i].ID
						in.MerchantID = &id
						break
					}
				}
			}
		}
		for _, name := range parseTagList(nativeCell(table, row, "tags")) {
			tg, err := tag.Create(ctx, db, r.userID, name)
			if err != nil {
				list, _ := tag.List(ctx, db, r.userID, "")
				for i := range list {
					if strings.EqualFold(list[i].Name, name) {
						in.TagIDs = append(in.TagIDs, list[i].ID)
						break
					}
				}
				continue
			}
			in.TagIDs = append(in.TagIDs, tg.ID)
		}
		if _, err := transactiontemplate.Create(ctx, db, r.userID, in); err != nil {
			continue
		}
	}
	return nil
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func orDefault(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func parseOptionalIntAmount(s string) *int64 {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v, err := parseOptionalAmount(s)
	if err != nil {
		return nil
	}
	return &v
}
