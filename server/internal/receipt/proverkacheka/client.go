package proverkacheka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

const defaultEndpoint = "https://proverkacheka.com/api/v1/check/get"

// Client calls proverkacheka.com check/get with an instance token.
type Client struct {
	Token      string
	Endpoint   string
	HTTPClient *http.Client
}

// NewClient returns a client bound to the given API token.
func NewClient(token string) *Client {
	return &Client{Token: strings.TrimSpace(token)}
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func (c *Client) endpoint() string {
	if strings.TrimSpace(c.Endpoint) != "" {
		return strings.TrimSpace(c.Endpoint)
	}
	return defaultEndpoint
}

// FetchReceipt loads receipt details for a 54-FZ QR string.
func (c *Client) FetchReceipt(ctx context.Context, qr string) (fns.ParsedReceipt, error) {
	token := strings.TrimSpace(c.Token)
	if token == "" {
		return fns.ParsedReceipt{}, fns.ErrFetchFailed
	}
	params, err := fns.ParseQR(qr)
	if err != nil {
		return fns.ParsedReceipt{}, err
	}

	body, _ := json.Marshal(map[string]string{
		"token": token,
		"qrraw": strings.TrimSpace(qr),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "buhgalter/1.10")

	resp, err := c.http().Do(req)
	if err != nil {
		return fns.ParsedReceipt{}, fmt.Errorf("%w: %v", fns.ErrUnavailable, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return fns.ParsedReceipt{}, fmt.Errorf("%w: status %d", fns.ErrFetchFailed, resp.StatusCode)
	}

	var wrapped struct {
		Code int `json:"code"`
		Data struct {
			JSON json.RawMessage `json:"json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return fns.ParsedReceipt{}, fmt.Errorf("%w: %v", fns.ErrFetchFailed, err)
	}
	if wrapped.Code != 1 {
		if wrapped.Code == 0 {
			return fns.ParsedReceipt{}, fns.ErrNotFound
		}
		return fns.ParsedReceipt{}, fmt.Errorf("%w: provider code %d", fns.ErrFetchFailed, wrapped.Code)
	}
	if len(wrapped.Data.JSON) == 0 {
		return fns.ParsedReceipt{}, fmt.Errorf("%w: empty json", fns.ErrFetchFailed)
	}
	return parseProviderJSON(params, wrapped.Data.JSON)
}

func parseProviderJSON(p fns.QRParams, raw json.RawMessage) (fns.ParsedReceipt, error) {
	var direct receiptJSON
	if err := json.Unmarshal(raw, &direct); err == nil && (direct.TotalSum > 0 || len(direct.Items) > 0) {
		return mapReceipt(p, direct), nil
	}
	// Provider returns FNS-shaped ticket JSON under data.json.
	var ticket struct {
		Ticket struct {
			Document struct {
				Receipt json.RawMessage `json:"receipt"`
			} `json:"document"`
		} `json:"ticket"`
	}
	if err := json.Unmarshal(raw, &ticket); err == nil && len(ticket.Ticket.Document.Receipt) > 0 {
		return fns.ParseReceiptResponse(p, ticket.Ticket.Document.Receipt)
	}
	return fns.ParseReceiptResponse(p, raw)
}
