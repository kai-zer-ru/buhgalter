package proverkacheka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func TestMapReceipt_fromDirectJSON(t *testing.T) {
	p := fns.QRParams{
		DateTime:             time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		TotalSumKopecks:      15000,
		FiscalDriveNumber:    "7288940700123456",
		FiscalDocumentNumber: "12345",
		FiscalSign:           "9876543210",
	}
	raw := receiptJSON{
		DateTime: json.RawMessage(`"2026-01-15T10:30:00"`),
		TotalSum: 15000,
		User:     "Магнит",
		Items: []struct {
			Name     string  `json:"name"`
			Quantity float64 `json:"quantity"`
			Price    int64   `json:"price"`
			Sum      int64   `json:"sum"`
		}{{Name: "Молоко", Quantity: 1, Price: 15000, Sum: 15000}},
	}
	out := mapReceipt(p, raw)
	if out.StoreName != "Магнит" || out.TotalSum != 15000 || len(out.Items) != 1 {
		t.Fatalf("unexpected mapped receipt: %+v", out)
	}
}
