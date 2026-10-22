package importexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/recurring"
	"github.com/kai-zer-ru/buhgalter/internal/subscription"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
	"github.com/kai-zer-ru/buhgalter/internal/transactiontemplate"
)

func exportNative(ctx context.Context, db *sql.DB, userID, displayName string, f ExportFilters) ([]byte, string, error) {
	q := sqlcdb.New(db)
	srcUser, err := auth.LoadUser(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	profileRows := [][]string{{srcUser.Timezone, srcUser.Currency, srcUser.Language}}

	accounts, err := q.ListAccountsForExport(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	acctRows := make([][]string, 0, len(accounts))
	for _, a := range accounts {
		acctRows = append(acctRows, []string{
			a.ID, a.Name, a.Type, csvStringPtr(a.BankName), FormatCubuxAmount(a.InitialBalance),
			csvAmountPtr(a.CreditLimit), csvBool(a.IsPrimary != 0), a.Status, csvStringPtr(a.PaymentAccountName),
			csvBool(a.AutoTopupEnabled != 0), csvAmountPtr(a.AutoTopupThreshold), csvAmountPtr(a.AutoTopupTarget),
			csvStringPtr(a.AutoTopupSourceName),
		})
	}

	cats, err := category.ListByUser(ctx, db, userID, "")
	if err != nil {
		return nil, "", err
	}
	catByID := make(map[string]category.Category, len(cats))
	catRows := make([][]string, 0, len(cats))
	for _, c := range cats {
		catByID[c.ID] = c
		catRows = append(catRows, []string{c.Name, c.Type, c.Icon, strconv.Itoa(c.SortOrder), csvBool(c.IsPrimary), csvBool(c.IsSystem)})
	}
	subs, err := category.ListSubcategoriesByUser(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	subRows := make([][]string, 0, len(subs))
	for _, s := range subs {
		parent := catByID[s.CategoryID]
		subRows = append(subRows, []string{parent.Name, parent.Type, s.Name, s.Icon, strconv.Itoa(s.SortOrder)})
	}

	merchants, err := merchant.List(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	merchRows := make([][]string, 0, len(merchants))
	for _, m := range merchants {
		merchRows = append(merchRows, []string{m.Name, m.Icon})
	}
	tags, err := tag.List(ctx, db, userID, "")
	if err != nil {
		return nil, "", err
	}
	tagRows := make([][]string, 0, len(tags))
	for _, tg := range tags {
		tagRows = append(tagRows, []string{tg.Name})
	}

	subscrip, err := subscription.List(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	subNameByID := make(map[string]string, len(subscrip))
	for _, s := range subscrip {
		subNameByID[s.ID] = s.Name
	}

	txRows, err := collectExportTransactionRows(ctx, db, userID, displayName, f, subNameByID)
	if err != nil {
		return nil, "", err
	}

	credits, err := credit.List(ctx, db, userID, "")
	if err != nil {
		return nil, "", err
	}
	creditRows := make([][]string, 0, len(credits))
	payRows := make([][]string, 0)
	for _, c := range credits {
		creditRows = append(creditRows, []string{
			c.ID, csvStringPtr(c.Name), c.CreditKind, FormatCubuxAmount(c.PrincipalAmount), csvAmountPtr(c.PropertyPrice),
			FormatCubuxAmount(c.DownPayment), csvBool(c.DownPaymentAffectsBalance), csvStringPtr(c.DownPaymentTransactionID),
			csvBool(c.PrincipalAffectsBalance), csvStringPtr(c.PrincipalTransactionID), c.IssueDate, strconv.Itoa(c.TermMonths),
			strconv.FormatFloat(c.InterestRate, 'f', -1, 64), c.PaymentInterval, FormatCubuxAmount(c.PaidAmount),
			FormatCubuxAmount(c.MonthlyPayment), c.DebitAccountName, csvStringPtr(c.DebitTimeLocal), csvStringPtr(c.BankName),
			csvBool(c.BankIDLocked), csvBool(c.AddedRetroactively), c.RecordedAt, c.Status, csvStringPtr(c.ClosedAt),
		})
		payments, err := q.ListCreditPayments(ctx, c.ID)
		if err != nil {
			return nil, "", err
		}
		for _, p := range payments {
			payRows = append(payRows, []string{
				p.ID, p.CreditID, csvStringPtr(p.TransactionID), FormatCubuxAmount(p.Amount), p.PaymentDate,
				p.Kind, csvBool(p.IsApplied != 0), csvBool(p.ExcludeFromStats != 0),
			})
		}
	}

	debtors, err := debt.ListDebtors(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	debtorRows := make([][]string, 0, len(debtors))
	for _, d := range debtors {
		debtorRows = append(debtorRows, []string{d.ID, d.Name})
	}
	debts, err := debt.List(ctx, db, userID, "")
	if err != nil {
		return nil, "", err
	}
	debtRows := make([][]string, 0, len(debts))
	for _, d := range debts {
		debtRows = append(debtRows, []string{
			d.ID, d.DebtorID, d.DebtorName, d.Direction, FormatCubuxAmount(d.Amount), csvBool(d.AffectsBalance),
			d.DebtDate, d.DueDate, csvStringPtr(d.Description), csvStringPtr(d.TransactionID),
			csvBool(d.IsSettled), csvStringPtr(d.SettledAt),
		})
	}
	links, err := q.ListDebtLinksByUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	linkRows := make([][]string, 0, len(links))
	for _, l := range links {
		linkRows = append(linkRows, []string{l.DebtID, l.TransactionID, l.Role})
	}

	subscrRows := make([][]string, 0, len(subscrip))
	for _, s := range subscrip {
		upcoming, _ := json.Marshal(s.UpcomingRunAts)
		subscrRows = append(subscrRows, []string{
			s.ID, s.Name, csvStringPtr(s.Description), csvStringPtr(s.Icon), csvStringPtr(s.WebsiteURL),
			FormatCubuxAmount(s.Amount), s.AccountName, csvStringPtr(s.SubcategoryName), s.Period,
			csvIntPtr(s.Weekday), csvIntPtr(s.DayOfMonth), s.StartDate, s.TimeLocal, s.NextRunAt,
			string(upcoming), csvStringPtr(s.LastRunAt), csvBool(s.Active),
		})
	}

	recurringOps, err := recurring.List(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	recRows := make([][]string, 0, len(recurringOps))
	for _, r := range recurringOps {
		recRows = append(recRows, []string{
			r.ID, r.Type, FormatCubuxAmount(r.Amount), csvStringPtr(r.Description), r.AccountName, r.CategoryName,
			csvStringPtr(r.SubcategoryName), r.Period, csvIntPtr(r.Weekday), csvIntPtr(r.DayOfMonth),
			r.StartDate, r.TimeLocal, r.NextRunAt, csvStringPtr(r.LastRunAt), csvBool(r.Active),
		})
	}

	budgets, err := q.ListBudgetsByUser(ctx, sqlcdb.ListBudgetsByUserParams{UserID: userID, Column2: "", Month: ""})
	if err != nil {
		return nil, "", err
	}
	budgetRows := make([][]string, 0, len(budgets))
	for _, b := range budgets {
		budgetRows = append(budgetRows, []string{
			b.ID, b.Name, b.Scope, csvStringPtr(b.CategoryName), csvStringPtr(b.SubcategoryName),
			FormatCubuxAmount(b.Amount), csvStringPtr(b.AccountName), b.Month,
			csvBool(b.CopyForward != 0), csvBool(b.Rollover != 0), csvInt(b.AlertAtPercent), csvBool(b.IsActive != 0),
		})
	}

	templates, err := transactiontemplate.List(ctx, db, userID)
	if err != nil {
		return nil, "", err
	}
	tmplRows := make([][]string, 0, len(templates))
	acctName := func(id *string) string {
		if id == nil {
			return ""
		}
		for _, a := range accounts {
			if a.ID == *id {
				return a.Name
			}
		}
		return ""
	}
	catName := func(id *string) string {
		if id == nil {
			return ""
		}
		if c, ok := catByID[*id]; ok {
			return c.Name
		}
		return ""
	}
	subName := func(id *string) string {
		if id == nil {
			return ""
		}
		for _, s := range subs {
			if s.ID == *id {
				return s.Name
			}
		}
		return ""
	}
	merchName := func(id *string) string {
		if id == nil {
			return ""
		}
		for _, m := range merchants {
			if m.ID == *id {
				return m.Name
			}
		}
		return ""
	}
	for _, tmpl := range templates {
		tagNames := make([]string, 0, len(tmpl.Tags))
		for _, tg := range tmpl.Tags {
			tagNames = append(tagNames, tg.Name)
		}
		tmplRows = append(tmplRows, []string{
			tmpl.Name, tmpl.Type, acctName(tmpl.AccountID), acctName(tmpl.ToAccountID),
			catName(tmpl.CategoryID), subName(tmpl.SubcategoryID), csvAmountPtr(tmpl.Amount),
			csvStringPtr(tmpl.Description), merchName(tmpl.MerchantID), csvStringPtr(tmpl.Icon),
			strings.Join(tagNames, ", "),
		})
	}

	data, err := writeNativeCSV([]nativeSection{
		{sectionProfile, nativeProfileHeaders, profileRows},
		{sectionAccounts, nativeAccountHeaders, acctRows},
		{sectionCategories, nativeCategoryHeaders, catRows},
		{sectionSubcats, nativeSubcatHeaders, subRows},
		{sectionMerchants, nativeMerchantHeaders, merchRows},
		{sectionTags, nativeTagHeaders, tagRows},
		{sectionTx, BuhgalterHeaders, txRows},
		{sectionCredits, nativeCreditHeaders, creditRows},
		{sectionCreditPays, nativeCreditPayHeaders, payRows},
		{sectionDebtors, nativeDebtorHeaders, debtorRows},
		{sectionDebts, nativeDebtHeaders, debtRows},
		{sectionDebtLinks, nativeDebtLinkHeaders, linkRows},
		{sectionSubs, nativeSubHeaders, subscrRows},
		{sectionRecurring, nativeRecurringHeaders, recRows},
		{sectionBudgets, nativeBudgetHeaders, budgetRows},
		{sectionTemplates, nativeTemplateHeaders, tmplRows},
	})
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("buhgalter_export_%s.csv", time.Now().Format("2006"))
	return data, filename, nil
}

func csvIntPtr(v *int64) string {
	if v == nil {
		return ""
	}
	return csvInt(*v)
}

func collectExportTransactionRows(ctx context.Context, db *sql.DB, userID, displayName string, f ExportFilters, subNameByID map[string]string) ([][]string, error) {
	from := normalizeExportDate(f.From, false)
	to := normalizeExportDate(f.To, true)
	var all []transaction.Transaction
	page := 1
	for {
		res, err := transaction.List(ctx, db, userID, transaction.ListFilters{
			AccountID:  f.AccountID,
			CategoryID: f.CategoryID,
			From:       from,
			To:         to,
			Sort:       "date_asc",
			Page:       page,
			Limit:      200,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res.Data...)
		if int64(page*200) >= res.Meta.Total {
			break
		}
		page++
	}
	seenTransfers := make(map[string]bool)
	rows := make([][]string, 0, len(all))
	for _, tx := range all {
		if tx.Type == "transfer" {
			if tx.TransferGroupID == nil || !tx.TransferIsOut {
				continue
			}
			if seenTransfers[*tx.TransferGroupID] {
				continue
			}
			seenTransfers[*tx.TransferGroupID] = true
		}
		line := txToExportLine(tx, displayName, "buhgalter", subNameByID)
		if line == nil {
			continue
		}
		rows = append(rows, line)
	}
	return rows, nil
}
