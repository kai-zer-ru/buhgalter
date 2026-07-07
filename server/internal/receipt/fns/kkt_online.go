package fns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const kktOnlineURL = "https://kkt-online.nalog.ru/openapikkt.html"

func (c *Client) validateKktOnline(ctx context.Context, p QRParams) error {
	client := c.http()
	sum := formatKktSum(p.TotalSumKopecks)
	date := p.DateTime.In(moscowLocation()).Format("02.01.2006")
	tm := p.DateTime.In(moscowLocation()).Format("15:04")

	form := url.Values{}
	form.Set("type", "request")
	form.Set("date", date)
	form.Set("time", tm)
	form.Set("operationtype", p.OperationType)
	form.Set("summ", sum)
	form.Set("fn", p.FiscalDriveNumber)
	form.Set("fd", p.FiscalDocumentNumber)
	form.Set("fp", p.FiscalSign)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kktOnlineURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	setKktOnlineHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return err
	}
	var created struct {
		Success   bool   `json:"success"`
		UserToken string `json:"UserToken"`
		ID        string `json:"ID"`
	}
	if err := json.Unmarshal(body, &created); err != nil || !created.Success {
		return fmt.Errorf("%w: kkt-online request rejected", ErrFetchFailed)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)

		poll := url.Values{}
		poll.Set("type", "poll")
		poll.Set("UserToken", created.UserToken)
		poll.Set("id", created.ID)
		preq, err := http.NewRequestWithContext(ctx, http.MethodPost, kktOnlineURL, strings.NewReader(poll.Encode()))
		if err != nil {
			return err
		}
		setKktOnlineHeaders(preq)

		presp, err := client.Do(preq)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		pb, _ := io.ReadAll(io.LimitReader(presp.Body, 16<<10))
		presp.Body.Close()

		if strings.Contains(string(pb), `"code":"200"`) {
			return nil
		}
		if !strings.Contains(string(pb), `"code":"202"`) {
			return fmt.Errorf("%w: kkt-online poll %s", ErrNotFound, strings.TrimSpace(string(pb)))
		}
	}
	return fmt.Errorf("%w: kkt-online poll timeout", ErrUnavailable)
}

func formatKktSum(kopecks int64) string {
	rub := float64(kopecks) / 100
	s := fmt.Sprintf("%.2f", rub)
	return strings.ReplaceAll(s, ".", ",")
}

func setKktOnlineHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://kkt-online.nalog.ru")
	req.Header.Set("Referer", "https://kkt-online.nalog.ru/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
}
