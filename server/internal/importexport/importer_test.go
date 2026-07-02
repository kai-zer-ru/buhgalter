package importexport

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/bank"
)

func sampleCSVRows() []byte {
	return []byte(`Тип,Дата,Сумма списания,Валюта списания,Счет списания,Сумма пополнения,Валюта назначения,Счет пополнения,Категория,Subcategory,Описание,Проект,Пользователь
Расходы,01.01.2025,50.00_-₽,RUB,Наличные,,,,Транспорт,Автобус,,,User
Расходы,02.01.2025,100.00_-₽,RUB,Яндекс,,,,Связь,Подписки,,,User
Доходы,03.01.2025,,,,200.00_-₽,RUB,Яндекс,Прочие доходы,Авито,,,User
Перевод,04.01.2025,300.00_-₽,RUB,Яндекс,300.00_-₽,RUB,Кредитка,Перевод,,,,User
`)
}

func seedImportUser(t *testing.T) (context.Context, *sql.DB, string) {
	ctx, handle, userID := seedImportHandle(t)
	return ctx, handle.DB(), userID
}

func TestDedupSet(t *testing.T) {
	h := DedupHash("2025-01-01", 5000, "Cash", "Food", "expense")
	s := NewDedupSet([]string{h})
	if !s.Has(h) {
		t.Fatal("expected hash in set")
	}
	if s.Has("other") {
		t.Fatal("unexpected hash")
	}
	s.Add("new")
	if !s.Has("new") {
		t.Fatal("expected added hash")
	}
}

func TestFormatCubuxAmountRoundTrip(t *testing.T) {
	got := FormatCubuxAmount(12345)
	if got == "" {
		t.Fatal("expected formatted amount")
	}
	parsed, err := ParseCubuxAmount(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != 12345 {
		t.Fatalf("round trip: %d", parsed)
	}
}

func TestPreviewAndImportCubux(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	data := sampleCSVRows()

	report, err := Preview(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRows != 4 || report.ValidRows != 4 {
		t.Fatalf("preview: %+v", report)
	}
	if len(report.AccountsToCreate) < 3 {
		t.Fatalf("accounts to create: %v", report.AccountsToCreate)
	}

	committed, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.CreatedTransactions != 4 {
		t.Fatalf("created %d", committed.CreatedTransactions)
	}

	// second import should skip duplicates
	again, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.SkippedDuplicates != 4 {
		t.Fatalf("skipped %d", again.SkippedDuplicates)
	}
}

func TestImportIdempotencyKey(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	data := sampleCSVRows()
	key := "idem-key-1"

	first, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.CreatedTransactions != 4 {
		t.Fatalf("created %d", first.CreatedTransactions)
	}

	second, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.CreatedTransactions != first.CreatedTransactions {
		t.Fatalf("cached %d vs first %d", second.CreatedTransactions, first.CreatedTransactions)
	}
}

func TestExportCSV(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	data := sampleCSVRows()

	_, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, filename, err := ExportCSV(ctx, sqlDB, userID, "User", ExportFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected csv bytes")
	}
	if filename == "" {
		t.Fatal("expected filename")
	}
	if string(out[:3]) != "\xef\xbb\xbf" {
		t.Fatal("expected UTF-8 BOM")
	}
}

func TestNormalizeExportDate(t *testing.T) {
	if got := normalizeExportDate("2025-01-15", false); got != "2025-01-15 00:00:00" {
		t.Fatalf("from: %q", got)
	}
	if got := normalizeExportDate("2025-01-15", true); got != "2025-01-15 23:59:59" {
		t.Fatalf("to: %q", got)
	}
}

func TestFormatExportDate(t *testing.T) {
	if got := formatExportDate("2025-06-24 12:00:00"); got != "24.06.2025" {
		t.Fatalf("got %q", got)
	}
}

func TestImportWithExplicitAccountMap(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	data := sampleCSVRows()

	accountMap := map[string]AccountMapEntry{
		"Наличные": {Mode: "create", AccountType: "cash"},
		"Яндекс":   {Mode: "create", AccountType: "cash"},
		"Кредитка": {Mode: "create", AccountType: "cash"},
	}
	report, err := Preview(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, AccountMap: accountMap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ValidRows != 4 {
		t.Fatalf("valid %d", report.ValidRows)
	}

	committed, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true, AccountMap: accountMap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.CreatedTransactions != 4 {
		t.Fatalf("created %d", committed.CreatedTransactions)
	}
}

func TestImportCreateCreditCardAccount(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	if err := bank.SeedIfEmpty(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	banks, err := bank.ListAll(ctx, sqlDB)
	if err != nil || len(banks) == 0 {
		t.Fatalf("banks: %v", err)
	}
	bankID := banks[0].ID

	accountMap := map[string]AccountMapEntry{
		"Наличные": {Mode: "create", AccountType: "cash"},
		"Яндекс":   {Mode: "create", AccountType: "cash"},
		"Кредитка": {Mode: "create", AccountType: "credit_card", BankID: bankID, CreditLimit: "65000.00"},
	}
	data := sampleCSVRows()
	committed, err := Import(ctx, sqlDB, userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true, AccountMap: accountMap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.CreatedTransactions != 4 {
		t.Fatalf("created %d", committed.CreatedTransactions)
	}
	var accType string
	var creditLimit sql.NullInt64
	err = sqlDB.QueryRowContext(ctx, `
		SELECT type, credit_limit FROM accounts WHERE user_id = ? AND name = 'Кредитка'`,
		userID).Scan(&accType, &creditLimit)
	if err != nil {
		t.Fatal(err)
	}
	if accType != "credit_card" || !creditLimit.Valid || creditLimit.Int64 != 6_500_000 {
		t.Fatalf("credit card account: type=%s limit=%v", accType, creditLimit)
	}
}
