// Command receipt-services-probe surveys free/low-cost receipt lookup endpoints.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	fn     = "7384440900988131"
	fd     = "6614"
	fp     = "0538715847"
	qr     = "t=20260706T1753&s=4047.50&fn=7384440900988131&i=6614&fp=0538715847&n=1"
	sumRub = "4047.50"
	date   = "2026-07-06T17:53:00"
)

type row struct {
	Service    string `json:"service"`
	Reachable  bool   `json:"reachable"`
	HasItems   bool   `json:"has_items"`
	AuthNeeded string `json:"auth_needed"`
	FreeTier   string `json:"free_tier"`
	Status     int    `json:"status,omitempty"`
	Latency    string `json:"latency"`
	Notes      string `json:"notes"`
}

func main() {
	c := &http.Client{Timeout: 12 * time.Second}
	var out []row

	out = append(out, probe(c, "ФНС proverkacheka :9999 GET", func() row {
		u := fmt.Sprintf("https://proverkacheka.nalog.ru:9999/v1/inns/*/kkts/*/fss/%s/tickets/%s?fiscalSign=%s&sendToEmail=no", fn, fd, url.QueryEscape(fp))
		return hitGET(u, map[string]string{"Device-OS": "Android", "Accept": "application/json"})
	}, "Нет", "Публичный (неофициальный)", "Позиции да; с многих серверов timeout"))

	out = append(out, probe(c, "ФНС kkt-online openapikkt", func() row {
		form := url.Values{"type": {"request"}, "date": {"06.07.2026"}, "time": {"17:53"}, "operationtype": {"1"}, "summ": {"4047,5"}, "fn": {fn}, "fd": {fd}, "fp": {fp}}
		r := hitPOST("https://kkt-online.nalog.ru/openapikkt.html", form.Encode(), kktHeaders())
		if !r.Reachable {
			return r
		}
		var created struct {
			Success       bool
			UserToken, ID string
		}
		_ = json.Unmarshal([]byte(r.Notes), &created)
		if !created.Success {
			r.Notes = "request failed: " + trunc(r.Notes)
			return r
		}
		for i := 0; i < 6; i++ {
			time.Sleep(400 * time.Millisecond)
			poll := url.Values{"type": {"poll"}, "UserToken": {created.UserToken}, "id": {created.ID}}
			pr := hitPOST("https://kkt-online.nalog.ru/openapikkt.html", poll.Encode(), kktHeaders())
			if strings.Contains(pr.Notes, `"code":"200"`) {
				r.HasItems = false
				r.Notes = "validation only, no items"
				return r
			}
		}
		r.Notes = "poll timeout"
		return r
	}, "Нет", "Бесплатно (официальный)", "Только факт подлинности"))

	out = append(out, probe(c, "ФНС irkkt-mobile v2/check (no auth)", func() row {
		q := url.Values{"fsId": {fn}, "operationType": {"1"}, "documentId": {fd}, "fiscalSign": {fp}, "date": {date}, "sum": {sumRub}}
		r := hitGET("https://irkkt-mobile.nalog.ru:8888/v2/check/ticket?"+q.Encode(), mobileHeaders(""))
		if r.Status == 204 {
			r.Notes = "valid receipt, no items"
		}
		return r
	}, "Нет", "Бесплатно (моб. API)", "Проверка без позиций; полный чек — sessionId"))

	out = append(out, probe(c, "ФНС irkkt-mobile v2/ticket", func() row {
		body, _ := json.Marshal(map[string]string{"qr": qr})
		return hitPOSTJSON("https://irkkt-mobile.nalog.ru:8888/v2/ticket", body, mobileHeaders(""))
	}, "sessionId (SMS)", "Бесплатно", "Позиции да; нужна сессия приложения"))

	out = append(out, probe(c, "ФНС openapi.nalog.ru:8090", func() row {
		return hitGET("https://openapi.nalog.ru:8090/", nil)
	}, "Мастер-токен (заявка ИП/ООО)", "Бесплатно после регистрации", "Официальный SOAP/REST; позиции да"))

	out = append(out, probe(c, "OFD.ru GetReceipt (no token)", func() row {
		body := fmt.Sprintf(`{"TotalSum":404750,"DocDateTime":"%s","FnNumber":"%s","ReceiptOperationType":1,"DocNumber":"%s","DocFiscalSign":"%s","tokenSecret":"test"}`, date, fn, fd, fp)
		return hitPOSTJSON("https://ofd.ru/api/partner/v3/receipts/GetReceipt", []byte(body), map[string]string{"Content-Type": "application/json"})
	}, "tokenSecret (платная подписка)", "Тестовый период по заявке", "Позиции да; платный QR Cash"))

	out = append(out, probe(c, "check.ofd.ru web", func() row {
		return hitGET(fmt.Sprintf("https://check.ofd.ru/recipient/check?fn=%s&fp=%s&fd=%s", fn, fp, fd), map[string]string{"User-Agent": "Mozilla/5.0"})
	}, "Нет (веб)", "Бесплатно вручную", "Веб-форма, не API"))

	out = append(out, probe(c, "consumer.1-ofd.ru", func() row {
		return hitGET("https://consumer.1-ofd.ru/", map[string]string{"User-Agent": "Mozilla/5.0"})
	}, "Веб/кабинет", "Бесплатно для своих чеков", "1-OFD, не универсальный API"))

	out = append(out, probe(c, "proverkacheka.com /check/get (legacy)", func() row {
		form := url.Values{"fn": {fn}, "fd": {fd}, "fp": {fp}, "n": {"1"}, "s": {sumRub}, "t": {"06.07.2026 17:53"}, "qr": {"0"}}
		return hitPOST("https://proverkacheka.com/check/get", form.Encode(), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"User-Agent":   "Mozilla/5.0",
		})
	}, "Не рекомендуется", "Без токена (legacy)", "Сторонний; без токена может меняться"))

	out = append(out, probe(c, "proverkacheka.com /api/v1/check/get", func() row {
		body, _ := json.Marshal(map[string]string{"token": "", "qrraw": qr})
		return hitPOSTJSON("https://proverkacheka.com/api/v1/check/get", body, map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "Mozilla/5.0",
		})
	}, "token (регистрация)", "~15 запросов/сутки бесплатно", "Позиции да; сторонний прокси к ФНС"))

	out = append(out, probe(c, "platformaofd.ru check", func() row {
		return hitGET("https://lk.platformaofd.ru/web/noauth/cheque/search?fn="+fn+"&fp="+fp+"&i="+fd, map[string]string{"User-Agent": "Mozilla/5.0"})
	}, "Веб", "Бесплатно вручную", "Платформа ОФД, веб"))

	out = append(out, probe(c, "taxcom.ru check", func() row {
		return hitGET("https://receipt.taxcom.ru/", map[string]string{"User-Agent": "Mozilla/5.0"})
	}, "Веб", "Бесплатно вручную", "Такском ОФД"))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func probe(c *http.Client, name string, fn func() row, auth, free, hint string) row {
	start := time.Now()
	r := fn()
	r.Service = name
	r.AuthNeeded = auth
	r.FreeTier = free
	if r.Notes == "" {
		r.Notes = hint
	}
	r.Latency = time.Since(start).Round(time.Millisecond).String()
	return r
}

func hitGET(u string, hdr map[string]string) row {
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	return do(req)
}

func hitPOST(u, body string, hdr map[string]string) row {
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	return do(req)
}

func hitPOSTJSON(u string, body []byte, hdr map[string]string) row {
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	return do(req)
}

func do(req *http.Request) row {
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return row{Reachable: false, Notes: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	s := string(b)
	items := countItems(b)
	return row{
		Reachable: resp.StatusCode > 0,
		HasItems:  items > 0,
		Status:    resp.StatusCode,
		Notes:     trunc(s),
		Latency:   time.Since(start).String(),
	}
}

func kktHeaders() map[string]string {
	return map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Origin":           "https://kkt-online.nalog.ru",
		"Referer":          "https://kkt-online.nalog.ru/",
		"X-Requested-With": "XMLHttpRequest",
		"User-Agent":       "Mozilla/5.0",
	}
}

func mobileHeaders(session string) map[string]string {
	h := map[string]string{
		"Accept":        "application/json",
		"Device-OS":     "Android",
		"Device-Id":     "",
		"clientVersion": "2.9.0",
		"User-Agent":    "billchecker/2.9.0 (Android)",
	}
	if session != "" {
		h["sessionId"] = session
	}
	return h
}

func countItems(b []byte) int {
	var any map[string]any
	if json.Unmarshal(b, &any) != nil {
		return 0
	}
	return walkItems(any)
}

func walkItems(v any) int {
	switch t := v.(type) {
	case map[string]any:
		if items, ok := t["items"].([]any); ok {
			return len(items)
		}
		n := 0
		for _, c := range t {
			n += walkItems(c)
		}
		return n
	case []any:
		n := 0
		for _, c := range t {
			n += walkItems(c)
		}
		return n
	default:
		return 0
	}
}

func trunc(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 180 {
		return s[:180] + "..."
	}
	return s
}
