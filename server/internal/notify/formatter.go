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
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	tm, err := timeutil.ParseUTC(value)
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
	tm, err := timeutil.ParseUTC(value)
	if err != nil {
		return FormatDateInTimezone(value, timezone, "02.01.2006")
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
		return target.Format("02.01.2006")
	}
}

func dayDiff(from, to time.Time) int {
	yearA, monthA, dayA := from.Date()
	yearB, monthB, dayB := to.Date()
	a := time.Date(yearA, monthA, dayA, 0, 0, 0, 0, from.Location())
	b := time.Date(yearB, monthB, dayB, 0, 0, 0, 0, to.Location())
	return int(b.Sub(a).Hours() / 24)
}
