package receipt

import (
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// NormalizeFiscalKey returns trimmed fiscal fields, falling back to receipt payload.
func NormalizeFiscalKey(fn, fd, fp string, receipt fns.ParsedReceipt) (string, string, string) {
	return strings.TrimSpace(firstNonEmpty(fn, receipt.FiscalDriveNumber)),
		strings.TrimSpace(firstNonEmpty(fd, receipt.FiscalDocumentNumber)),
		strings.TrimSpace(firstNonEmpty(fp, receipt.FiscalSign))
}
