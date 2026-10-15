package notify

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxTGCallbackDataBytes = 64
	maxTGButtonTextLen     = 64
	maxTGURLLen            = 2048
	maxTGButtons           = 100
	maxTGButtonsPerRow     = 8

	maxMAXButtons       = 210
	maxMAXRows          = 30
	maxMAXButtonsPerRow = 7
	maxMAXSpecialPerRow = 3 // link, open_app, request_geo_location, request_contact
	maxMAXURLLen        = 2048
	maxMAXPayloadLen    = 1024
)

var telegramButtonTypes = map[string]struct{}{
	"url":                              {},
	"callback_data":                    {},
	"web_app":                          {},
	"login_url":                        {},
	"switch_inline_query":              {},
	"switch_inline_query_current_chat": {},
	"switch_inline_query_chosen_chat":  {},
	"callback_game":                    {},
	"pay":                              {},
	"copy_text":                        {},
}

var maxButtonTypes = map[string]struct{}{
	"callback":             {},
	"link":                 {},
	"message":              {},
	"open_app":             {},
	"request_contact":      {},
	"request_geo_location": {},
	"clipboard":            {},
}

func ValidateTelegramKeyboard(kb InlineKeyboard) error {
	if kb.IsEmpty() {
		return nil
	}
	if kb.ButtonCount() > maxTGButtons {
		return fmt.Errorf("telegram keyboard: max %d buttons", maxTGButtons)
	}
	for i, row := range kb.Rows {
		if len(row) > maxTGButtonsPerRow {
			return fmt.Errorf("telegram keyboard: row %d has more than %d buttons", i, maxTGButtonsPerRow)
		}
		for j, btn := range row {
			if err := validateTelegramButton(btn); err != nil {
				return fmt.Errorf("telegram keyboard: row %d button %d: %w", i, j, err)
			}
		}
	}
	return nil
}

func ValidateMaxKeyboard(kb InlineKeyboard) error {
	if kb.IsEmpty() {
		return nil
	}
	if len(kb.Rows) > maxMAXRows {
		return fmt.Errorf("max keyboard: max %d rows", maxMAXRows)
	}
	if kb.ButtonCount() > maxMAXButtons {
		return fmt.Errorf("max keyboard: max %d buttons", maxMAXButtons)
	}
	for i, row := range kb.Rows {
		if len(row) > maxMAXButtonsPerRow {
			return fmt.Errorf("max keyboard: row %d has more than %d buttons", i, maxMAXButtonsPerRow)
		}
		special := 0
		for j, btn := range row {
			if err := validateMaxButton(btn); err != nil {
				return fmt.Errorf("max keyboard: row %d button %d: %w", i, j, err)
			}
			if isMaxSpecialRowType(btn.Type) {
				special++
			}
		}
		// MAX: rows that include link/open_app/request_* are limited to 3 buttons total.
		if special > 0 && len(row) > maxMAXSpecialPerRow {
			return fmt.Errorf("max keyboard: row %d with link/open_app/request_* may have at most %d buttons", i, maxMAXSpecialPerRow)
		}
	}
	return nil
}

func isMaxSpecialRowType(t string) bool {
	switch t {
	case "link", "open_app", "request_contact", "request_geo_location":
		return true
	default:
		return false
	}
}

func validateTelegramButton(btn KeyboardButton) error {
	text := strings.TrimSpace(btn.Text)
	if text == "" {
		return fmt.Errorf("text is required")
	}
	if utf8.RuneCountInString(text) > maxTGButtonTextLen {
		return fmt.Errorf("text max length is %d", maxTGButtonTextLen)
	}
	if _, ok := telegramButtonTypes[btn.Type]; !ok {
		return fmt.Errorf("unknown type %q", btn.Type)
	}
	switch btn.Type {
	case "url":
		if strings.TrimSpace(btn.URL) == "" {
			return fmt.Errorf("url is required")
		}
		if len(btn.URL) > maxTGURLLen {
			return fmt.Errorf("url max length is %d", maxTGURLLen)
		}
	case "callback_data":
		if btn.CallbackData == "" {
			return fmt.Errorf("callback_data is required")
		}
		if len(btn.CallbackData) > maxTGCallbackDataBytes {
			return fmt.Errorf("callback_data max %d bytes", maxTGCallbackDataBytes)
		}
	case "web_app":
		if btn.WebApp == nil || strings.TrimSpace(btn.WebApp.URL) == "" {
			return fmt.Errorf("web_app.url is required")
		}
		if len(btn.WebApp.URL) > maxTGURLLen {
			return fmt.Errorf("web_app.url max length is %d", maxTGURLLen)
		}
	case "login_url":
		if btn.LoginURL == nil || strings.TrimSpace(btn.LoginURL.URL) == "" {
			return fmt.Errorf("login_url.url is required")
		}
		if len(btn.LoginURL.URL) > maxTGURLLen {
			return fmt.Errorf("login_url.url max length is %d", maxTGURLLen)
		}
	case "copy_text":
		if btn.CopyText == nil || strings.TrimSpace(btn.CopyText.Text) == "" {
			return fmt.Errorf("copy_text.text is required")
		}
	case "switch_inline_query", "switch_inline_query_current_chat":
		// query may be empty string
	case "switch_inline_query_chosen_chat", "callback_game", "pay":
		// no extra required fields
	}
	return nil
}

func validateMaxButton(btn KeyboardButton) error {
	text := strings.TrimSpace(btn.Text)
	if text == "" {
		return fmt.Errorf("text is required")
	}
	if _, ok := maxButtonTypes[btn.Type]; !ok {
		return fmt.Errorf("unknown type %q", btn.Type)
	}
	switch btn.Type {
	case "link", "open_app":
		if strings.TrimSpace(btn.URL) == "" {
			return fmt.Errorf("url is required")
		}
		if len(btn.URL) > maxMAXURLLen {
			return fmt.Errorf("url max length is %d", maxMAXURLLen)
		}
	case "callback", "clipboard":
		if strings.TrimSpace(btn.Payload) == "" {
			return fmt.Errorf("payload is required")
		}
		if len(btn.Payload) > maxMAXPayloadLen {
			return fmt.Errorf("payload max length is %d", maxMAXPayloadLen)
		}
	case "message":
		// payload optional: if set, length-check
		if len(btn.Payload) > maxMAXPayloadLen {
			return fmt.Errorf("payload max length is %d", maxMAXPayloadLen)
		}
	}
	return nil
}
