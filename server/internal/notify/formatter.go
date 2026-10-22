package notify

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var placeholderRe = regexp.MustCompile(`\{([a-z_]+)\}`)

type FormatData map[string]string

func Format(triggerType, localeCode string, userTemplate *string, data FormatData) (string, error) {
	template := defaultTemplate(localeCode, triggerType)
	if userTemplate != nil {
		template = strings.TrimSpace(*userTemplate)
	}
	if err := ValidateTemplate(triggerType, template); err != nil {
		return "", err
	}
	return replacePlaceholders(template, data), nil
}

func ValidateTemplate(triggerType, template string) error {
	template = strings.TrimSpace(template)
	if template == "" {
		return fmt.Errorf("template must not be empty")
	}
	allowed := make(map[string]struct{}, len(triggerPlaceholders[triggerType]))
	for _, key := range triggerPlaceholders[triggerType] {
		allowed[key] = struct{}{}
	}
	unknown := make(map[string]struct{})
	matches := placeholderRe.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if _, ok := allowed[match[1]]; !ok {
			unknown[match[1]] = struct{}{}
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	items := make([]string, 0, len(unknown))
	for key := range unknown {
		items = append(items, key)
	}
	sort.Strings(items)
	return fmt.Errorf("unknown placeholders: %s", strings.Join(items, ", "))
}

func AvailablePlaceholders(triggerType string) []string {
	items := triggerPlaceholders[triggerType]
	out := make([]string, len(items))
	copy(out, items)
	return out
}

func replacePlaceholders(template string, data FormatData) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		value, ok := data[key]
		if !ok {
			return ""
		}
		return value
	})
}

func FormatAmountDisplay(amount int64, currencyCode string) string {
	base := money.FormatRubles(amount)
	parts := strings.SplitN(base, ".", 2)
	intPart := addThousandsSeparator(parts[0])
	symbol := currencySymbol(currencyCode)
	if len(parts) == 1 {
		return intPart + " " + symbol
	}
	return intPart + "." + parts[1] + " " + symbol
}

func currencySymbol(currencyCode string) string {
	switch strings.ToUpper(strings.TrimSpace(currencyCode)) {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	default:
		return "₽"
	}
}

func addThousandsSeparator(value string) string {
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	if len(value) <= 3 {
		if negative {
			return "-" + value
		}
		return value
	}
	var out strings.Builder
	for i, ch := range value {
		if i != 0 && (len(value)-i)%3 == 0 {
			out.WriteRune(' ')
		}
		out.WriteRune(ch)
	}
	if negative {
		return "-" + out.String()
	}
	return out.String()
}

func FormatDateInTimezone(value, timezone, layout string) string {
	switch layout {
	case timeutil.DisplayDateLayout:
		return timeutil.FormatDisplayDateInTimezone(value, timezone)
	case timeutil.DisplayDateTimeLayout:
		return timeutil.FormatDisplayDateTimeInTimezone(value, timezone)
	case timeutil.DisplayDateTimeShortLayout:
		return timeutil.FormatDisplayDateTimeShortInTimezone(value, timezone)
	default:
		return formatDateInTimezoneWithLayout(value, timezone, layout)
	}
}

func formatDateInTimezoneWithLayout(value, timezone, layout string) string {
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	tm, err := timeutil.ParseFlexibleUTC(value)
	if err != nil {
		return value
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return value
	}
	return tm.In(loc).Format(layout)
}

func RelativeWhen(localeCode string, value string, now time.Time, timezone string) string {
	tm, err := timeutil.ParseFlexibleUTC(value)
	if err != nil {
		return FormatDateInTimezone(value, timezone, timeutil.DisplayDateLayout)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	target := tm.In(loc)
	current := now.In(loc)
	days := dayDiff(current, target)

	ru := normalizeLocale(localeCode) == "ru"
	switch days {
	case 0:
		if ru {
			return "сегодня"
		}
		return "today"
	case 1:
		if ru {
			return "завтра"
		}
		return "tomorrow"
	default:
		if ru && days > 1 && days <= 7 {
			return "через " + strconv.Itoa(days) + " дн."
		}
		return target.Format(timeutil.DisplayDateLayout)
	}
}

// RelativeDays returns a human-readable relative phrase for a day offset
// (0 → сегодня/today, 1 → завтра/tomorrow, N → через N дн. / in N days).
func RelativeDays(localeCode string, days int) string {
	ru := normalizeLocale(localeCode) == "ru"
	if days <= 0 {
		if ru {
			return "сегодня"
		}
		return "today"
	}
	if days == 1 {
		if ru {
			return "завтра"
		}
		return "tomorrow"
	}
	if ru {
		return "через " + strconv.Itoa(days) + " дн."
	}
	return "in " + strconv.Itoa(days) + " days"
}

// DebtActionPhrase is the direction-aware verb phrase for debt_due_soon templates.
// borrowed → «вернуть долг» / «repay debt to»; lent → «получить долг от» / «collect debt from».
func DebtActionPhrase(localeCode, direction string) string {
	ru := normalizeLocale(localeCode) == "ru"
	if strings.TrimSpace(direction) == "lent" {
		if ru {
			return "получить долг от"
		}
		return "collect debt from"
	}
	if ru {
		return "вернуть долг"
	}
	return "repay debt to"
}

func dayDiff(from, to time.Time) int {
	yearA, monthA, dayA := from.Date()
	yearB, monthB, dayB := to.Date()
	a := time.Date(yearA, monthA, dayA, 0, 0, 0, 0, from.Location())
	b := time.Date(yearB, monthB, dayB, 0, 0, 0, 0, to.Location())
	return int(b.Sub(a).Hours() / 24)
}
