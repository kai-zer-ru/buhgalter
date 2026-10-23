package proverkacheka

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

type receiptJSON struct {
	DateTime    json.RawMessage `json:"dateTime"`
	TotalSum    int64           `json:"totalSum"`
	User        string          `json:"user"`
	UserINN     string          `json:"userInn"`
	RetailPlace string          `json:"retailPlace"`
	Items       []struct {
		Name     string  `json:"name"`
		Quantity float64 `json:"quantity"`
		Price    int64   `json:"price"`
		Sum      int64   `json:"sum"`
	} `json:"items"`
}

func mapReceipt(p fns.QRParams, rcpt receiptJSON) fns.ParsedReceipt {
	storeName := strings.TrimSpace(rcpt.User)
	if storeName == "" {
		storeName = strings.TrimSpace(rcpt.RetailPlace)
	}
	items := make([]fns.Item, 0, len(rcpt.Items))
	for _, it := range rcpt.Items {
		items = append(items, fns.Item{
			Name:     strings.TrimSpace(it.Name),
			Quantity: it.Quantity,
			Price:    it.Price,
			Sum:      it.Sum,
		})
	}
	total := rcpt.TotalSum
	if total <= 0 {
		total = p.TotalSumKopecks
	}
	return fns.ParsedReceipt{
		FiscalDriveNumber:    p.FiscalDriveNumber,
		FiscalDocumentNumber: p.FiscalDocumentNumber,
		FiscalSign:           p.FiscalSign,
		DateTime:             parseDateTime(rcpt.DateTime, p.DateTime),
		TotalSum:             total,
		StoreName:            storeName,
		UserINN:              strings.TrimSpace(rcpt.UserINN),
		RetailPlace:          strings.TrimSpace(rcpt.RetailPlace),
		Items:                items,
		FNSCheckURL:          fns.CheckURL(p),
	}
}

// parseDateTime reads provider dateTime (local MSK string, RFC3339 or unix seconds);
// falls back to the QR timestamp.
func parseDateTime(raw json.RawMessage, fallback time.Time) time.Time {
	if len(raw) == 0 {
		return fallback
	}
	msk := fallback.Location()
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSuffix(strings.TrimSpace(s), "Z")
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, msk); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
		return fallback
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil && n > 0 {
		return time.Unix(int64(n), 0).In(msk)
	}
	return fallback
}
