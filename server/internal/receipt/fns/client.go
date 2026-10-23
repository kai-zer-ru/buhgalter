package fns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotFound          = errors.New("receipt not found")
	ErrUnavailable       = errors.New("fns unavailable")
	ErrFetchFailed       = errors.New("fns fetch failed")
	defaultBaseURL       = "https://proverkacheka.nalog.ru:9999"
	defaultTimeout       = 25 * time.Second
	proverkachekaTimeout = 8 * time.Second
)

type Item struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Price    int64   `json:"price"`
	Sum      int64   `json:"sum"`
}

type ParsedReceipt struct {
	FiscalDriveNumber    string    `json:"fiscal_drive_number"`
	FiscalDocumentNumber string    `json:"fiscal_document_number"`
	FiscalSign           string    `json:"fiscal_sign"`
	DateTime             time.Time `json:"date_time"`
	TotalSum             int64     `json:"total_sum"`
	StoreName            string    `json:"store_name"`
	UserINN              string    `json:"user_inn,omitempty"`
	RetailPlace          string    `json:"retail_place,omitempty"`
	Items                []Item    `json:"items"`
	FNSCheckURL          string    `json:"fns_check_url"`
	AlreadyImported      bool      `json:"already_imported,omitempty"`
	AlreadyInLedger      bool      `json:"already_in_ledger,omitempty"`
}

type Payload struct {
	FiscalDriveNumber    string `json:"fiscal_drive_number"`
	FiscalDocumentNumber string `json:"fiscal_document_number"`
	FiscalSign           string `json:"fiscal_sign"`
	DateTime             string `json:"date_time"`
	TotalSum             int64  `json:"total_sum"`
	StoreName            string `json:"store_name"`
	UserINN              string `json:"user_inn,omitempty"`
	RetailPlace          string `json:"retail_place,omitempty"`
	Items                []Item `json:"items"`
}

type Client struct {
	BaseURL    string
	SessionID  string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL:   defaultBaseURL,
		SessionID: strings.TrimSpace(os.Getenv("BUHGALTER_FNS_SESSION_ID")),
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

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

type ticketResponse struct {
	Code int `json:"code"`
	Data struct {
		JSON struct {
			Ticket struct {
				Document struct {
					Receipt receiptJSON `json:"receipt"`
				} `json:"document"`
			} `json:"ticket"`
		} `json:"json"`
	} `json:"data"`
}

func (c *Client) FetchReceipt(ctx context.Context, qr string) (ParsedReceipt, error) {
	params, err := ParseQR(qr)
	if err != nil {
		return ParsedReceipt{}, err
	}

	if c.SessionID != "" {
		if out, err := c.fetchMobile(ctx, qr, params); err == nil {
			return out, nil
		} else if errors.Is(err, ErrNotFound) {
			return ParsedReceipt{}, err
		}
	}

	if out, err := c.fetchProverkacheka(ctx, params); err == nil {
		return out, nil
	} else if errors.Is(err, ErrNotFound) {
		return ParsedReceipt{}, err
	}

	if err := c.validateKktOnline(ctx, params); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ParsedReceipt{}, err
		}
		return ParsedReceipt{}, err
	}
	return buildQRFallback(params), nil
}

func (c *Client) proverkachekaHTTP() *http.Client {
	if c.HTTPClient != nil && c.HTTPClient.Timeout > 0 && c.HTTPClient.Timeout < proverkachekaTimeout {
		return c.HTTPClient
	}
	return &http.Client{Timeout: proverkachekaTimeout}
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultTimeout}
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultBaseURL
}

func (c *Client) setMobileHeaders(req *http.Request) {
	req.Header.Set("Device-Id", "")
	req.Header.Set("Device-OS", "Android")
	req.Header.Set("Accept", "application/json")
}

func (c *Client) fetchProverkacheka(ctx context.Context, p QRParams) (ParsedReceipt, error) {
	endpoint := fmt.Sprintf("%s/v1/inns/*/kkts/*/fss/%s/tickets/%s?fiscalSign=%s&sendToEmail=no",
		c.base(),
		url.PathEscape(p.FiscalDriveNumber),
		url.PathEscape(p.FiscalDocumentNumber),
		url.QueryEscape(p.FiscalSign),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ParsedReceipt{}, err
	}
	c.setMobileHeaders(req)

	resp, err := c.proverkachekaHTTP().Do(req)
	if err != nil {
		return ParsedReceipt{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ParsedReceipt{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return ParsedReceipt{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return ParsedReceipt{}, fmt.Errorf("%w: status %d", ErrFetchFailed, resp.StatusCode)
	}
	return parseReceiptResponse(p, data)
}

// ParseReceiptResponse maps FNS-shaped receipt JSON bytes to ParsedReceipt.
func ParseReceiptResponse(p QRParams, data []byte) (ParsedReceipt, error) {
	return parseReceiptResponse(p, data)
}

func parseReceiptResponse(p QRParams, data []byte) (ParsedReceipt, error) {
	var wrapped ticketResponse
	if err := json.Unmarshal(data, &wrapped); err == nil {
		rcpt := wrapped.Data.JSON.Ticket.Document.Receipt
		if rcpt.TotalSum > 0 || len(rcpt.Items) > 0 {
			return mapReceiptJSON(p, rcpt)
		}
		if wrapped.Code != 0 && wrapped.Code != 1 {
			return ParsedReceipt{}, ErrNotFound
		}
	}

	var alt struct {
		Document struct {
			Receipt receiptJSON `json:"receipt"`
		} `json:"document"`
	}
	if err := json.Unmarshal(data, &alt); err == nil {
		rcpt := alt.Document.Receipt
		if rcpt.TotalSum > 0 || len(rcpt.Items) > 0 {
			return mapReceiptJSON(p, rcpt)
		}
	}

	var mobile struct {
		Ticket struct {
			Document struct {
				Receipt receiptJSON `json:"receipt"`
			} `json:"document"`
		} `json:"ticket"`
	}
	if err := json.Unmarshal(data, &mobile); err == nil {
		rcpt := mobile.Ticket.Document.Receipt
		if rcpt.TotalSum > 0 || len(rcpt.Items) > 0 {
			return mapReceiptJSON(p, rcpt)
		}
	}

	return ParsedReceipt{}, fmt.Errorf("%w: empty receipt body", ErrFetchFailed)
}

func mapReceiptJSON(p QRParams, rcpt receiptJSON) (ParsedReceipt, error) {
	dt, err := parseReceiptDateTime(rcpt.DateTime, p.DateTime)
	if err != nil {
		dt = p.DateTime
	}
	storeName := strings.TrimSpace(rcpt.RetailPlace)
	if storeName == "" {
		storeName = strings.TrimSpace(rcpt.User)
	}
	items := make([]Item, 0, len(rcpt.Items))
	for _, it := range rcpt.Items {
		items = append(items, Item{
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
	return ParsedReceipt{
		FiscalDriveNumber:    p.FiscalDriveNumber,
		FiscalDocumentNumber: p.FiscalDocumentNumber,
		FiscalSign:           p.FiscalSign,
		DateTime:             dt,
		TotalSum:             total,
		StoreName:            storeName,
		UserINN:              rcpt.UserINN,
		RetailPlace:          rcpt.RetailPlace,
		Items:                items,
		FNSCheckURL:          CheckURL(p),
	}, nil
}

func parseReceiptDateTime(raw json.RawMessage, fallback time.Time) (time.Time, error) {
	if len(raw) == 0 {
		return fallback, errors.New("empty datetime")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSuffix(strings.TrimSpace(s), "Z")
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, moscowLocation()); err == nil {
			return t, nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, nil
		}
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		if i, err := n.Int64(); err == nil {
			return time.Unix(i, 0).In(moscowLocation()), nil
		}
		if f, err := n.Float64(); err == nil {
			return time.Unix(int64(f), 0).In(moscowLocation()), nil
		}
	}
	return fallback, errors.New("unparsed datetime")
}

func ToPayload(p ParsedReceipt) Payload {
	return Payload{
		FiscalDriveNumber:    p.FiscalDriveNumber,
		FiscalDocumentNumber: p.FiscalDocumentNumber,
		FiscalSign:           p.FiscalSign,
		DateTime:             p.DateTime.UTC().Format(time.RFC3339),
		TotalSum:             p.TotalSum,
		StoreName:            p.StoreName,
		UserINN:              p.UserINN,
		RetailPlace:          p.RetailPlace,
		Items:                p.Items,
	}
}

func PayloadFromJSON(raw string) (Payload, error) {
	var p Payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Payload{}, err
	}
	return p, nil
}

// ParsedFromPayload rebuilds ParsedReceipt from stored import payload.
func ParsedFromPayload(p Payload, fnsCheckURL string) (ParsedReceipt, error) {
	dt, err := time.Parse(time.RFC3339, p.DateTime)
	if err != nil {
		return ParsedReceipt{}, err
	}
	url := strings.TrimSpace(fnsCheckURL)
	if url == "" {
		url = CheckURL(QRParams{
			FiscalDriveNumber:    p.FiscalDriveNumber,
			FiscalDocumentNumber: p.FiscalDocumentNumber,
			FiscalSign:           p.FiscalSign,
			TotalSumKopecks:      p.TotalSum,
			DateTime:             dt,
		})
	}
	return ParsedReceipt{
		FiscalDriveNumber:    p.FiscalDriveNumber,
		FiscalDocumentNumber: p.FiscalDocumentNumber,
		FiscalSign:           p.FiscalSign,
		DateTime:             dt,
		TotalSum:             p.TotalSum,
		StoreName:            p.StoreName,
		UserINN:              p.UserINN,
		RetailPlace:          p.RetailPlace,
		Items:                p.Items,
		FNSCheckURL:          url,
	}, nil
}

func FormatItemSum(kopecks int64) string {
	return strconv.FormatFloat(float64(kopecks)/100, 'f', 2, 64)
}
