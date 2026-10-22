package importexport

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

const (
	nativeMagic       = "#BUHGALTER"
	nativeVersion     = "1"
	sectionAccounts   = "accounts"
	sectionCategories = "categories"
	sectionSubcats    = "subcategories"
	sectionMerchants  = "merchants"
	sectionTags       = "tags"
	sectionTx         = "transactions"
	sectionCredits    = "credits"
	sectionCreditPays = "credit_payments"
	sectionDebtors    = "debtors"
	sectionDebts      = "debts"
	sectionDebtLinks  = "debt_links"
	sectionSubs       = "subscriptions"
	sectionRecurring  = "recurring"
	sectionBudgets    = "budgets"
	sectionTemplates  = "templates"
	sectionProfile    = "profile"
)

var nativeAccountHeaders = []string{
	"id", "name", "type", "bank", "initial_balance", "credit_limit", "is_primary", "status",
	"payment_account", "auto_topup_enabled", "auto_topup_threshold", "auto_topup_target", "auto_topup_source",
}

var nativeCategoryHeaders = []string{"name", "type", "icon", "sort_order", "is_primary", "is_system"}
var nativeSubcatHeaders = []string{"category", "category_type", "name", "icon", "sort_order"}
var nativeMerchantHeaders = []string{"name", "icon"}
var nativeTagHeaders = []string{"name"}
var nativeCreditHeaders = []string{
	"id", "name", "kind", "principal", "property_price", "down_payment", "down_payment_affects_balance",
	"down_payment_tx_id", "principal_affects_balance", "principal_tx_id", "issue_date", "term_months",
	"interest_rate", "payment_interval", "paid_amount", "monthly_payment", "debit_account", "debit_time_local",
	"bank", "bank_locked", "added_retroactively", "recorded_at", "status", "closed_at",
}
var nativeCreditPayHeaders = []string{"id", "credit_id", "transaction_id", "amount", "payment_date", "kind", "is_applied", "exclude_from_stats"}
var nativeDebtorHeaders = []string{"id", "name"}
var nativeDebtHeaders = []string{
	"id", "debtor_id", "debtor", "direction", "amount", "affects_balance", "debt_date", "due_date",
	"description", "transaction_id", "is_settled", "settled_at",
}
var nativeDebtLinkHeaders = []string{"debt_id", "transaction_id", "role"}
var nativeSubHeaders = []string{
	"id", "name", "description", "icon", "website_url", "amount", "account", "subcategory", "period",
	"weekday", "day_of_month", "start_date", "time_local", "next_run_at", "upcoming_run_ats", "last_run_at", "active",
}
var nativeRecurringHeaders = []string{
	"id", "type", "amount", "description", "account", "category", "subcategory", "period",
	"weekday", "day_of_month", "start_date", "time_local", "next_run_at", "last_run_at", "active",
}
var nativeBudgetHeaders = []string{
	"id", "name", "scope", "category", "subcategory", "amount", "account", "month",
	"copy_forward", "rollover", "alert_at_percent", "is_active",
}
var nativeTemplateHeaders = []string{
	"name", "type", "account", "to_account", "category", "subcategory", "amount", "description", "merchant", "icon", "tags",
}
var nativeProfileHeaders = []string{"timezone", "currency", "language"}

// NativeFile is a sectioned Buhgalter CSV.
type NativeFile struct {
	Sections map[string]RawTable
}

func IsNativeCSV(data []byte) bool {
	data = StripUTF8BOM(decodeImportText(data))
	s := strings.TrimLeft(string(data), "\ufeff \t\r\n\"'")
	if strings.HasPrefix(strings.ToLower(s), "sep=") {
		if i := strings.IndexAny(s, "\r\n"); i >= 0 {
			s = strings.TrimLeft(s[i:], "\r\n\"'")
		}
	}
	return strings.HasPrefix(s, nativeMagic)
}

func ParseNativeCSV(data []byte) (NativeFile, error) {
	data = StripUTF8BOM(decodeImportText(data))
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = ','
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	all, err := r.ReadAll()
	if err != nil {
		return NativeFile{}, fmt.Errorf("csv: %w", err)
	}
	return parseNativeRecords(all)
}

func parseNativeRecords(all [][]string) (NativeFile, error) {
	out := NativeFile{Sections: make(map[string]RawTable)}
	var name string
	var headers []string
	var rows []RawRow
	flush := func() {
		if name == "" {
			return
		}
		out.Sections[name] = RawTable{Headers: headers, Rows: rows}
	}
	for i, rec := range all {
		if isEmptyRow(rec) {
			continue
		}
		cell0 := strings.TrimSpace(rec[0])
		if strings.HasPrefix(strings.ToLower(cell0), "sep=") {
			continue
		}
		if strings.EqualFold(strings.Trim(cell0, "\"'"), nativeMagic) {
			continue
		}
		if strings.EqualFold(cell0, "#SECTION") {
			flush()
			name = ""
			headers = nil
			rows = nil
			if len(rec) > 1 {
				name = strings.ToLower(strings.TrimSpace(rec[1]))
			}
			continue
		}
		if name == "" {
			continue
		}
		if headers == nil {
			headers = make([]string, len(rec))
			for j, h := range rec {
				headers[j] = NormalizeHeader(h)
			}
			continue
		}
		rows = append(rows, RawRow{RowNum: i + 1, Values: rec})
	}
	flush()
	return out, nil
}

type nativeSection struct {
	name    string
	headers []string
	rows    [][]string
}

func writeNativeCSV(sections []nativeSection) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{nativeMagic, nativeVersion}); err != nil {
		return nil, err
	}
	for _, sec := range sections {
		if err := w.Write([]string{"#SECTION", sec.name}); err != nil {
			return nil, err
		}
		if err := w.Write(sec.headers); err != nil {
			return nil, err
		}
		for _, row := range sec.rows {
			if err := w.Write(row); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func nativeCell(table RawTable, row RawRow, names ...string) string {
	idx := cubuxFieldIndex(table.Headers)
	return cellAt(row, idx, names...)
}

func csvBool(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func parseCSVBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes"
}

func csvInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func csvAmountPtr(v *int64) string {
	if v == nil {
		return ""
	}
	return FormatCubuxAmount(*v)
}

func csvStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func parseOptionalAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return ParseCubuxAmount(s)
}

func parseOptionalInt(s string) *int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseOptionalFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return n
}
