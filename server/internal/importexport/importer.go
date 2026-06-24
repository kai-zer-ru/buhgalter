package importexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

type resolver struct {
	db                *sql.DB
	userID            string
	accounts          map[string]string // lower(db name) -> id
	acctNames         map[string]string // lower(db name) -> canonical name
	fileAccounts      map[string]string // lower(file name) -> resolved id (during import)
	categories        map[string]string
	catNames          map[string]string
	subcategories     map[string]string
	subByCategoryID   map[string]map[string]category.Subcategory
	createdAccounts   map[string]struct{}
	createdCategories map[string]struct{}
	accountMap        map[string]AccountMapEntry
	categoryMap       map[string]CategoryMapEntry
	subcategoryMap    map[string]SubcategoryMapEntry
	autoSubcategory   bool
	banks             []bank.Bank
	dryRun            bool
}

func newResolver(
	ctx context.Context,
	db *sql.DB,
	userID string,
	accountMap map[string]AccountMapEntry,
	categoryMap map[string]CategoryMapEntry,
	subcategoryMap map[string]SubcategoryMapEntry,
	autoSubcategory bool,
	dryRun bool,
) (*resolver, error) {
	r := &resolver{
		db:                db,
		userID:            userID,
		accounts:          make(map[string]string),
		acctNames:         make(map[string]string),
		fileAccounts:      make(map[string]string),
		categories:        make(map[string]string),
		catNames:          make(map[string]string),
		subcategories:     make(map[string]string),
		subByCategoryID:   make(map[string]map[string]category.Subcategory),
		createdAccounts:   make(map[string]struct{}),
		createdCategories: make(map[string]struct{}),
		accountMap:        accountMap,
		categoryMap:       categoryMap,
		subcategoryMap:    subcategoryMap,
		autoSubcategory:   autoSubcategory,
		dryRun:            dryRun,
	}
	accs, err := account.ListByUser(ctx, db, userID, "active")
	if err != nil {
		return nil, err
	}
	for _, a := range accs {
		key := strings.ToLower(a.Name)
		r.accounts[key] = a.ID
		r.acctNames[key] = a.Name
	}
	r.banks, err = bank.ListAll(ctx, db)
	if err != nil {
		return nil, err
	}
	cats, err := category.ListByUser(ctx, db, userID, "")
	if err != nil {
		return nil, err
	}
	for _, c := range cats {
		r.categories[catKey(c.Name, c.Type)] = c.ID
		r.catNames[catKey(c.Name, c.Type)] = c.Name
		if _, ok := r.subByCategoryID[c.ID]; !ok {
			r.subByCategoryID[c.ID] = make(map[string]category.Subcategory)
		}
		for _, sub := range c.Subcategories {
			r.subcategories[subKey(c.Name, sub.Name)] = sub.ID
			r.subByCategoryID[c.ID][strings.ToLower(strings.TrimSpace(sub.Name))] = sub
		}
	}
	return r, nil
}

func catKey(name, catType string) string {
	return strings.ToLower(strings.TrimSpace(name)) + "|" + catType
}

func subKey(catName, subName string) string {
	return strings.ToLower(strings.TrimSpace(catName)) + "|" + strings.ToLower(strings.TrimSpace(subName))
}

func (r *resolver) resolveAccount(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("пустое имя счёта")
	}
	key := strings.ToLower(name)

	if id, ok := r.fileAccounts[key]; ok {
		if id == "" && r.dryRun {
			return "", nil
		}
		if id != "" {
			return id, nil
		}
	}

	if entry, ok := r.accountMap[name]; ok {
		switch entry.Mode {
		case "existing":
			if entry.AccountID == "" {
				return "", fmt.Errorf("не указан account_id для %q", name)
			}
			r.fileAccounts[key] = entry.AccountID
			return entry.AccountID, nil
		case "create", "":
			return r.createAccountByFileName(ctx, name, key, entry)
		default:
			return "", fmt.Errorf("неизвестный mode для счёта %q", name)
		}
	}

	if id, ok := r.accounts[key]; ok {
		r.fileAccounts[key] = id
		return id, nil
	}

	return r.createAccountByFileName(ctx, name, key, AccountMapEntry{})
}

func resolveCreateType(name string, entry AccountMapEntry, banks []bank.Bank) (string, *string) {
	switch entry.AccountType {
	case "cash":
		return "cash", nil
	case "bank":
		if entry.BankID != "" {
			id := entry.BankID
			return "bank", &id
		}
		if id := MatchBank(name, banks); id != nil {
			return "bank", id
		}
		return "bank", nil
	}
	if id := MatchBank(name, banks); id != nil {
		return "bank", id
	}
	return "cash", nil
}

func (r *resolver) createAccountByFileName(ctx context.Context, name, key string, entry AccountMapEntry) (string, error) {
	if r.dryRun {
		r.createdAccounts[name] = struct{}{}
		r.fileAccounts[key] = ""
		return "", nil
	}
	accType, bankID := resolveCreateType(name, entry, r.banks)
	created, err := account.Create(ctx, r.db, r.userID, account.CreateInput{
		Name: name, Type: accType, BankID: bankID, InitialBalance: 0,
	})
	if err != nil {
		return "", err
	}
	r.fileAccounts[key] = created.ID
	r.accounts[key] = created.ID
	r.acctNames[key] = created.Name
	r.createdAccounts[name] = struct{}{}
	return created.ID, nil
}

func (r *resolver) resolveCategory(ctx context.Context, name, catType string) (*string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	lookupKey := categoryMapLookupKey(name, catType)
	if entry, ok := r.categoryMap[lookupKey]; ok {
		switch entry.Mode {
		case "existing":
			if entry.CategoryID == "" {
				return nil, fmt.Errorf("не указан category_id для %q", name)
			}
			return &entry.CategoryID, nil
		case "create", "":
			return r.createCategoryByFileName(ctx, name, catType)
		default:
			return nil, fmt.Errorf("неизвестный mode для категории %q", name)
		}
	}

	key := catKey(name, catType)
	if id, ok := r.categories[key]; ok {
		return &id, nil
	}
	// fuzzy: case-insensitive match among same type
	for k, id := range r.categories {
		parts := strings.SplitN(k, "|", 2)
		if len(parts) == 2 && parts[1] == catType && strings.EqualFold(parts[0], name) {
			return &id, nil
		}
	}
	return r.createCategoryByFileName(ctx, name, catType)
}

func (r *resolver) createCategoryByFileName(ctx context.Context, name, catType string) (*string, error) {
	lookup := catKey(name, catType)
	if id, ok := r.categories[lookup]; ok && id != "" {
		return &id, nil
	}
	if r.dryRun {
		r.createdCategories[name] = struct{}{}
		return nil, nil
	}
	created, err := category.Create(ctx, r.db, r.userID, name, catType, "default", 0)
	if err != nil {
		if errors.Is(err, category.ErrNameTaken) {
			// A category with this name already exists; reuse it instead of failing the import.
			if id, ok := r.categories[lookup]; ok && id != "" {
				return &id, nil
			}
			cats, listErr := category.ListByUser(ctx, r.db, r.userID, catType)
			if listErr == nil {
				for _, c := range cats {
					if strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(name)) {
						r.categories[catKey(c.Name, catType)] = c.ID
						r.catNames[catKey(c.Name, catType)] = c.Name
						return &c.ID, nil
					}
				}
			}
		}
		return nil, err
	}
	r.categories[catKey(created.Name, catType)] = created.ID
	r.catNames[catKey(created.Name, catType)] = created.Name
	r.createdCategories[name] = struct{}{}
	return &created.ID, nil
}

func (r *resolver) resolveSubcategoryInput(
	ctx context.Context,
	catID *string,
	catName, catType, subName string,
) (*string, *string, error) {
	subName = strings.TrimSpace(subName)
	if subName == "" || catID == nil {
		return nil, nil, nil
	}
	if !r.autoSubcategory {
		key := subcategoryMapLookupKey(catName, catType, subName)
		if entry, ok := r.subcategoryMap[key]; ok {
			switch entry.Mode {
			case "existing":
				if entry.SubcategoryID == "" {
					return nil, nil, fmt.Errorf("не указан subcategory_id для %q", subName)
				}
				id := entry.SubcategoryID
				return &id, nil, nil
			case "create", "":
				name := subName
				return nil, &name, nil
			default:
				return nil, nil, fmt.Errorf("неизвестный mode для подкатегории %q", subName)
			}
		}
	}

	if byCategory, ok := r.subByCategoryID[*catID]; ok {
		if sub, ok := byCategory[strings.ToLower(subName)]; ok {
			id := sub.ID
			return &id, nil, nil
		}
	}
	if id, ok := r.subcategories[subKey(catName, subName)]; ok {
		return &id, nil, nil
	}
	name := subName
	return nil, &name, nil
}

func loadExistingDedup(ctx context.Context, db *sql.DB, userID string) ([]string, error) {
	rows, err := sqlcdb.New(db).ListTransactionDedupRows(ctx, userID)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, 0, len(rows))
	for _, row := range rows {
		hashes = append(hashes, DedupHash(row.TxDate, row.Amount, row.AccountName, row.CategoryName, row.Type))
	}
	return hashes, nil
}

func dedupHashForMapped(m MappedRow) (string, string, int64, string, string, error) {
	date := m.Date.Format("2006-01-02")
	switch m.CubuxType {
	case "Расходы":
		return DedupHash(date, m.DebitAmount, m.DebitAccount, m.Category, "expense"),
			date, m.DebitAmount, m.DebitAccount, "expense", nil
	case "Доходы":
		return DedupHash(date, m.CreditAmount, m.CreditAccount, m.Category, "income"),
			date, m.CreditAmount, m.CreditAccount, "income", nil
	case "Перевод":
		return DedupHash(date, m.DebitAmount, m.DebitAccount, "Перевод", "transfer"),
			date, m.DebitAmount, m.DebitAccount, "transfer", nil
	default:
		return "", "", 0, "", "", fmt.Errorf("неизвестный тип")
	}
}

// Preview runs dry-run import analysis.
func Preview(ctx context.Context, db *sql.DB, userID string, filename string, data []byte, opts ImportOptions) (Report, error) {
	table, err := ParseFile(filename, data)
	if err != nil {
		return Report{}, err
	}
	mapped, mapErrs := MapTable(table, opts)
	report, acctSet, _ := PreviewFromMapped(mapped)
	report.Errors = append(report.Errors, mapErrs...)
	report.TotalRows = len(table.Rows)

	res, err := newResolver(
		ctx, db, userID, opts.AccountMap, opts.CategoryMap, opts.SubcategoryMap, opts.AutoSubcategory, true,
	)
	if err != nil {
		return Report{}, err
	}

	existing, err := loadExistingDedup(ctx, db, userID)
	if err != nil {
		return Report{}, err
	}
	dedup := NewDedupSet(existing)

	for _, m := range mapped {
		if hasMapErr(mapErrs, m.RowNum) {
			continue
		}
		hash, _, _, _, _, err := dedupHashForMapped(m)
		if err != nil {
			continue
		}
		if opts.Deduplicate && dedup.Has(hash) {
			report.SkippedDuplicates++
			continue
		}
		dedup.Add(hash)

		if err := res.touchRow(ctx, m); err != nil {
			report.Errors = append(report.Errors, RowError{Row: m.RowNum, Message: err.Error()})
		}
	}

	report.AccountMappings = buildAccountMappings(acctSet, res.accounts, res.acctNames, res.banks)
	report.AccountsToCreate = accountsToCreateFromMap(report.AccountMappings)
	fileCats := collectFileCategories(mapped)
	report.CategoryMappings = buildCategoryMappings(fileCats, res.categories, res.catNames)
	catMap := make(map[string]CategoryMapEntry, len(report.CategoryMappings))
	for _, m := range report.CategoryMappings {
		key := categoryMapLookupKey(m.FileName, m.Type)
		entry := CategoryMapEntry{Mode: m.Mode}
		if m.CategoryID != nil {
			entry.CategoryID = *m.CategoryID
		}
		catMap[key] = entry
	}
	for k, v := range opts.CategoryMap {
		catMap[k] = v
	}
	report.SubcategoryMappings = buildSubcategoryMappings(collectFileSubcategories(mapped), catMap, res)
	report.CategoriesToCreate = categoriesToCreateFromMap(report.CategoryMappings)
	if len(res.createdAccounts) > 0 {
		for n := range res.createdAccounts {
			report.AccountsToCreate = appendUnique(report.AccountsToCreate, n)
		}
	}
	if len(res.createdCategories) > 0 {
		for n := range res.createdCategories {
			report.CategoriesToCreate = appendUnique(report.CategoriesToCreate, n)
		}
	}
	warnNonRUB(mapped)
	return report, nil
}

func (r *resolver) touchRow(ctx context.Context, m MappedRow) error {
	switch m.CubuxType {
	case "Расходы":
		if _, err := r.resolveAccount(ctx, m.DebitAccount); err != nil {
			return err
		}
		catID, err := r.resolveCategory(ctx, m.Category, "expense")
		if err != nil {
			return err
		}
		if catID != nil {
			_, _, _ = r.resolveSubcategoryInput(ctx, catID, m.Category, "expense", m.Subcategory)
		}
	case "Доходы":
		if _, err := r.resolveAccount(ctx, m.CreditAccount); err != nil {
			return err
		}
		catID, err := r.resolveCategory(ctx, m.Category, "income")
		if err != nil {
			return err
		}
		if catID != nil {
			_, _, _ = r.resolveSubcategoryInput(ctx, catID, m.Category, "income", m.Subcategory)
		}
	case "Перевод":
		if _, err := r.resolveAccount(ctx, m.DebitAccount); err != nil {
			return err
		}
		if _, err := r.resolveAccount(ctx, m.CreditAccount); err != nil {
			return err
		}
	}
	return nil
}

type importProgressFn func(Report)

// Import commits rows to the database.
func Import(ctx context.Context, db *sql.DB, userID string, filename string, data []byte, opts ImportOptions) (Report, error) {
	return importWithProgress(ctx, db, userID, filename, data, opts, nil)
}

// ImportWithProgress commits rows and periodically reports intermediate progress.
func ImportWithProgress(
	ctx context.Context,
	db *sql.DB,
	userID string,
	filename string,
	data []byte,
	opts ImportOptions,
	onProgress func(Report),
) (Report, error) {
	return importWithProgress(ctx, db, userID, filename, data, opts, onProgress)
}

func importWithProgress(
	ctx context.Context,
	db *sql.DB,
	userID string,
	filename string,
	data []byte,
	opts ImportOptions,
	onProgress importProgressFn,
) (Report, error) {
	if !opts.Confirm {
		return Report{}, fmt.Errorf("confirm=true required")
	}
	if opts.IdempotencyKey != "" {
		if cached, err := getIdempotency(ctx, db, userID, opts.IdempotencyKey); err == nil && cached != nil {
			return *cached, nil
		}
	}

	table, err := ParseFile(filename, data)
	if err != nil {
		return Report{}, err
	}
	mapped, mapErrs := MapTable(table, opts)
	report, _, _ := PreviewFromMapped(mapped)
	report.Errors = append(report.Errors, mapErrs...)
	report.TotalRows = len(table.Rows)
	report.ProcessedRows = 0
	report.ValidRows = 0
	report.SkippedDuplicates = 0
	report.Preview = nil

	res, err := newResolver(
		ctx, db, userID, opts.AccountMap, opts.CategoryMap, opts.SubcategoryMap, opts.AutoSubcategory, false,
	)
	if err != nil {
		return Report{}, err
	}

	existing, err := loadExistingDedup(ctx, db, userID)
	if err != nil {
		return Report{}, err
	}
	dedup := NewDedupSet(existing)

	lastProgressAt := time.Time{}
	emitProgress := func(force bool) {
		if onProgress == nil {
			return
		}
		if !force && report.ProcessedRows%25 != 0 {
			return
		}
		if !force && !lastProgressAt.IsZero() && time.Since(lastProgressAt) < 750*time.Millisecond {
			return
		}
		onProgress(cloneReportForProgress(report))
		lastProgressAt = time.Now()
	}
	emitProgress(true)

	mapErrRows := make(map[int]string, len(mapErrs))
	for _, e := range mapErrs {
		if _, exists := mapErrRows[e.Row]; exists {
			continue
		}
		mapErrRows[e.Row] = e.Message
	}

	for rowNum, msg := range mapErrRows {
		report.ProcessedRows++
		report.Logs = append(report.Logs, fmt.Sprintf("row %d: mapping error: %s", rowNum, msg))
		emitProgress(false)
	}

	for _, m := range mapped {
		if hasMapErr(mapErrs, m.RowNum) {
			continue
		}
		hash, _, _, _, _, err := dedupHashForMapped(m)
		if err != nil {
			report.Errors = append(report.Errors, RowError{Row: m.RowNum, Message: err.Error()})
			report.Logs = append(report.Logs, fmt.Sprintf("row %d: error: %s", m.RowNum, err.Error()))
			report.ProcessedRows++
			emitProgress(false)
			continue
		}
		if opts.Deduplicate && dedup.Has(hash) {
			report.SkippedDuplicates++
			report.Logs = append(report.Logs, fmt.Sprintf("row %d: skipped duplicate", m.RowNum))
			report.ProcessedRows++
			emitProgress(false)
			continue
		}
		dedup.Add(hash)

		if err := importRowWithRetry(ctx, db, userID, res, m); err != nil {
			report.Errors = append(report.Errors, RowError{Row: m.RowNum, Message: err.Error()})
			report.Logs = append(report.Logs, fmt.Sprintf("row %d: error: %s", m.RowNum, err.Error()))
			report.ProcessedRows++
			emitProgress(false)
			continue
		}
		report.ValidRows++
		report.CreatedTransactions++
		report.ProcessedRows++
		report.Logs = append(report.Logs, fmt.Sprintf("row %d: imported", m.RowNum))
		emitProgress(false)
	}

	for n := range res.createdAccounts {
		report.AccountsToCreate = appendUnique(report.AccountsToCreate, n)
	}
	report.AccountMappings = buildAccountMappings(collectFileAccounts(mapped), res.accounts, res.acctNames, res.banks)
	fileCats := collectFileCategories(mapped)
	report.CategoryMappings = buildCategoryMappings(fileCats, res.categories, res.catNames)
	catMap := make(map[string]CategoryMapEntry, len(report.CategoryMappings))
	for _, m := range report.CategoryMappings {
		key := categoryMapLookupKey(m.FileName, m.Type)
		entry := CategoryMapEntry{Mode: m.Mode}
		if m.CategoryID != nil {
			entry.CategoryID = *m.CategoryID
		}
		catMap[key] = entry
	}
	for k, v := range opts.CategoryMap {
		catMap[k] = v
	}
	report.SubcategoryMappings = buildSubcategoryMappings(collectFileSubcategories(mapped), catMap, res)
	for n := range res.createdCategories {
		report.CategoriesToCreate = appendUnique(report.CategoriesToCreate, n)
	}

	if opts.IdempotencyKey != "" {
		_ = saveIdempotency(ctx, db, userID, opts.IdempotencyKey, report)
	}
	emitProgress(true)
	warnNonRUB(mapped)
	return report, nil
}

func cloneReportForProgress(src Report) Report {
	out := src
	out.Errors = append([]RowError(nil), src.Errors...)
	out.Logs = append([]string(nil), src.Logs...)
	out.Preview = nil
	out.AccountMappings = nil
	out.SubcategoryMappings = nil
	out.CategoryMappings = nil
	out.AccountsToCreate = nil
	out.CategoriesToCreate = nil
	return out
}

func importRowWithRetry(ctx context.Context, db *sql.DB, userID string, res *resolver, m MappedRow) error {
	const maxBusyRetries = 7
	backoff := 40 * time.Millisecond
	for attempt := 0; ; attempt++ {
		err := importRow(ctx, db, userID, res, m)
		if err == nil {
			return nil
		}
		if !isSQLiteBusyError(err) || attempt >= maxBusyRetries {
			return err
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if backoff < 1200*time.Millisecond {
			backoff *= 2
		}
	}
}

func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlite_busy") || strings.Contains(msg, "database is locked")
}

func importRow(ctx context.Context, db *sql.DB, userID string, res *resolver, m MappedRow) error {
	txDate := time.Date(m.Date.Year(), m.Date.Month(), m.Date.Day(), 12, 0, 0, 0, time.UTC)
	desc := strPtr(m.Description)

	switch m.CubuxType {
	case "Расходы":
		accID, err := res.resolveAccount(ctx, m.DebitAccount)
		if err != nil {
			return err
		}
		catID, err := res.resolveCategory(ctx, m.Category, "expense")
		if err != nil {
			return err
		}
		subID, subName, err := res.resolveSubcategoryInput(ctx, catID, m.Category, "expense", m.Subcategory)
		if err != nil {
			return err
		}
		created, err := transaction.Create(ctx, db, userID, transaction.CreateInput{
			AccountID: accID, Type: "expense", Amount: m.DebitAmount,
			Description: desc, CategoryID: catID, SubcategoryID: subID, SubcategoryName: subName,
			TransactionDate: txDate,
		})
		if err != nil {
			return err
		}
		res.rememberCreatedSubcategory(catID, m.Category, subName, created.SubcategoryID)
		return nil
	case "Доходы":
		accID, err := res.resolveAccount(ctx, m.CreditAccount)
		if err != nil {
			return err
		}
		catID, err := res.resolveCategory(ctx, m.Category, "income")
		if err != nil {
			return err
		}
		subID, subName, err := res.resolveSubcategoryInput(ctx, catID, m.Category, "income", m.Subcategory)
		if err != nil {
			return err
		}
		created, err := transaction.Create(ctx, db, userID, transaction.CreateInput{
			AccountID: accID, Type: "income", Amount: m.CreditAmount,
			Description: desc, CategoryID: catID, SubcategoryID: subID, SubcategoryName: subName,
			TransactionDate: txDate,
		})
		if err != nil {
			return err
		}
		res.rememberCreatedSubcategory(catID, m.Category, subName, created.SubcategoryID)
		return nil
	case "Перевод":
		fromID, err := res.resolveAccount(ctx, m.DebitAccount)
		if err != nil {
			return err
		}
		toID, err := res.resolveAccount(ctx, m.CreditAccount)
		if err != nil {
			return err
		}
		_, err = transaction.CreateTransfer(ctx, db, userID, transaction.TransferInput{
			FromAccountID: fromID, ToAccountID: toID, Amount: m.DebitAmount,
			Description: desc, TransactionDate: txDate,
		})
		return err
	default:
		return fmt.Errorf("неизвестный тип")
	}
}

func (r *resolver) rememberCreatedSubcategory(catID *string, catName string, subName *string, subID *string) {
	if catID == nil || subName == nil || subID == nil {
		return
	}
	name := strings.ToLower(strings.TrimSpace(*subName))
	if name == "" {
		return
	}
	r.subcategories[subKey(catName, *subName)] = *subID
	byCategory, ok := r.subByCategoryID[*catID]
	if !ok {
		byCategory = make(map[string]category.Subcategory)
		r.subByCategoryID[*catID] = byCategory
	}
	byCategory[name] = category.Subcategory{ID: *subID, Name: *subName, CategoryID: *catID}
}

func getIdempotency(ctx context.Context, db *sql.DB, userID, key string) (*Report, error) {
	raw, err := sqlcdb.New(db).GetImportIdempotency(ctx, sqlcdb.GetImportIdempotencyParams{
		UserID: userID, IdempotencyKey: key,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rep Report
	if err := json.Unmarshal([]byte(raw), &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func saveIdempotency(ctx context.Context, db *sql.DB, userID, key string, report Report) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return sqlcdb.New(db).InsertImportIdempotency(ctx, sqlcdb.InsertImportIdempotencyParams{
		ID: uuid.NewString(), UserID: userID, IdempotencyKey: key,
		ResponseJson: string(data), CreatedAt: now,
	})
}

func hasMapErr(errs []RowError, row int) bool {
	for _, e := range errs {
		if e.Row == row {
			return true
		}
	}
	return false
}

func appendUnique(slice []string, v string) []string {
	for _, s := range slice {
		if s == v {
			return slice
		}
	}
	return append(slice, v)
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func warnNonRUB(rows []MappedRow) {
	for _, m := range rows {
		for _, cur := range []string{m.DebitCurrency, m.CreditCurrency} {
			c := strings.TrimSpace(strings.ToUpper(cur))
			if c != "" && c != "RUB" {
				slog.Warn("import: non-RUB currency ignored", "currency", cur, "row", m.RowNum)
			}
		}
	}
}
