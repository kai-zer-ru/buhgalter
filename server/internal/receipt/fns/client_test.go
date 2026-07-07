package fns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testQR = "t=20260115T1030&s=150.00&fn=7288940700123456&i=12345&fp=9876543210&n=1"

func receiptJSONBody() string {
	return `{"code":1,"data":{"json":{"ticket":{"document":{"receipt":{"dateTime":"2026-01-15T10:30:00","totalSum":15000,"retailPlace":"Магнит","userInn":"7701234567","items":[{"name":"Молоко","quantity":1,"price":10000,"sum":10000},{"name":"Хлеб","quantity":1,"price":5000,"sum":5000}]}}}}}}`
}

func TestClientFetchReceipt_mapsGETResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !stringsHasPrefix(r.URL.Path, "/v1/inns/") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Device-OS") == "" {
			t.Fatal("expected Device-OS header")
		}
		if r.URL.Query().Get("fiscalSign") != "9876543210" {
			t.Fatalf("fiscalSign: %s", r.URL.Query().Get("fiscalSign"))
		}
		_, _ = w.Write([]byte(receiptJSONBody()))
	}))
	t.Cleanup(srv.Close)

	client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	out, err := client.FetchReceipt(context.Background(), testQR)
	if err != nil {
		t.Fatal(err)
	}
	if out.StoreName != "Магнит" || out.TotalSum != 15000 || len(out.Items) != 2 {
		t.Fatalf("unexpected receipt: %+v", out)
	}
	if out.FNSCheckURL == "" {
		t.Fatal("expected fns check url")
	}
	want := time.Date(2026, 1, 15, 10, 30, 0, 0, moscowLocation())
	if !out.DateTime.Equal(want) {
		t.Fatalf("expected %v, got %v", want, out.DateTime)
	}
}

func TestBuildQRFallback(t *testing.T) {
	t.Parallel()
	p, err := ParseQR(testQR)
	if err != nil {
		t.Fatal(err)
	}
	out := buildQRFallback(p)
	if len(out.Items) != 1 || out.TotalSum != p.TotalSumKopecks {
		t.Fatalf("fallback: %+v", out)
	}
	if out.Items[0].Sum != p.TotalSumKopecks {
		t.Fatalf("item sum %d", out.Items[0].Sum)
	}
}

func TestFNSDate_moscowLocal(t *testing.T) {
	t.Parallel()
	p, err := ParseQR(testQR)
	if err != nil {
		t.Fatal(err)
	}
	if got := FNSDate(p.DateTime); got != "2026-01-15T10:30:00" {
		t.Fatalf("FNSDate %q", got)
	}
}

func TestParseReceiptDateTime_unix(t *testing.T) {
	t.Parallel()
	raw := []byte(`1595740860`)
	dt, err := parseReceiptDateTime(raw, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if dt.IsZero() {
		t.Fatal("expected parsed time")
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
