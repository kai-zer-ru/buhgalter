package notify

import (
	"strings"
)

// MergeKeyboards concatenates rows: global → trigger → shortfall (nil zones skipped).
func MergeKeyboards(global, trigger, shortfall *InlineKeyboard) *InlineKeyboard {
	out := &InlineKeyboard{Rows: make([][]KeyboardButton, 0)}
	appendRows := func(src *InlineKeyboard) {
		if src == nil || src.IsEmpty() {
			return
		}
		for _, row := range src.Rows {
			if len(row) == 0 {
				continue
			}
			cp := make([]KeyboardButton, len(row))
			copy(cp, row)
			out.Rows = append(out.Rows, cp)
		}
	}
	appendRows(global)
	appendRows(trigger)
	appendRows(shortfall)
	if out.IsEmpty() {
		return nil
	}
	return out
}

// ResolveKeyboardPlaceholders replaces {placeholders} in URL/payload/text fields
// using the same replacePlaceholders mechanism as message templates.
func ResolveKeyboardPlaceholders(kb *InlineKeyboard, data FormatData) *InlineKeyboard {
	if kb == nil || kb.IsEmpty() {
		return kb
	}
	out := &InlineKeyboard{Rows: make([][]KeyboardButton, 0, len(kb.Rows)), Placement: kb.Placement}
	for _, row := range kb.Rows {
		cp := make([]KeyboardButton, len(row))
		for i, btn := range row {
			cp[i] = resolveButtonPlaceholders(btn, data)
		}
		out.Rows = append(out.Rows, cp)
	}
	return out
}

func resolveButtonPlaceholders(btn KeyboardButton, data FormatData) KeyboardButton {
	btn.Text = replacePlaceholders(btn.Text, data)
	btn.URL = replacePlaceholders(btn.URL, data)
	btn.CallbackData = replacePlaceholders(btn.CallbackData, data)
	btn.Payload = replacePlaceholders(btn.Payload, data)
	if btn.WebApp != nil {
		u := replacePlaceholders(btn.WebApp.URL, data)
		btn.WebApp = &WebAppInfo{URL: u}
	}
	if btn.LoginURL != nil {
		lu := *btn.LoginURL
		lu.URL = replacePlaceholders(lu.URL, data)
		if lu.ForwardText != nil {
			ft := replacePlaceholders(*lu.ForwardText, data)
			lu.ForwardText = &ft
		}
		btn.LoginURL = &lu
	}
	if btn.CopyText != nil {
		btn.CopyText = &CopyTextButton{Text: replacePlaceholders(btn.CopyText.Text, data)}
	}
	if btn.SwitchInlineQuery != nil {
		q := replacePlaceholders(*btn.SwitchInlineQuery, data)
		btn.SwitchInlineQuery = &q
	}
	if btn.SwitchInlineQueryCurrentChat != nil {
		q := replacePlaceholders(*btn.SwitchInlineQueryCurrentChat, data)
		btn.SwitchInlineQueryCurrentChat = &q
	}
	if btn.SwitchInlineQueryChosenChat != nil && btn.SwitchInlineQueryChosenChat.Query != nil {
		q := replacePlaceholders(*btn.SwitchInlineQueryChosenChat.Query, data)
		chosen := *btn.SwitchInlineQueryChosenChat
		chosen.Query = &q
		btn.SwitchInlineQueryChosenChat = &chosen
	}
	return btn
}

// FilterUnresolvedURLButtons drops buttons that require a resolved http(s) URL
// when the URL is empty or not absolute http(s) (T1). Empty keyboard → nil.
func FilterUnresolvedURLButtons(kb *InlineKeyboard) *InlineKeyboard {
	if kb == nil || kb.IsEmpty() {
		return nil
	}
	out := &InlineKeyboard{Rows: make([][]KeyboardButton, 0, len(kb.Rows))}
	for _, row := range kb.Rows {
		kept := make([]KeyboardButton, 0, len(row))
		for _, btn := range row {
			if buttonRequiresURL(btn) && !isResolvedHTTPURL(buttonURLValue(btn)) {
				continue
			}
			kept = append(kept, btn)
		}
		if len(kept) > 0 {
			out.Rows = append(out.Rows, kept)
		}
	}
	if out.IsEmpty() {
		return nil
	}
	return out
}

func buttonRequiresURL(btn KeyboardButton) bool {
	switch btn.Type {
	case "url", "link", "web_app", "open_app", "login_url":
		return true
	default:
		return false
	}
}

func buttonURLValue(btn KeyboardButton) string {
	switch btn.Type {
	case "url", "link", "open_app":
		return strings.TrimSpace(btn.URL)
	case "web_app":
		if btn.WebApp == nil {
			return ""
		}
		return strings.TrimSpace(btn.WebApp.URL)
	case "login_url":
		if btn.LoginURL == nil {
			return ""
		}
		return strings.TrimSpace(btn.LoginURL.URL)
	default:
		return ""
	}
}

func isResolvedHTTPURL(u string) bool {
	u = strings.TrimSpace(u)
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// ArrangeTemplateKeyboard builds the template zone.
// No placement: global rows, then local rows.
// With placement: tokens g:<id> / l:<id>. A global id that is missing from placement
// is appended (original row grouping). A token whose global id is gone is dropped.
// balance_shortfall is not part of this zone.
func ArrangeTemplateKeyboard(global, trigger *InlineKeyboard) *InlineKeyboard {
	if trigger == nil || len(trigger.Placement) == 0 {
		return MergeKeyboards(global, trigger, nil)
	}
	gIndex := map[string]KeyboardButton{}
	if global != nil {
		for _, row := range global.Rows {
			for _, btn := range row {
				if id := strings.TrimSpace(btn.ID); id != "" {
					gIndex[id] = btn
				}
			}
		}
	}
	lIndex := map[string]KeyboardButton{}
	for _, row := range trigger.Rows {
		for _, btn := range row {
			if id := strings.TrimSpace(btn.ID); id != "" {
				lIndex[id] = btn
			}
		}
	}
	usedG := map[string]bool{}
	usedL := map[string]bool{}
	out := &InlineKeyboard{Rows: make([][]KeyboardButton, 0)}
	for _, prow := range trigger.Placement {
		row := make([]KeyboardButton, 0, len(prow))
		for _, token := range prow {
			token = strings.TrimSpace(token)
			switch {
			case strings.HasPrefix(token, "g:"):
				id := strings.TrimPrefix(token, "g:")
				btn, ok := gIndex[id]
				if !ok {
					continue
				}
				row = append(row, btn)
				usedG[id] = true
			case strings.HasPrefix(token, "l:"):
				id := strings.TrimPrefix(token, "l:")
				btn, ok := lIndex[id]
				if !ok {
					continue
				}
				row = append(row, btn)
				usedL[id] = true
			}
		}
		if len(row) > 0 {
			out.Rows = append(out.Rows, row)
		}
	}
	appendUnused := func(src *InlineKeyboard, index map[string]KeyboardButton, used map[string]bool) {
		if src == nil {
			return
		}
		for _, srcRow := range src.Rows {
			pending := make([]KeyboardButton, 0)
			for _, btn := range srcRow {
				id := strings.TrimSpace(btn.ID)
				if id == "" || !used[id] {
					if id != "" {
						if _, ok := index[id]; ok {
							used[id] = true
						}
					}
					pending = append(pending, btn)
				}
			}
			if len(pending) > 0 {
				out.Rows = append(out.Rows, pending)
			}
		}
	}
	appendUnused(global, gIndex, usedG)
	appendUnused(trigger, lIndex, usedL)
	if out.IsEmpty() {
		return nil
	}
	return out
}

// NormalizeTemplateKeyboard keeps local rows and puts every live global id back into placement.
// Empty placement stays empty (legacy order). Global keyboard itself is not modified.
func NormalizeTemplateKeyboard(global *InlineKeyboard, trigger InlineKeyboard) InlineKeyboard {
	EnsureButtonIDs(&trigger)
	if len(trigger.Placement) == 0 {
		trigger.Placement = nil
		return trigger
	}
	knownG := map[string]bool{}
	gIDs := make([]string, 0)
	if global != nil {
		for _, row := range global.Rows {
			for _, btn := range row {
				id := strings.TrimSpace(btn.ID)
				if id == "" || knownG[id] {
					continue
				}
				knownG[id] = true
				gIDs = append(gIDs, id)
			}
		}
	}
	knownL := map[string]bool{}
	for _, row := range trigger.Rows {
		for _, btn := range row {
			if id := strings.TrimSpace(btn.ID); id != "" {
				knownL[id] = true
			}
		}
	}
	usedG := map[string]bool{}
	placement := make([][]string, 0, len(trigger.Placement))
	for _, prow := range trigger.Placement {
		row := make([]string, 0, len(prow))
		for _, token := range prow {
			token = strings.TrimSpace(token)
			switch {
			case strings.HasPrefix(token, "g:"):
				id := strings.TrimPrefix(token, "g:")
				if !knownG[id] || usedG[id] {
					continue
				}
				usedG[id] = true
				row = append(row, "g:"+id)
			case strings.HasPrefix(token, "l:"):
				id := strings.TrimPrefix(token, "l:")
				if !knownL[id] {
					continue
				}
				row = append(row, "l:"+id)
			}
		}
		if len(row) > 0 {
			placement = append(placement, row)
		}
	}
	for _, id := range gIDs {
		if !usedG[id] {
			placement = append(placement, []string{"g:" + id})
		}
	}
	trigger.Placement = placement
	return trigger
}

// PrepareChannelKeyboard arranges global+trigger, appends shortfall, resolves placeholders per zone, applies T1.
// shortfallData == nil skips the shortfall zone. When non-nil, shortfall keys overlay data
// (so {amount} in the shortfall keyboard is the shortfall sum, not the main trigger amount).
func PrepareChannelKeyboard(
	globalRaw, triggerRaw, shortfallRaw string,
	data FormatData,
	shortfallData FormatData,
	provider string, // ChannelTelegram | ChannelMax — for docs only; shape is shared
) *InlineKeyboard {
	_ = provider
	global, _ := ParseInlineKeyboard(globalRaw)
	trigger, _ := ParseInlineKeyboard(triggerRaw)
	globalResolved := ResolveKeyboardPlaceholders(&global, data)
	triggerResolved := ResolveKeyboardPlaceholders(&trigger, data)
	var shortfallResolved *InlineKeyboard
	if shortfallData != nil {
		sf, _ := ParseInlineKeyboard(shortfallRaw)
		shortfallResolved = ResolveKeyboardPlaceholders(&sf, overlayFormatData(data, shortfallData))
	}
	arranged := ArrangeTemplateKeyboard(globalResolved, triggerResolved)
	merged := MergeKeyboards(arranged, shortfallResolved, nil)
	return FilterUnresolvedURLButtons(merged)
}

func overlayFormatData(base, overlay FormatData) FormatData {
	out := make(FormatData, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func keyboardJSONOrEmpty(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return emptyKeyboardJSON
	}
	return raw
}
