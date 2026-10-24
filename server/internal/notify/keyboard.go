package notify

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const emptyKeyboardJSON = `{"rows":[]}`

// InlineKeyboard is the stored/API shape for TG and MAX keyboards.
// Example: {"rows":[[{"type":"url","text":"Открыть","url":"{debt_url}"}]]}
// Placement is template-only: tokens "g:<id>" and "l:<id>". Absent → global rows, then local rows.
type InlineKeyboard struct {
	Rows      [][]KeyboardButton `json:"rows"`
	Placement [][]string         `json:"placement,omitempty"`
}

// KeyboardButton is a provider-agnostic button. Exactly one action type is set
// via Type; provider-specific fields are filled accordingly.
// ID is stored so a template can refer to a global button. Provider payloads omit it.
type KeyboardButton struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Text string `json:"text"`

	// Shared / Telegram URL-like
	URL string `json:"url,omitempty"`

	// Telegram
	CallbackData                 string                       `json:"callback_data,omitempty"`
	WebApp                       *WebAppInfo                  `json:"web_app,omitempty"`
	LoginURL                     *LoginURL                    `json:"login_url,omitempty"`
	SwitchInlineQuery            *string                      `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat *string                      `json:"switch_inline_query_current_chat,omitempty"`
	SwitchInlineQueryChosenChat  *SwitchInlineQueryChosenChat `json:"switch_inline_query_chosen_chat,omitempty"`
	CallbackGame                 *CallbackGame                `json:"callback_game,omitempty"`
	Pay                          bool                         `json:"pay,omitempty"`
	CopyText                     *CopyTextButton              `json:"copy_text,omitempty"`

	// MAX: callback / clipboard / message payload; open_app may use URL
	Payload string `json:"payload,omitempty"`
}

type WebAppInfo struct {
	URL string `json:"url"`
}

type LoginURL struct {
	URL                string  `json:"url"`
	ForwardText        *string `json:"forward_text,omitempty"`
	BotUsername        *string `json:"bot_username,omitempty"`
	RequestWriteAccess *bool   `json:"request_write_access,omitempty"`
}

type SwitchInlineQueryChosenChat struct {
	Query             *string `json:"query,omitempty"`
	AllowUserChats    *bool   `json:"allow_user_chats,omitempty"`
	AllowBotChats     *bool   `json:"allow_bot_chats,omitempty"`
	AllowGroupChats   *bool   `json:"allow_group_chats,omitempty"`
	AllowChannelChats *bool   `json:"allow_channel_chats,omitempty"`
}

type CallbackGame struct{}

type CopyTextButton struct {
	Text string `json:"text"`
}

func EmptyInlineKeyboard() InlineKeyboard {
	return InlineKeyboard{Rows: [][]KeyboardButton{}}
}

func ParseInlineKeyboard(raw string) (InlineKeyboard, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return EmptyInlineKeyboard(), nil
	}
	var kb InlineKeyboard
	if err := json.Unmarshal([]byte(raw), &kb); err != nil {
		return InlineKeyboard{}, fmt.Errorf("invalid keyboard json: %w", err)
	}
	if kb.Rows == nil {
		kb.Rows = [][]KeyboardButton{}
	}
	return kb, nil
}

func MarshalInlineKeyboard(kb InlineKeyboard) (string, error) {
	if kb.Rows == nil {
		kb.Rows = [][]KeyboardButton{}
	}
	b, err := json.Marshal(kb)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (kb *InlineKeyboard) IsEmpty() bool {
	if kb == nil {
		return true
	}
	for _, row := range kb.Rows {
		if len(row) > 0 {
			return false
		}
	}
	return true
}

// EnsureButtonIDs assigns a stable id to every button that does not have one.
func EnsureButtonIDs(kb *InlineKeyboard) {
	if kb == nil {
		return
	}
	for i := range kb.Rows {
		for j := range kb.Rows[i] {
			if strings.TrimSpace(kb.Rows[i][j].ID) == "" {
				kb.Rows[i][j].ID = uuid.NewString()
			}
		}
	}
}

func (kb *InlineKeyboard) ButtonCount() int {
	if kb == nil {
		return 0
	}
	n := 0
	for _, row := range kb.Rows {
		n += len(row)
	}
	return n
}

// ToTelegramReplyMarkup builds Bot API reply_markup.inline_keyboard (no type field).
func (kb *InlineKeyboard) ToTelegramReplyMarkup() map[string]any {
	if kb == nil || kb.IsEmpty() {
		return nil
	}
	rows := make([][]map[string]any, 0, len(kb.Rows))
	for _, row := range kb.Rows {
		if len(row) == 0 {
			continue
		}
		outRow := make([]map[string]any, 0, len(row))
		for _, btn := range row {
			outRow = append(outRow, telegramButtonPayload(btn))
		}
		rows = append(rows, outRow)
	}
	if len(rows) == 0 {
		return nil
	}
	return map[string]any{
		"inline_keyboard": rows,
	}
}

func telegramButtonPayload(btn KeyboardButton) map[string]any {
	m := map[string]any{"text": btn.Text}
	switch btn.Type {
	case "url":
		m["url"] = btn.URL
	case "callback_data":
		m["callback_data"] = btn.CallbackData
	case "web_app":
		if btn.WebApp != nil {
			m["web_app"] = map[string]any{"url": btn.WebApp.URL}
		}
	case "login_url":
		if btn.LoginURL != nil {
			lu := map[string]any{"url": btn.LoginURL.URL}
			if btn.LoginURL.ForwardText != nil {
				lu["forward_text"] = *btn.LoginURL.ForwardText
			}
			if btn.LoginURL.BotUsername != nil {
				lu["bot_username"] = *btn.LoginURL.BotUsername
			}
			if btn.LoginURL.RequestWriteAccess != nil {
				lu["request_write_access"] = *btn.LoginURL.RequestWriteAccess
			}
			m["login_url"] = lu
		}
	case "switch_inline_query":
		q := ""
		if btn.SwitchInlineQuery != nil {
			q = *btn.SwitchInlineQuery
		}
		m["switch_inline_query"] = q
	case "switch_inline_query_current_chat":
		q := ""
		if btn.SwitchInlineQueryCurrentChat != nil {
			q = *btn.SwitchInlineQueryCurrentChat
		}
		m["switch_inline_query_current_chat"] = q
	case "switch_inline_query_chosen_chat":
		if btn.SwitchInlineQueryChosenChat != nil {
			m["switch_inline_query_chosen_chat"] = btn.SwitchInlineQueryChosenChat
		} else {
			m["switch_inline_query_chosen_chat"] = map[string]any{}
		}
	case "callback_game":
		m["callback_game"] = map[string]any{}
	case "pay":
		m["pay"] = true
	case "copy_text":
		if btn.CopyText != nil {
			m["copy_text"] = map[string]any{"text": btn.CopyText.Text}
		}
	}
	return m
}

// ToMaxInlineKeyboardAttachment builds MAX attachments[] entry.
func (kb *InlineKeyboard) ToMaxInlineKeyboardAttachment() map[string]any {
	if kb == nil || kb.IsEmpty() {
		return nil
	}
	rows := make([][]map[string]any, 0, len(kb.Rows))
	for _, row := range kb.Rows {
		if len(row) == 0 {
			continue
		}
		outRow := make([]map[string]any, 0, len(row))
		for _, btn := range row {
			outRow = append(outRow, maxButtonPayload(btn))
		}
		rows = append(rows, outRow)
	}
	if len(rows) == 0 {
		return nil
	}
	return map[string]any{
		"type": "inline_keyboard",
		"payload": map[string]any{
			"buttons": rows,
		},
	}
}

func maxButtonPayload(btn KeyboardButton) map[string]any {
	m := map[string]any{
		"type": btn.Type,
		"text": btn.Text,
	}
	switch btn.Type {
	case "link", "open_app":
		if btn.URL != "" {
			m["url"] = btn.URL
		}
	case "callback", "clipboard", "message":
		if btn.Payload != "" {
			m["payload"] = btn.Payload
		}
	}
	return m
}
