package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type TelegramNotifier struct {
	Token   string
	ChatID  string
	BaseURL string
	Client  *http.Client
}

func (n *TelegramNotifier) ValidateConfig() error {
	if strings.TrimSpace(n.Token) == "" || strings.TrimSpace(n.ChatID) == "" {
		return ErrInvalidConfig
	}
	return nil
}

func (n *TelegramNotifier) Send(ctx context.Context, recipient string, text string) error {
	if err := n.ValidateConfig(); err != nil {
		return err
	}
	if strings.TrimSpace(recipient) == "" {
		recipient = n.ChatID
	}
	baseURL := strings.TrimSuffix(n.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimSuffix(os.Getenv("BUHGALTER_TELEGRAM_BASE_URL"), "/")
	}
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	payload, _ := json.Marshal(map[string]string{
		"chat_id": recipient,
		"text":    text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/sendMessage", baseURL, n.Token), bytes.NewReader(payload))
	if err != nil {
		return err
	}
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
		return fmt.Errorf("telegram send failed: status=%d", resp.StatusCode)
	}
	return nil
}
