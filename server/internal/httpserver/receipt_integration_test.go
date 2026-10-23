package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/features"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

const testNotificationSecret = "12345678901234567890123456789012"

func enableReceiptScanner(t *testing.T, env *testEnv) {
	t.Helper()
	features.Invalidate()
	if _, err := env.db.Exec(
		`UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`,
		testNotificationSecret,
	); err != nil {
		t.Fatal(err)
	}
	secretBody, _ := json.Marshal(map[string]string{"receipt_api_token": "integration-test-token"})
	secretResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/settings/receipt-api-token", bytes.NewReader(secretBody))
	if err != nil {
		t.Fatal(err)
	}
	secretResp.Body.Close()
	if secretResp.StatusCode != http.StatusOK {
		t.Fatalf("receipt-api-token status %d", secretResp.StatusCode)
	}
	flagsBody, _ := json.Marshal(map[string]bool{features.ReceiptScanner: true})
	flagsResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/features", bytes.NewReader(flagsBody))
	if err != nil {
		t.Fatal(err)
	}
	flagsResp.Body.Close()
	if flagsResp.StatusCode != http.StatusOK {
		t.Fatalf("features status %d", flagsResp.StatusCode)
	}
}

func sampleAPIReceipt() fns.ParsedReceipt {
	dt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	return fns.ParsedReceipt{
		FiscalDriveNumber:    "7288940700123456",
		FiscalDocumentNumber: "99999",
		FiscalSign:           "111222333",
		DateTime:             dt,
		TotalSum:             10000,
		StoreName:            "Пятёрочка",
		Items: []fns.Item{
			{Name: "Сыр", Quantity: 1, Price: 10000, Sum: 10000},
		},
		FNSCheckURL: "https://proverkacheka.nalog.ru/?fn=7288940700123456",
	}
}

func TestReceiptImportAndGet(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	enableReceiptScanner(t, env)
	accID := createTestAccount(t, env, "Чеки")
	catID := getExpenseCategory(t, env)
	receipt := sampleAPIReceipt()

	importBody, _ := json.Marshal(map[string]any{
		"account_id":             accID,
		"fiscal_drive_number":    receipt.FiscalDriveNumber,
		"fiscal_document_number": receipt.FiscalDocumentNumber,
		"fiscal_sign":            receipt.FiscalSign,
		"receipt":                receipt,
		"items": []map[string]any{
			{"index": 0, "category_id": catID, "include": true},
		},
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/receipts/import", bytes.NewReader(importBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import status %d", resp.StatusCode)
	}
	var imported struct {
		ReceiptImportID string `json:"receipt_import_id"`
		Transactions    []struct {
			ID              string  `json:"id"`
			ReceiptImportID *string `json:"receipt_import_id"`
			StoreName       *string `json:"store_name"`
			Description     *string `json:"description"`
		} `json:"transactions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&imported)
	if imported.ReceiptImportID == "" || len(imported.Transactions) != 1 {
		t.Fatalf("unexpected import response: %+v", imported)
	}
	if imported.Transactions[0].StoreName == nil || *imported.Transactions[0].StoreName != "Пятёрочка" {
		t.Fatalf("expected store name, got %+v", imported.Transactions[0])
	}

	dupResp, _ := env.authedRequest(http.MethodPost, "/api/v1/receipts/import", bytes.NewReader(importBody))
	defer dupResp.Body.Close()
	if dupResp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate import status %d", dupResp.StatusCode)
	}

	getResp, err := env.authedRequest(http.MethodGet,
		"/api/v1/receipts/"+imported.ReceiptImportID+"?transaction_id="+imported.Transactions[0].ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get receipt status %d", getResp.StatusCode)
	}
	var detail struct {
		Items              []fns.Item `json:"items"`
		FNSCheckURL        string     `json:"fns_check_url"`
		HighlightItemIndex *int       `json:"highlight_item_index"`
		StoreName          string     `json:"store_name"`
	}
	_ = json.NewDecoder(getResp.Body).Decode(&detail)
	if len(detail.Items) != 1 || detail.FNSCheckURL == "" || detail.StoreName != "Пятёрочка" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.HighlightItemIndex == nil || *detail.HighlightItemIndex != 0 {
		t.Fatalf("expected highlight 0, got %v", detail.HighlightItemIndex)
	}

	storesResp, err := env.authedRequest(http.MethodGet, "/api/v1/stores", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer storesResp.Body.Close()
	var stores []struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(storesResp.Body).Decode(&stores)
	if len(stores) != 1 || stores[0].Name != "Пятёрочка" {
		t.Fatalf("unexpected stores: %+v", stores)
	}
}

func TestStatsByStore(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	enableReceiptScanner(t, env)
	accID := createTestAccount(t, env, "Магазины")
	catID := getExpenseCategory(t, env)
	receipt := sampleAPIReceipt()
	receipt.FiscalDocumentNumber = "88888"
	receipt.FiscalSign = "444555666"

	importBody, _ := json.Marshal(map[string]any{
		"account_id":             accID,
		"fiscal_drive_number":    receipt.FiscalDriveNumber,
		"fiscal_document_number": receipt.FiscalDocumentNumber,
		"fiscal_sign":            receipt.FiscalSign,
		"receipt":                receipt,
		"items": []map[string]any{
			{"index": 0, "category_id": catID, "include": true},
		},
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/receipts/import", bytes.NewReader(importBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import status %d", resp.StatusCode)
	}

	statsResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/by-store?type=expense", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer statsResp.Body.Close()
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("by-store status %d", statsResp.StatusCode)
	}
	var stats struct {
		Items []struct {
			StoreName string `json:"store_name"`
			Total     int64  `json:"total"`
			Count     int64  `json:"count"`
		} `json:"items"`
	}
	_ = json.NewDecoder(statsResp.Body).Decode(&stats)
	if len(stats.Items) == 0 || stats.Items[0].StoreName != "Пятёрочка" || stats.Items[0].Total != 10000 {
		t.Fatalf("unexpected by-store: %+v", stats)
	}
}
