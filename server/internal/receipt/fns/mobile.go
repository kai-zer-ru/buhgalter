package fns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const mobileHost = "https://irkkt-mobile.nalog.ru:8888"

func (c *Client) fetchMobile(ctx context.Context, qr string, p QRParams) (ParsedReceipt, error) {
	if strings.TrimSpace(c.SessionID) == "" {
		return ParsedReceipt{}, ErrFetchFailed
	}
	client := c.http()

	body, _ := json.Marshal(map[string]string{"qr": qr})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mobileHost+"/v2/ticket", bytes.NewReader(body))
	if err != nil {
		return ParsedReceipt{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setMobileAppHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return ParsedReceipt{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return ParsedReceipt{}, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ParsedReceipt{}, fmt.Errorf("%w: mobile unauthorized", ErrFetchFailed)
	}
	if resp.StatusCode != http.StatusOK {
		return ParsedReceipt{}, fmt.Errorf("%w: mobile status %d", ErrFetchFailed, resp.StatusCode)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &created); err != nil || created.ID == "" {
		return ParsedReceipt{}, fmt.Errorf("%w: mobile ticket id", ErrFetchFailed)
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return ParsedReceipt{}, err
		}
		time.Sleep(600 * time.Millisecond)

		treq, err := http.NewRequestWithContext(ctx, http.MethodGet, mobileHost+"/v2/tickets/"+created.ID, nil)
		if err != nil {
			return ParsedReceipt{}, err
		}
		c.setMobileAppHeaders(treq)

		tresp, err := client.Do(treq)
		if err != nil {
			return ParsedReceipt{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		tdata, _ := io.ReadAll(io.LimitReader(tresp.Body, 2<<20))
		tresp.Body.Close()

		if tresp.StatusCode == http.StatusNotFound {
			return ParsedReceipt{}, ErrNotFound
		}
		if tresp.StatusCode != http.StatusOK {
			continue
		}

		var ticket mobileTicketResponse
		if err := json.Unmarshal(tdata, &ticket); err != nil {
			continue
		}
		switch ticket.Status {
		case 2:
			return mapMobileTicket(p, ticket)
		case 8:
			return ParsedReceipt{}, ErrNotFound
		}
	}
	return ParsedReceipt{}, fmt.Errorf("%w: mobile poll timeout", ErrUnavailable)
}

type mobileTicketResponse struct {
	Status       int `json:"status"`
	Organization struct {
		Name string `json:"name"`
	} `json:"organization"`
	Seller struct {
		Name string `json:"name"`
	} `json:"seller"`
	Ticket struct {
		Document struct {
			Receipt receiptJSON `json:"receipt"`
		} `json:"document"`
	} `json:"ticket"`
}

func mapMobileTicket(p QRParams, t mobileTicketResponse) (ParsedReceipt, error) {
	rcpt := t.Ticket.Document.Receipt
	if rcpt.TotalSum <= 0 && len(rcpt.Items) == 0 {
		return ParsedReceipt{}, fmt.Errorf("%w: empty mobile receipt", ErrFetchFailed)
	}
	out, err := mapReceiptJSON(p, rcpt)
	if err != nil {
		return ParsedReceipt{}, err
	}
	if out.StoreName == "" {
		out.StoreName = strings.TrimSpace(t.Organization.Name)
	}
	if out.StoreName == "" {
		out.StoreName = strings.TrimSpace(t.Seller.Name)
	}
	return out, nil
}

func (c *Client) setMobileAppHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Device-OS", "Android")
	req.Header.Set("Device-Id", "")
	req.Header.Set("clientVersion", "2.9.0")
	req.Header.Set("Accept-Language", "ru-RU;q=1")
	req.Header.Set("User-Agent", "billchecker/2.9.0 (Android)")
	if sid := strings.TrimSpace(c.SessionID); sid != "" {
		req.Header.Set("sessionId", sid)
	}
}
