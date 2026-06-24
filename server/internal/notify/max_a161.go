package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type MaxA161Notifier struct {
	Token   string
	UserID  string
	BaseURL string
	Client  *http.Client
}

func (n *MaxA161Notifier) ValidateConfig() error {
	token := strings.TrimSpace(n.Token)
	if token == "" || len(token) < 16 {
		return ErrInvalidConfig
	}
	if strings.TrimSpace(n.UserID) == "" {
		return ErrInvalidConfig
	}
	return nil
}

func (n *MaxA161Notifier) Send(ctx context.Context, recipient string, text string) error {
	if err := n.ValidateConfig(); err != nil {
		return err
	}
	target := strings.TrimSpace(recipient)
	if target == "" {
		target = n.UserID
	}
	baseURL := strings.TrimSuffix(n.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimSuffix(os.Getenv("BUHGALTER_MAX_A161_BASE_URL"), "/")
	}
	if baseURL == "" {
		baseURL = "https://notify.a161.ru"
	}
	endpoint, err := buildMaxA161MessageEndpoint(baseURL, target)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", n.Token)
	req.Header.Set("Content-Type", "application/json")
	client := n.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("max a161 send failed: status=%d", resp.StatusCode)
	}
	return nil
}

func buildMaxA161MessageEndpoint(baseURL, recipient string) (string, error) {
	rid := strings.TrimSpace(recipient)
	if rid == "" {
		return "", ErrInvalidConfig
	}
	parsedID, err := strconv.ParseInt(rid, 10, 64)
	if err != nil || parsedID == 0 {
		return "", ErrInvalidConfig
	}

	query := url.Values{}
	if parsedID < 0 {
		query.Set("chat_id", rid)
	} else {
		query.Set("user_id", rid)
	}
	return strings.TrimSuffix(baseURL, "/") + "/messages?" + query.Encode(), nil
}
