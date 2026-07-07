// Command fns-probe tests FNS receipt API endpoints from the current host.
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
	qr         = "t=20260706T1753&s=4047.50&fn=7384440900988131&i=6614&fp=0538715847&n=1"
	fn         = "7384440900988131"
	fd         = "6614"
	fp         = "0538715847"
	sumKopecks = "404750"
	sumRub     = "4047.50"
	dateMSK    = "2026-07-06T17:53:00"
)

type result struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Latency string `json:"latency"`
	Status  int    `json:"status,omitempty"`
	Items   int    `json:"items,omitempty"`
	Store   string `json:"store,omitempty"`
	Error   string `json:"error,omitempty"`
	Preview string `json:"preview,omitempty"`
}

func main() {
	client := &http.Client{Timeout: 12 * time.Second}
	var out []result

	out = append(out, probe(client, "host:9999 GET ticket", probeMobileGET))
	out = append(out, probe(client, "host:9999 captcha+POST", probeCaptchaPOST))
	out = append(out, probe(client, "host:9999 ofds check", probeOfdsCheck))
	out = append(out, probe(client, "kkt-online request+poll", probeKktOnline))
	out = append(out, probe(client, "irkkt-mobile:8888 GET ticket", probeIrkktGET))
	out = append(out, probe(client, "irkkt-mobile:8888 POST ticket+qr", probeIrkktPOSTTicket))
	out = append(out, probe(client, "irkkt-mobile:8888 captcha", probeIrkktCaptcha))
	out = append(out, probe(client, "openapi:8090 root", probeOpenAPIRoot))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func probe(client *http.Client, name string, fn func(*http.Client) result) result {
	r := fn(client)
	r.Name = name
	return r
}

func probeMobileGET(client *http.Client) result {
	start := time.Now()
	u := fmt.Sprintf("https://proverkacheka.nalog.ru:9999/v1/inns/*/kkts/*/fss/%s/tickets/%s?fiscalSign=%s&sendToEmail=no", fn, fd, url.QueryEscape(fp))
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	setMobile(req)
	return doJSON(client, start, req)
}

func probeCaptchaPOST(client *http.Client) result {
	start := time.Now()
	req1, _ := http.NewRequest(http.MethodGet, "https://proverkacheka.nalog.ru:9999/v1/captcha", nil)
	setMobile(req1)
	resp1, err := client.Do(req1)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp1.Body.Close()
	body1, _ := io.ReadAll(resp1.Body)
	var cap struct {
		CaptchaToken string `json:"captchaToken"`
	}
	_ = json.Unmarshal(body1, &cap)
	if cap.CaptchaToken == "" {
		return result{Status: resp1.StatusCode, Error: "no captcha token", Preview: trunc(string(body1)), Latency: time.Since(start).String()}
	}
	q := url.Values{}
	q.Set("fiscalDriveNumber", fn)
	q.Set("fiscalDocumentNumber", fd)
	q.Set("fiscalSign", fp)
	q.Set("date", dateMSK)
	q.Set("sum", sumRub)
	q.Set("captchaToken", cap.CaptchaToken)
	q.Set("captchaCode", "1")
	endpoint := "https://proverkacheka.nalog.ru:9999/v1/inns/*/kkts/*/fiscal-docs/*/fiscal-drives/*/fiscal-docs/*/tickets?" + q.Encode()
	req2, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	req2.Header.Set("Content-Type", "application/json")
	setMobile(req2)
	return doJSON(client, start, req2)
}

func probeOfdsCheck(client *http.Client) result {
	start := time.Now()
	q := url.Values{}
	q.Set("fiscalSign", fp)
	q.Set("date", dateMSK)
	q.Set("sum", sumKopecks)
	u := fmt.Sprintf("https://proverkacheka.nalog.ru:9999/v1/ofds/*/inns/*/fss/%s/operations/1/tickets/%s?%s", fn, fd, q.Encode())
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	setMobile(req)
	resp, err := client.Do(req)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
	return result{
		OK:      resp.StatusCode == 204 || resp.StatusCode == 406,
		Status:  resp.StatusCode,
		Preview: trunc(string(b)),
		Latency: time.Since(start).String(),
	}
}

func probeKktOnline(client *http.Client) result {
	start := time.Now()
	form := url.Values{}
	form.Set("type", "request")
	form.Set("date", "06.07.2026")
	form.Set("time", "17:53")
	form.Set("operationtype", "1")
	form.Set("summ", "4047,5")
	form.Set("fn", fn)
	form.Set("fd", fd)
	form.Set("fp", fp)
	req, _ := http.NewRequest(http.MethodPost, "https://kkt-online.nalog.ru/openapikkt.html", strings.NewReader(form.Encode()))
	setKkt(req)
	resp, err := client.Do(req)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var created struct {
		Success   bool   `json:"success"`
		UserToken string `json:"UserToken"`
		ID        string `json:"ID"`
	}
	_ = json.Unmarshal(b, &created)
	if !created.Success {
		return result{Status: resp.StatusCode, Error: "request failed", Preview: trunc(string(b)), Latency: time.Since(start).String()}
	}
	for i := 0; i < 8; i++ {
		time.Sleep(500 * time.Millisecond)
		poll := url.Values{}
		poll.Set("type", "poll")
		poll.Set("UserToken", created.UserToken)
		poll.Set("id", created.ID)
		preq, _ := http.NewRequest(http.MethodPost, "https://kkt-online.nalog.ru/openapikkt.html", strings.NewReader(poll.Encode()))
		setKkt(preq)
		presp, err := client.Do(preq)
		if err != nil {
			return result{Error: err.Error(), Latency: time.Since(start).String()}
		}
		pb, _ := io.ReadAll(presp.Body)
		presp.Body.Close()
		if strings.Contains(string(pb), `"code":"200"`) {
			return result{OK: true, Status: 200, Preview: "validation only, no items", Latency: time.Since(start).String()}
		}
		if !strings.Contains(string(pb), `"code":"202"`) {
			return result{Status: presp.StatusCode, Preview: trunc(string(pb)), Latency: time.Since(start).String()}
		}
	}
	return result{Error: "poll timeout", Latency: time.Since(start).String()}
}

func probeIrkktGET(client *http.Client) result {
	start := time.Now()
	u := fmt.Sprintf("https://irkkt-mobile.nalog.ru:8888/v1/inns/*/kkts/*/fss/%s/tickets/%s?fiscalSign=%s&sendToEmail=no", fn, fd, url.QueryEscape(fp))
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	setMobile(req)
	return doJSON(client, start, req)
}

func probeIrkktPOSTTicket(client *http.Client) result {
	start := time.Now()
	body, _ := json.Marshal(map[string]string{"qr": qr})
	req, _ := http.NewRequest(http.MethodPost, "https://irkkt-mobile.nalog.ru:8888/v2/ticket", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	setMobile(req)
	return doJSON(client, start, req)
}

func probeIrkktCaptcha(client *http.Client) result {
	start := time.Now()
	req, _ := http.NewRequest(http.MethodGet, "https://irkkt-mobile.nalog.ru:8888/v2/captcha", nil)
	setMobile(req)
	resp, err := client.Do(req)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
	return result{Status: resp.StatusCode, Preview: trunc(string(b)), Latency: time.Since(start).String()}
}

func probeOpenAPIRoot(client *http.Client) result {
	start := time.Now()
	req, _ := http.NewRequest(http.MethodGet, "https://openapi.nalog.ru:8090/", nil)
	resp, err := client.Do(req)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	return result{OK: true, Status: resp.StatusCode, Latency: time.Since(start).String()}
}

func doJSON(client *http.Client, start time.Time, req *http.Request) result {
	resp, err := client.Do(req)
	if err != nil {
		return result{Error: err.Error(), Latency: time.Since(start).String()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	items, store := countItems(b)
	ok := items > 0 || (store != "" && resp.StatusCode == 200)
	return result{
		OK:      ok,
		Status:  resp.StatusCode,
		Items:   items,
		Store:   store,
		Preview: trunc(string(b)),
		Latency: time.Since(start).String(),
	}
}

func countItems(b []byte) (int, string) {
	var any map[string]any
	if json.Unmarshal(b, &any) != nil {
		return 0, ""
	}
	return walkItems(any), walkStore(any)
}

func walkItems(v any) int {
	switch t := v.(type) {
	case map[string]any:
		if items, ok := t["items"].([]any); ok {
			return len(items)
		}
		n := 0
		for _, child := range t {
			n += walkItems(child)
		}
		return n
	case []any:
		n := 0
		for _, child := range t {
			n += walkItems(child)
		}
		return n
	default:
		return 0
	}
}

func walkStore(v any) string {
	switch t := v.(type) {
	case map[string]any:
		for _, key := range []string{"retailPlace", "user", "store_name", "StoreName"} {
			if s, ok := t[key].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
		for _, child := range t {
			if s := walkStore(child); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range t {
			if s := walkStore(child); s != "" {
				return s
			}
		}
	}
	return ""
}

func setMobile(req *http.Request) {
	req.Header.Set("Device-Id", "")
	req.Header.Set("Device-OS", "Android")
	req.Header.Set("Accept", "application/json")
}

func setKkt(req *http.Request) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://kkt-online.nalog.ru")
	req.Header.Set("Referer", "https://kkt-online.nalog.ru/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
}

func trunc(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 220 {
		return s[:220] + "..."
	}
	return s
}
