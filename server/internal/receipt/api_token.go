package receipt

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/proverkacheka"
)

// ErrAPINotConfigured means no receipt API token is stored (or it cannot be decrypted).
var ErrAPINotConfigured = errors.New("receipt api not configured")

// ResolveAPIToken returns the decrypted proverkacheka token, or "" when not configured.
func ResolveAPIToken(ctx context.Context, sqlDB *sql.DB) (string, error) {
	q := sqlcdb.New(sqlDB)
	enc, err := q.GetReceiptAPITokenEncrypted(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(enc) == "" {
		return "", nil
	}
	secret, err := notify.ResolveSecretKey(ctx, sqlDB)
	if err != nil {
		return "", err
	}
	box, err := notify.NewSecretBox(secret)
	if err != nil {
		return "", nil
	}
	token, err := box.Decrypt(enc)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(token), nil
}

// APIConfigured reports whether a usable receipt API token exists.
func APIConfigured(ctx context.Context, sqlDB *sql.DB) (bool, error) {
	token, err := ResolveAPIToken(ctx, sqlDB)
	return token != "", err
}

const receiptTokenCacheTTL = 5 * time.Minute

var (
	receiptTokenCacheMu  sync.Mutex
	receiptTokenCached   string
	receiptTokenCachedAt time.Time
)

func resolveCachedAPIToken(ctx context.Context, sqlDB *sql.DB) (string, error) {
	receiptTokenCacheMu.Lock()
	defer receiptTokenCacheMu.Unlock()
	if receiptTokenCached != "" && time.Since(receiptTokenCachedAt) < receiptTokenCacheTTL {
		return receiptTokenCached, nil
	}
	token, err := ResolveAPIToken(ctx, sqlDB)
	if err != nil {
		return "", err
	}
	receiptTokenCached = token
	receiptTokenCachedAt = time.Now()
	return token, nil
}

// InvalidateReceiptTokenCache clears the in-process token cache (after admin updates token).
func InvalidateReceiptTokenCache() {
	receiptTokenCacheMu.Lock()
	receiptTokenCached = ""
	receiptTokenCachedAt = time.Time{}
	receiptTokenCacheMu.Unlock()
}

// TokenFetcher resolves the instance token per call and queries proverkacheka.
type TokenFetcher struct {
	DB *sql.DB
}

func (f TokenFetcher) FetchReceipt(ctx context.Context, qr string) (fns.ParsedReceipt, error) {
	token, err := resolveCachedAPIToken(ctx, f.DB)
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	if token == "" {
		return fns.ParsedReceipt{}, ErrAPINotConfigured
	}
	return proverkacheka.NewClient(token).FetchReceipt(ctx, qr)
}

// RequireAPIConfigured rejects requests with 503 ERR_RECEIPT_API_NOT_CONFIGURED when no token is set.
func RequireAPIConfigured(store *db.Handle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := APIConfigured(r.Context(), store.DB())
			if err != nil {
				apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
				return
			}
			if !ok {
				writeAPINotConfigured(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAPINotConfigured(w http.ResponseWriter, r *http.Request) {
	apperror.WriteL(w, r, http.StatusServiceUnavailable, "ERR_RECEIPT_API_NOT_CONFIGURED", "ERR_RECEIPT_API_NOT_CONFIGURED", "ERR_RECEIPT_API_NOT_CONFIGURED")
}

// StoreAPIToken encrypts and persists the proverkacheka API token.
func StoreAPIToken(ctx context.Context, sqlDB *sql.DB, plainToken string) error {
	plainToken = strings.TrimSpace(plainToken)
	if plainToken == "" {
		return errors.New("empty token")
	}
	secret, err := notify.ResolveSecretKey(ctx, sqlDB)
	if err != nil {
		return err
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("notification secret not configured")
	}
	box, err := notify.NewSecretBox(secret)
	if err != nil {
		return err
	}
	cipher, err := box.Encrypt(plainToken)
	if err != nil {
		return err
	}
	if err := sqlcdb.New(sqlDB).UpdateReceiptAPIToken(ctx, cipher); err != nil {
		return err
	}
	InvalidateReceiptTokenCache()
	return nil
}
