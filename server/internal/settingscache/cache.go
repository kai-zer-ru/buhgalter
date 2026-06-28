package settingscache

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

type externalAccess struct {
	externalURL sql.NullString
	fetchedAt   time.Time
}

var (
	mu    sync.RWMutex
	cache externalAccess
	ttl   = time.Minute
)

func Invalidate() {
	mu.Lock()
	cache = externalAccess{}
	mu.Unlock()
}

func ExternalURL(ctx context.Context, db *sql.DB) (sql.NullString, error) {
	mu.RLock()
	if time.Since(cache.fetchedAt) < ttl {
		v := cache.externalURL
		mu.RUnlock()
		return v, nil
	}
	mu.RUnlock()

	var externalURL sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT external_url FROM system_settings WHERE id = 1`,
	).Scan(&externalURL)
	if err != nil {
		return sql.NullString{}, err
	}

	mu.Lock()
	cache = externalAccess{externalURL: externalURL, fetchedAt: time.Now()}
	mu.Unlock()
	return externalURL, nil
}
