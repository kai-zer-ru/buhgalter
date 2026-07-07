package fns

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidQR = errors.New("invalid receipt QR")

type QRParams struct {
	DateTime             time.Time
	TotalSumKopecks      int64
	FiscalDriveNumber    string
	FiscalDocumentNumber string
	FiscalSign           string
	OperationType        string
}

func moscowLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("MSK", 3*3600)
	}
	return loc
}

func ParseQR(raw string) (QRParams, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return QRParams{}, ErrInvalidQR
	}
	values := url.Values{}
	if strings.Contains(raw, "?") {
		u, err := url.Parse(raw)
		if err != nil {
			return QRParams{}, ErrInvalidQR
		}
		values = u.Query()
	} else if strings.Contains(raw, "=") {
		var err error
		values, err = url.ParseQuery(strings.TrimPrefix(raw, "?"))
		if err != nil {
			return QRParams{}, ErrInvalidQR
		}
	} else {
		return QRParams{}, ErrInvalidQR
	}

	t := strings.TrimSpace(values.Get("t"))
	fn := strings.TrimSpace(values.Get("fn"))
	i := strings.TrimSpace(values.Get("i"))
	fp := strings.TrimSpace(values.Get("fp"))
	s := strings.TrimSpace(values.Get("s"))
	n := strings.TrimSpace(values.Get("n"))
	if t == "" || fn == "" || i == "" || fp == "" || s == "" {
		return QRParams{}, ErrInvalidQR
	}
	if n == "" {
		n = "1"
	}

	dt, err := parseQRDateTime(t)
	if err != nil {
		return QRParams{}, err
	}
	sumKopecks, err := parseQRSum(s)
	if err != nil {
		return QRParams{}, err
	}

	return QRParams{
		DateTime:             dt,
		TotalSumKopecks:      sumKopecks,
		FiscalDriveNumber:    fn,
		FiscalDocumentNumber: i,
		FiscalSign:           fp,
		OperationType:        n,
	}, nil
}

func parseQRDateTime(v string) (time.Time, error) {
	layouts := []string{"20060102T1504", "20060102T150405"}
	loc := moscowLocation()
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, v, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: bad datetime", ErrInvalidQR)
}

func parseQRSum(v string) (int64, error) {
	v = strings.ReplaceAll(v, ",", ".")
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("%w: bad sum", ErrInvalidQR)
	}
	return int64(f*100 + 0.5), nil
}

// FNSDate formats receipt datetime for FNS API (local MSK, no offset).
func FNSDate(t time.Time) string {
	return t.In(moscowLocation()).Format("2006-01-02T15:04:05")
}

func CheckURL(p QRParams) string {
	sum := fmt.Sprintf("%.2f", float64(p.TotalSumKopecks)/100)
	t := p.DateTime.In(moscowLocation()).Format("20060102T1504")
	q := url.Values{}
	q.Set("fn", p.FiscalDriveNumber)
	q.Set("i", p.FiscalDocumentNumber)
	q.Set("fp", p.FiscalSign)
	q.Set("n", p.OperationType)
	q.Set("s", sum)
	q.Set("t", t)
	return "https://proverkacheka.nalog.ru/?" + q.Encode()
}
