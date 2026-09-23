package importexport

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func TestNativeCSVParseSections(t *testing.T) {
	data, err := writeNativeCSV([]nativeSection{
		{sectionAccounts, nativeAccountHeaders, [][]string{{
			"id-1", "Кошелёк", "cash", "", "100.00", "", "1", "active", "", "0", "", "", "",
		}}},
		{sectionTags, nativeTagHeaders, [][]string{{"дом"}}},
		{sectionTx, BuhgalterHeaders, [][]string{{
			"Расходы", "01.01.2025", "10.00", "RUB", "Кошелёк", "", "", "", "Транспорт", "Автобус", "", "", "User",
			"09:15:00", "Пятёрочка", "дом", "", "tx-1", "",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !IsNativeCSV(data) {
		t.Fatal("expected native magic")
	}
	nf, err := ParseNativeCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := nf.Sections[sectionAccounts]; !ok {
		t.Fatal("missing accounts")
	}
	if _, ok := nf.Sections[sectionTags]; !ok {
		t.Fatal("missing tags")
	}
	tx := nf.Sections[sectionTx]
	if len(tx.Rows) != 1 {
		t.Fatalf("tx rows %d", len(tx.Rows))
	}
}

func TestParseImportFileIgnoresCatalogSections(t *testing.T) {
	data, err := writeNativeCSV([]nativeSection{
		{sectionAccounts, nativeAccountHeaders, [][]string{{
			"id-1", "Кошелёк", "cash", "", "100.00", "", "1", "active", "", "0", "", "", "",
		}}},
		{sectionCategories, nativeCategoryHeaders, [][]string{{
			"Транспорт", "expense", "transport", "1", "0", "0",
		}}},
		{sectionTx, BuhgalterHeaders, [][]string{{
			"Расходы", "01.01.2025", "10.00", "RUB", "Кошелёк", "", "", "", "Транспорт", "Автобус", "", "", "User",
			"09:15:00", "", "", "", "tx-1", "",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	prefixed := append([]byte("sep=,\n"), StripUTF8BOM(data)...)
	for i, blob := range [][]byte{data, prefixed} {
		table, nf, err := ParseImportFile("export.csv", blob)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if nf == nil || len(table.Rows) != 1 {
			t.Fatalf("case %d: native=%v tx rows=%d", i, nf != nil, len(table.Rows))
		}
		mapped, errs := MapTable(table, ImportOptions{Preset: "cubux"})
		if len(errs) != 0 {
			t.Fatalf("case %d map errors: %+v", i, errs)
		}
		if len(mapped) != 1 {
			t.Fatalf("case %d mapped %d", i, len(mapped))
		}
	}
}

func TestNativeFallbackWhenMagicParsedAsHeader(t *testing.T) {
	data, err := writeNativeCSV([]nativeSection{
		{sectionAccounts, nativeAccountHeaders, [][]string{{
			"id-1", "Кошелёк", "cash", "", "100.00", "", "1", "active", "", "0", "", "", "",
		}}},
		{sectionTx, BuhgalterHeaders, [][]string{{
			"Расходы", "18.09.2026", "10.00", "RUB", "Кошелёк", "", "", "", "Транспорт", "Автобус", "", "", "User",
			"", "", "", "", "", "",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	flat, err := ParseCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	if !looksLikeNativeTable(flat) {
		t.Fatalf("headers %q should look native", flat.Headers)
	}
	nf, err := nativeFileFromFlatTable(flat)
	if err != nil {
		t.Fatal(err)
	}
	tx := nf.Sections[sectionTx]
	if len(tx.Rows) != 1 {
		t.Fatalf("tx rows %d, headers=%q", len(tx.Rows), tx.Headers)
	}
	mapped, errs := MapTable(tx, ImportOptions{Preset: "buhgalter"})
	if len(errs) != 0 || len(mapped) != 1 {
		t.Fatalf("mapped %d errs %+v", len(mapped), errs)
	}
}

func TestPreviewNativeFileDoesNotReportMissingDates(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	data, err := writeNativeCSV([]nativeSection{
		{sectionAccounts, nativeAccountHeaders, [][]string{{
			"id-1", "Наличные", "cash", "", "288.00", "", "0", "active", "", "0", "", "", "",
		}}},
		{sectionTx, BuhgalterHeaders, [][]string{{
			"Расходы", "01.01.2025", "50.00", "RUB", "Наличные", "", "", "", "Транспорт", "Автобус", "", "", "User",
			"09:15:00", "", "", "", "tx-1", "",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Preview(ctx, sqlDB, userID, "buhgalter_export_2026.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRows != 1 {
		t.Fatalf("total_rows %d, want transactions only", report.TotalRows)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors: %+v", report.Errors)
	}
	if report.ValidRows != 1 {
		t.Fatalf("valid %d", report.ValidRows)
	}
}

func TestNativeExportImportPreservesLedger(t *testing.T) {
	ctx, sqlDB, srcUser := seedImportUser(t)
	if err := bank.SeedIfEmpty(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	banks, err := bank.ListAll(ctx, sqlDB)
	if err != nil || len(banks) == 0 {
		t.Fatalf("banks: %v", err)
	}
	bankID := banks[0].ID
	limit := int64(13_000_000)

	cash, err := account.Create(ctx, sqlDB, srcUser, account.CreateInput{
		Name: "Кошелёк тест", Type: "cash", InitialBalance: 12_345,
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := account.Create(ctx, sqlDB, srcUser, account.CreateInput{
		Name: "Карта тест", Type: "credit_card", BankID: &bankID,
		InitialBalance: 6_500_000, CreditLimit: &limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := merchant.Create(ctx, sqlDB, srcUser, "Лишний магазин", "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := tag.Create(ctx, sqlDB, srcUser, "архив"); err != nil {
		t.Fatal(err)
	}
	srcProfile, err := auth.LoadUser(ctx, sqlDB, srcUser)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.UpdateUserProfile(ctx, sqlDB, srcUser, srcProfile.DisplayName, srcProfile.Language, srcProfile.Currency, "Asia/Irkutsk", srcProfile.Theme); err != nil {
		t.Fatal(err)
	}
	txDate := time.Date(2025, 3, 1, 9, 15, 0, 0, time.UTC)
	if _, err := transaction.Create(ctx, sqlDB, srcUser, transaction.CreateInput{
		AccountID: cash.ID, Type: "expense", Amount: 1234,
		MerchantName: strPtr("Пятёрочка"), TagNames: []string{"дом", "еда"},
		TransactionDate: txDate,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.CreateTransfer(ctx, sqlDB, srcUser, transaction.TransferInput{
		FromAccountID: card.ID, ToAccountID: cash.ID, Amount: 50000,
		Commission: 1500, TransactionDate: txDate.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	srcCash, srcCard := mustAccountByName(t, ctx, sqlDB, srcUser, "Кошелёк тест"), mustAccountByName(t, ctx, sqlDB, srcUser, "Карта тест")
	srcMerchants, err := merchant.List(ctx, sqlDB, srcUser)
	if err != nil {
		t.Fatal(err)
	}
	srcTags, err := tag.List(ctx, sqlDB, srcUser, "")
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := ExportCSV(ctx, sqlDB, srcUser, "Import", ExportFilters{Format: "buhgalter"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "#SECTION,accounts") || !strings.Contains(string(out), "#SECTION,tags") {
		t.Fatalf("export missing catalogs: %s", string(out)[:min(400, len(out))])
	}

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	dstUser, err := auth.CreateUser(ctx, sqlDB, "importuser2", hash, "Import2", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Import(ctx, sqlDB, dstUser, "native.csv", out, ImportOptions{
		Preset: "buhgalter", Deduplicate: true, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) != 0 {
		t.Fatalf("import errors: %+v", rep.Errors)
	}
	if rep.ValidRows != 2 || rep.TransferRows != 1 || rep.ListRows != 2 {
		t.Fatalf("import counts valid=%d transfers=%d list=%d", rep.ValidRows, rep.TransferRows, rep.ListRows)
	}

	dstCash, dstCard := mustAccountByName(t, ctx, sqlDB, dstUser, "Кошелёк тест"), mustAccountByName(t, ctx, sqlDB, dstUser, "Карта тест")
	if dstCash.InitialBalance != srcCash.InitialBalance || dstCash.Balance != srcCash.Balance {
		t.Fatalf("cash initial/balance src=%d/%d dst=%d/%d", srcCash.InitialBalance, srcCash.Balance, dstCash.InitialBalance, dstCash.Balance)
	}
	if dstCard.Type != "credit_card" {
		t.Fatalf("card type %s", dstCard.Type)
	}
	if dstCard.InitialBalance != srcCard.InitialBalance || dstCard.Balance != srcCard.Balance {
		t.Fatalf("card initial/balance src=%d/%d dst=%d/%d", srcCard.InitialBalance, srcCard.Balance, dstCard.InitialBalance, dstCard.Balance)
	}
	if dstCard.CreditLimit == nil || srcCard.CreditLimit == nil || *dstCard.CreditLimit != *srcCard.CreditLimit {
		t.Fatalf("card limit src=%v dst=%v", srcCard.CreditLimit, dstCard.CreditLimit)
	}

	dstMerchants, err := merchant.List(ctx, sqlDB, dstUser)
	if err != nil {
		t.Fatal(err)
	}
	if len(dstMerchants) != len(srcMerchants) {
		t.Fatalf("merchants src=%d dst=%d", len(srcMerchants), len(dstMerchants))
	}
	dstTags, err := tag.List(ctx, sqlDB, dstUser, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(dstTags) != len(srcTags) {
		t.Fatalf("tags src=%d dst=%d", len(srcTags), len(dstTags))
	}

	srcTx := countUserTransactions(t, ctx, sqlDB, srcUser)
	dstTx := countUserTransactions(t, ctx, sqlDB, dstUser)
	if dstTx != srcTx {
		t.Fatalf("transactions src=%d dst=%d", srcTx, dstTx)
	}
	dstProfile, err := auth.LoadUser(ctx, sqlDB, dstUser)
	if err != nil {
		t.Fatal(err)
	}
	if dstProfile.Timezone != "Asia/Irkutsk" {
		t.Fatalf("timezone %s, want Asia/Irkutsk", dstProfile.Timezone)
	}
}

func mustAccountByName(t *testing.T, ctx context.Context, db *sql.DB, userID, name string) account.Account {
	t.Helper()
	list, err := account.ListByUser(ctx, db, userID, "active")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("account %q not found", name)
	return account.Account{}
}

func TestExportNativeIgnoresDateFilters(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	cash, err := account.Create(ctx, sqlDB, userID, account.CreateInput{Name: "Касса фильтр", Type: "cash"})
	if err != nil {
		t.Fatal(err)
	}
	past := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	mid := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	future := time.Now().UTC().AddDate(0, 0, 40)
	for _, d := range []time.Time{past, mid, future} {
		if _, err := transaction.Create(ctx, sqlDB, userID, transaction.CreateInput{
			AccountID: cash.ID, Type: "expense", Amount: 1000, TransactionDate: d,
		}); err != nil {
			t.Fatal(err)
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	out, _, err := ExportCSV(ctx, sqlDB, userID, "User", ExportFilters{
		Format: "buhgalter", From: "2025-01-01", To: today,
	})
	if err != nil {
		t.Fatal(err)
	}
	nf, err := ParseNativeCSV(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(nf.Sections[sectionTx].Rows); got != 3 {
		t.Fatalf("native export txs=%d want 3 (past, in-range, future)", got)
	}
	text := string(out)
	if !strings.Contains(text, "15.06.2020") || !strings.Contains(text, "15.06.2025") {
		t.Fatalf("native export missing out-of-range dates: %s", text[len(text)-min(800, len(text)):])
	}

	cubux, _, err := ExportCSV(ctx, sqlDB, userID, "User", ExportFilters{From: "2025-01-01", To: today})
	if err != nil {
		t.Fatal(err)
	}
	cubuxText := string(cubux)
	if strings.Contains(cubuxText, "15.06.2020") {
		t.Fatal("cubux export should honor from=2025-01-01")
	}
	if !strings.Contains(cubuxText, "15.06.2025") {
		t.Fatal("cubux export missing in-range row")
	}
}

func countUserTransactions(t *testing.T, ctx context.Context, db *sql.DB, userID string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions WHERE user_id = ?`, userID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
