package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func postImportMultipart(t *testing.T, env *testEnv, path string, fields map[string]string, filename string, fileData []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(fileData); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	req, err := http.NewRequest(http.MethodPost, env.server.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: env.cookie})
	return mustDo(t, req)
}

func mustDo(t *testing.T, req *http.Request) *http.Response {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sampleCSVRows() []byte {
	return []byte(`Тип,Дата,Сумма списания,Валюта списания,Счет списания,Сумма пополнения,Валюта назначения,Счет пополнения,Категория,Subcategory,Описание,Проект,Пользователь
Расходы,01.01.2025,50.00_-₽,RUB,Наличные,,,,Транспорт,Автобус,,,User
Расходы,02.01.2025,100.00_-₽,RUB,Яндекс,,,,Связь,Подписки,,,User
Доходы,03.01.2025,,,,200.00_-₽,RUB,Яндекс,Прочие доходы,Авито,,,User
Перевод,04.01.2025,300.00_-₽,RUB,Яндекс,300.00_-₽,RUB,Кредитка,Перевод,,,,User
`)
}

func TestImportPreviewDryRun(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp := postImportMultipart(t, env, "/api/v1/import/preview", map[string]string{
		"preset": "cubux", "deduplicate": "true",
	}, "sample.csv", sampleCSVRows())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("preview status %d: %s", resp.StatusCode, body)
	}
	var report struct {
		TotalRows        int      `json:"total_rows"`
		ValidRows        int      `json:"valid_rows"`
		AccountsToCreate []string `json:"accounts_to_create"`
		AccountMappings  []struct {
			FileName string `json:"file_name"`
			Mode     string `json:"mode"`
		} `json:"account_mappings"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&report)
	if report.TotalRows != 4 || report.ValidRows != 4 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.AccountMappings) < 3 {
		t.Fatalf("expected account_mappings, got %+v", report.AccountMappings)
	}
}

func TestImportCommitCreatesTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp := postImportMultipart(t, env, "/api/v1/import", map[string]string{
		"preset": "cubux", "deduplicate": "true", "confirm": "true",
	}, "sample.csv", sampleCSVRows())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("import status %d: %s", resp.StatusCode, body)
	}
	var report struct {
		CreatedTransactions int `json:"created_transactions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&report)
	if report.CreatedTransactions != 4 {
		t.Fatalf("expected 4 created, got %d", report.CreatedTransactions)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?limit=50", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			Type string `json:"type"`
		} `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&list)
	if list.Meta.Total < 5 { // 4 import + possibly transfer = 5 legs? transfer creates 2 legs
		t.Fatalf("expected transactions in DB, total %d", list.Meta.Total)
	}
}

func TestImportAsyncJobLifecycle(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp := postImportMultipart(t, env, "/api/v1/import/jobs", map[string]string{
		"preset": "cubux", "deduplicate": "true",
	}, "sample.csv", sampleCSVRows())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create job status %d: %s", resp.StatusCode, body)
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created.ID == "" {
		t.Fatal("expected job id")
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		statusResp, err := env.authedRequest(http.MethodGet, "/api/v1/import/jobs/"+created.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		var job struct {
			Status string `json:"status"`
			Report struct {
				CreatedTransactions int `json:"created_transactions"`
			} `json:"report"`
			ErrorMessage string `json:"error_message"`
		}
		_ = json.NewDecoder(statusResp.Body).Decode(&job)
		statusResp.Body.Close()

		if job.Status == "done" {
			if job.Report.CreatedTransactions != 4 {
				t.Fatalf("expected 4 created, got %d", job.Report.CreatedTransactions)
			}
			return
		}
		if job.Status == "failed" {
			t.Fatalf("job failed: %s", job.ErrorMessage)
		}
		if time.Now().After(deadline) {
			t.Fatalf("job did not finish in time, last status=%s", job.Status)
		}
		time.Sleep(80 * time.Millisecond)
	}
}

func TestImportTransferCreatesLinkedRecords(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	csv := []byte(`Тип,Дата,Сумма списания,Валюта списания,Счет списания,Сумма пополнения,Валюта назначения,Счет пополнения,Категория,Subcategory,Описание,Проект,Пользователь
Перевод,04.01.2025,300.00_-₽,RUB,СчётА,300.00_-₽,RUB,СчётБ,Перевод,,,,User
`)
	resp := postImportMultipart(t, env, "/api/v1/import", map[string]string{
		"preset": "cubux", "confirm": "true",
	}, "transfer.csv", csv)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	listResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions?type=transfer&limit=10", nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			TransferGroupID *string `json:"transfer_group_id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&list)
	if len(list.Data) != 2 {
		t.Fatalf("expected 2 transfer legs, got %d", len(list.Data))
	}
	if list.Data[0].TransferGroupID == nil || *list.Data[0].TransferGroupID != *list.Data[1].TransferGroupID {
		t.Fatal("transfer legs should share group_id")
	}
}

func TestImportDeduplication(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	csv := []byte(`Тип,Дата,Сумма списания,Валюта списания,Счет списания,Сумма пополнения,Валюта назначения,Счет пополнения,Категория,Subcategory,Описание,Проект,Пользователь
Расходы,01.01.2025,50.00_-₽,RUB,Наличные,,,,Транспорт,Автобус,,,User
Расходы,01.01.2025,50.00_-₽,RUB,Наличные,,,,Транспорт,Автобус,,,User
`)
	fields := map[string]string{"preset": "cubux", "deduplicate": "true", "confirm": "true"}
	resp1 := postImportMultipart(t, env, "/api/v1/import", fields, "dup.csv", csv)
	resp1.Body.Close()

	resp2 := postImportMultipart(t, env, "/api/v1/import", fields, "dup.csv", csv)
	defer resp2.Body.Close()
	var report struct {
		SkippedDuplicates   int `json:"skipped_duplicates"`
		CreatedTransactions int `json:"created_transactions"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&report)
	if report.SkippedDuplicates < 2 {
		t.Fatalf("expected skipped duplicates on re-import, got %+v", report)
	}
}

func TestImport1CSVPreview(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "1.csv"))
	if err != nil {
		t.Skip("1.csv missing")
	}
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	resp := postImportMultipart(t, env, "/api/v1/import/preview", map[string]string{
		"preset": "cubux", "deduplicate": "true",
	}, "1.csv", data)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var report struct {
		TotalRows int `json:"total_rows"`
		ValidRows int `json:"valid_rows"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&report)
	if report.TotalRows < 1800 || report.ValidRows < 1800 {
		t.Fatalf("1.csv preview: %+v", report)
	}
}

func TestImport1XLSXAsyncJob(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "1.xlsx"))
	if err != nil {
		t.Skip("1.xlsx missing")
	}

	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp := postImportMultipart(t, env, "/api/v1/import/jobs", map[string]string{
		"preset": "cubux", "deduplicate": "false",
	}, "1.xlsx", data)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create job status %d: %s", resp.StatusCode, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created.ID == "" {
		t.Fatal("expected job id")
	}

	deadline := time.Now().Add(40 * time.Second)
	for {
		statusResp, err := env.authedRequest(http.MethodGet, "/api/v1/import/jobs/"+created.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		var job struct {
			Status string `json:"status"`
			Report struct {
				CreatedTransactions int `json:"created_transactions"`
				TotalRows           int `json:"total_rows"`
				ValidRows           int `json:"valid_rows"`
				Errors              []struct {
					Row     int    `json:"row"`
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"report"`
			ErrorMessage string `json:"error_message"`
		}
		_ = json.NewDecoder(statusResp.Body).Decode(&job)
		statusResp.Body.Close()

		if job.Status == "done" {
			if job.Report.TotalRows < 1800 || job.Report.ValidRows < 1700 {
				sample := ""
				if len(job.Report.Errors) > 0 {
					e := job.Report.Errors[0]
					sample = e.Message
				}
				t.Fatalf("unexpected report rows: total=%d valid=%d errors=%d sample=%q", job.Report.TotalRows, job.Report.ValidRows, len(job.Report.Errors), sample)
			}
			if job.Report.CreatedTransactions < 1700 {
				t.Fatalf("too few created: %d", job.Report.CreatedTransactions)
			}
			return
		}
		if job.Status == "failed" {
			t.Fatalf("job failed: %s", job.ErrorMessage)
		}
		if time.Now().After(deadline) {
			t.Fatalf("job timed out, last status=%s", job.Status)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func TestExportRoundtrip(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	csv := sampleCSVRows()
	importResp := postImportMultipart(t, env, "/api/v1/import", map[string]string{
		"preset": "cubux", "confirm": "true",
	}, "roundtrip.csv", csv)
	importResp.Body.Close()

	req, _ := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/export?from=2025-01-01&to=2025-01-31", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: env.cookie})
	exportResp := mustDo(t, req)
	defer exportResp.Body.Close()
	if exportResp.StatusCode != http.StatusOK {
		t.Fatalf("export status %d", exportResp.StatusCode)
	}
	body, _ := io.ReadAll(exportResp.Body)
	if len(body) < 100 {
		t.Fatal("export too small")
	}
	if ct := exportResp.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
}
