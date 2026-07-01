package importexport

import "time"

// RawTable is a parsed spreadsheet: header row + data rows (1-based file row numbers).
type RawTable struct {
	Headers []string
	Rows    []RawRow
}

type RawRow struct {
	RowNum int
	Values []string
}

// ColumnMap maps target field names to source column headers (custom preset).
type ColumnMap map[string]string

// Known mapping targets for custom imports.
const (
	ColType           = "type"
	ColDate           = "date"
	ColDebitAmount    = "debit_amount"
	ColDebitAccount   = "debit_account"
	ColCreditAmount   = "credit_amount"
	ColCreditAccount  = "credit_account"
	ColCategory       = "category"
	ColSubcategory    = "subcategory"
	ColDescription    = "description"
	ColProject        = "project"
	ColUser           = "user"
	ColDebitCurrency  = "debit_currency"
	ColCreditCurrency = "credit_currency"
)

// MappedRow is a normalized row ready for resolution.
type MappedRow struct {
	RowNum         int
	CubuxType      string // Расходы | Доходы | Перевод (or mapped equivalents)
	Date           time.Time
	DebitAmount    int64
	CreditAmount   int64
	DebitAccount   string
	CreditAccount  string
	Category       string
	Subcategory    string
	Description    string
	Project        string
	User           string
	DebitCurrency  string
	CreditCurrency string
}

// TxAction describes what will be created.
type TxAction string

const (
	ActionCreateExpense  TxAction = "create_expense"
	ActionCreateIncome   TxAction = "create_income"
	ActionCreateTransfer TxAction = "create_transfer"
)

// PreviewItem is one row in the import preview response.
type PreviewItem struct {
	Row         int      `json:"row"`
	Action      TxAction `json:"action"`
	Account     string   `json:"account,omitempty"`
	ToAccount   string   `json:"to_account,omitempty"`
	Amount      int64    `json:"amount"`
	Category    string   `json:"category,omitempty"`
	Subcategory string   `json:"subcategory,omitempty"`
	Date        string   `json:"date"`
	Description string   `json:"description,omitempty"`
}

// RowError is a per-row validation error.
type RowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// AccountMapEntry maps a file account name to create or existing account.
type AccountMapEntry struct {
	Mode        string `json:"mode"` // create | existing
	AccountID   string `json:"account_id,omitempty"`
	AccountType string `json:"account_type,omitempty"` // cash | bank | credit_card (create)
	BankID      string `json:"bank_id,omitempty"`      // bank / credit_card
	CreditLimit string `json:"credit_limit,omitempty"` // credit_card create
}

// AccountMappingSuggestion is a proposed mapping for one account name from the file.
type AccountMappingSuggestion struct {
	FileName    string  `json:"file_name"`
	Mode        string  `json:"mode"` // existing | create
	AccountID   *string `json:"account_id,omitempty"`
	AccountName *string `json:"account_name,omitempty"`
	AccountType *string `json:"account_type,omitempty"` // cash | bank | credit_card (create)
	BankID      *string `json:"bank_id,omitempty"`
	CreditLimit *string `json:"credit_limit,omitempty"`
}

// CategoryMapEntry maps a file category to create or existing category.
type CategoryMapEntry struct {
	Mode       string `json:"mode"` // create | existing
	CategoryID string `json:"category_id,omitempty"`
}

// CategoryMappingSuggestion is a proposed mapping for one category from the file.
type CategoryMappingSuggestion struct {
	FileName     string  `json:"file_name"`
	Type         string  `json:"type"` // expense | income
	Mode         string  `json:"mode"` // existing | create
	CategoryID   *string `json:"category_id,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
}

// SubcategoryMapEntry maps a file subcategory to create or existing subcategory.
type SubcategoryMapEntry struct {
	Mode          string `json:"mode"` // create | existing
	SubcategoryID string `json:"subcategory_id,omitempty"`
}

// SubcategoryMappingSuggestion is a proposed mapping for one subcategory from the file.
type SubcategoryMappingSuggestion struct {
	FileCategory    string  `json:"file_category"`
	FileSubcategory string  `json:"file_subcategory"`
	Type            string  `json:"type"` // expense | income
	Mode            string  `json:"mode"` // existing | create
	SubcategoryID   *string `json:"subcategory_id,omitempty"`
	SubcategoryName *string `json:"subcategory_name,omitempty"`
}

// ImportOptions configures preview/commit.
type ImportOptions struct {
	Preset          string
	Deduplicate     bool
	ColumnMap       ColumnMap
	AccountMap      map[string]AccountMapEntry
	CategoryMap     map[string]CategoryMapEntry
	SubcategoryMap  map[string]SubcategoryMapEntry
	AutoSubcategory bool
	Confirm         bool
	IdempotencyKey  string
}

// Report is the API response for preview and commit.
type Report struct {
	TotalRows           int                            `json:"total_rows"`
	ProcessedRows       int                            `json:"processed_rows,omitempty"`
	ValidRows           int                            `json:"valid_rows"`
	SkippedDuplicates   int                            `json:"skipped_duplicates"`
	CreatedTransactions int                            `json:"created_transactions,omitempty"`
	Errors              []RowError                     `json:"errors"`
	Logs                []string                       `json:"logs,omitempty"`
	Preview             []PreviewItem                  `json:"preview"`
	AccountsToCreate    []string                       `json:"accounts_to_create"`
	AccountMappings     []AccountMappingSuggestion     `json:"account_mappings"`
	CategoryMappings    []CategoryMappingSuggestion    `json:"category_mappings"`
	SubcategoryMappings []SubcategoryMappingSuggestion `json:"subcategory_mappings"`
	CategoriesToCreate  []string                       `json:"categories_to_create"`
}

// ExportFilters configures CSV export.
type ExportFilters struct {
	From       string
	To         string
	AccountID  string
	CategoryID string
}
