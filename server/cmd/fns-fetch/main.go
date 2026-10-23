// Command fns-fetch tests the production FNS client chain against a real QR.
package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func main() {
	qr := "t=20260706T1753&s=4047.50&fn=7384440900988131&i=6614&fp=0538715847&n=1"
	if v := os.Getenv("QR"); v != "" {
		qr = v
	}
	client := fns.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := client.FetchReceipt(ctx, qr)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err != nil {
		_ = enc.Encode(map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	_ = enc.Encode(map[string]any{
		"store":   out.StoreName,
		"total":   out.TotalSum,
		"items":   len(out.Items),
		"first":   itemName(out),
		"receipt": out,
	})
}

func itemName(p fns.ParsedReceipt) string {
	if len(p.Items) == 0 {
		return ""
	}
	return p.Items[0].Name
}
