package notify

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseInlineKeyboardEmpty(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", "null", "[]", `{"rows":[]}`, `{"rows":null}`} {
		kb, err := ParseInlineKeyboard(raw)
		if err != nil {
			t.Fatalf("%q: %v", raw, err)
		}
		if !kb.IsEmpty() {
			t.Fatalf("%q: expected empty", raw)
		}
	}
}

func TestMergeKeyboardsOrder(t *testing.T) {
	t.Parallel()
	g := &InlineKeyboard{Rows: [][]KeyboardButton{{{Type: "callback_data", Text: "G", CallbackData: "g"}}}}
	tr := &InlineKeyboard{Rows: [][]KeyboardButton{{{Type: "callback_data", Text: "T", CallbackData: "t"}}}}
	sf := &InlineKeyboard{Rows: [][]KeyboardButton{{{Type: "callback_data", Text: "S", CallbackData: "s"}}}}
	merged := MergeKeyboards(g, tr, sf)
	if merged == nil || len(merged.Rows) != 3 {
		t.Fatalf("rows: %+v", merged)
	}
	if merged.Rows[0][0].Text != "G" || merged.Rows[1][0].Text != "T" || merged.Rows[2][0].Text != "S" {
		t.Fatalf("order: %+v", merged.Rows)
	}
}

func TestFilterUnresolvedURLButtonsT1(t *testing.T) {
	t.Parallel()
	kb := &InlineKeyboard{Rows: [][]KeyboardButton{
		{
			{Type: "url", Text: "Open", URL: "https://example.com/debts"},
			{Type: "url", Text: "Bad", URL: "Нет внешней ссылки — настройте внешний URL в админке."},
			{Type: "callback_data", Text: "OK", CallbackData: "x"},
		},
		{
			{Type: "link", Text: "Empty", URL: ""},
		},
	}}
	filtered := FilterUnresolvedURLButtons(kb)
	if filtered == nil || len(filtered.Rows) != 1 || len(filtered.Rows[0]) != 2 {
		t.Fatalf("filtered: %+v", filtered)
	}
	if filtered.Rows[0][0].Type != "url" || filtered.Rows[0][1].Type != "callback_data" {
		t.Fatalf("kept: %+v", filtered.Rows[0])
	}
}

func TestFilterUnresolvedURLButtonsAllDropped(t *testing.T) {
	t.Parallel()
	kb := &InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "web_app", Text: "App", WebApp: &WebAppInfo{URL: ""}}},
	}}
	if FilterUnresolvedURLButtons(kb) != nil {
		t.Fatal("expected nil")
	}
}

func TestResolveKeyboardPlaceholders(t *testing.T) {
	t.Parallel()
	kb := &InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "url", Text: "D", URL: "{debt_url}"}},
	}}
	resolved := ResolveKeyboardPlaceholders(kb, FormatData{"debt_url": "https://ex.com/debts"})
	if resolved.Rows[0][0].URL != "https://ex.com/debts" {
		t.Fatalf("url: %q", resolved.Rows[0][0].URL)
	}
}

func TestValidateTelegramKeyboardLimits(t *testing.T) {
	t.Parallel()
	ok := InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "callback_data", Text: "A", CallbackData: "a"}},
	}}
	if err := ValidateTelegramKeyboard(ok); err != nil {
		t.Fatal(err)
	}
	bad := InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "callback_data", Text: "A", CallbackData: strings.Repeat("x", 65)}},
	}}
	if err := ValidateTelegramKeyboard(bad); err == nil {
		t.Fatal("expected callback_data length error")
	}
}

func TestValidateMaxKeyboardSpecialRow(t *testing.T) {
	t.Parallel()
	ok := InlineKeyboard{Rows: [][]KeyboardButton{
		{
			{Type: "link", Text: "1", URL: "https://a"},
			{Type: "link", Text: "2", URL: "https://b"},
			{Type: "open_app", Text: "3", URL: "https://c"},
		},
	}}
	if err := ValidateMaxKeyboard(ok); err != nil {
		t.Fatal(err)
	}
	bad := InlineKeyboard{Rows: [][]KeyboardButton{
		{
			{Type: "link", Text: "1", URL: "https://a"},
			{Type: "link", Text: "2", URL: "https://b"},
			{Type: "link", Text: "3", URL: "https://c"},
			{Type: "link", Text: "4", URL: "https://d"},
		},
	}}
	if err := ValidateMaxKeyboard(bad); err == nil {
		t.Fatal("expected special row limit error")
	}
}

func TestToTelegramAndMaxPayload(t *testing.T) {
	t.Parallel()
	kb := &InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "url", Text: "Open", URL: "https://ex.com"}},
		{{Type: "link", Text: "Max", URL: "https://max.ex"}},
	}}
	tg := kb.ToTelegramReplyMarkup()
	raw, _ := json.Marshal(tg)
	if !strings.Contains(string(raw), `"inline_keyboard"`) || !strings.Contains(string(raw), `"url":"https://ex.com"`) {
		t.Fatalf("tg: %s", raw)
	}
	maxAtt := kb.ToMaxInlineKeyboardAttachment()
	raw, _ = json.Marshal(maxAtt)
	if !strings.Contains(string(raw), `"type":"inline_keyboard"`) || !strings.Contains(string(raw), `"buttons"`) {
		t.Fatalf("max: %s", raw)
	}
}

func TestPrepareChannelKeyboardMergeAndFilter(t *testing.T) {
	t.Parallel()
	global := `{"rows":[[{"type":"callback_data","text":"G","callback_data":"g"}]]}`
	trigger := `{"rows":[[{"type":"url","text":"U","url":"{debt_url}"}]]}`
	shortfall := `{"rows":[[{"type":"callback_data","text":"{amount}","callback_data":"s"}]]}`
	data := FormatData{"debt_url": "https://ex.com/debts", "amount": "10 000.00 ₽"}
	sfData := FormatData{"amount": "1 200.00 ₽"}
	kb := PrepareChannelKeyboard(global, trigger, shortfall, data, sfData, ChannelTelegram)
	if kb == nil || len(kb.Rows) != 3 {
		t.Fatalf("got %+v", kb)
	}
	if kb.Rows[2][0].Text != "1 200.00 ₽" {
		t.Fatalf("shortfall zone must use shortfall amount, got %q", kb.Rows[2][0].Text)
	}
	dataEmpty := FormatData{"debt_url": "Нет внешней ссылки", "amount": "10 000.00 ₽"}
	kb2 := PrepareChannelKeyboard(global, trigger, "", dataEmpty, nil, ChannelTelegram)
	if kb2 == nil || len(kb2.Rows) != 1 || kb2.Rows[0][0].Text != "G" {
		t.Fatalf("filtered url: %+v", kb2)
	}
}
