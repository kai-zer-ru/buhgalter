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

const defaultMaxOfficialBaseURL = "https://platform-api2.max.ru"

type MaxOfficialNotifier struct {
	Token       string
	RecipientID string
	BaseURL     string
	APIVersion  string
	Client      *http.Client
}

func (n *MaxOfficialNotifier) ValidateConfig() error {
	if strings.TrimSpace(n.Token) == "" || strings.TrimSpace(n.RecipientID) == "" {
		return ErrInvalidConfig
	}
	return nil
}

func (n *MaxOfficialNotifier) Send(ctx context.Context, recipient string, text string) error {
	if err := n.ValidateConfig(); err != nil {
		return err
	}
	target := strings.TrimSpace(recipient)
	if target == "" {
		target = n.RecipientID
	}
	baseURL := strings.TrimSuffix(n.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimSuffix(os.Getenv("BUHGALTER_MAX_OFFICIAL_BASE_URL"), "/")
	}
	if baseURL == "" {
		baseURL = defaultMaxOfficialBaseURL
	}

	apiVersion := strings.TrimSpace(n.APIVersion)
	if apiVersion == "" {
		apiVersion = strings.TrimSpace(os.Getenv("BUHGALTER_MAX_OFFICIAL_API_VERSION"))
	}
	if apiVersion == "" {
		apiVersion = "1.2.5"
	}

	endpoint, err := buildMaxOfficialMessageEndpoint(baseURL, target, apiVersion)
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
		return fmt.Errorf("max official send failed: status=%d", resp.StatusCode)
	}
	return nil
}

func buildMaxOfficialMessageEndpoint(baseURL, recipientID, apiVersion string) (string, error) {
	rid := strings.TrimSpace(recipientID)
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
	version := strings.TrimSpace(apiVersion)
	if version == "" {
		version = "1.2.5"
	}
	query.Set("v", version)

	return strings.TrimSuffix(baseURL, "/") + "/messages?" + query.Encode(), nil
}
