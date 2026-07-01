package settingscache

import (
	"context"
	"database/sql"
	"sync"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
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

	raw, err := sqlcdb.New(db).GetExternalURL(ctx)
	if err != nil {
		return sql.NullString{}, err
	}
	var externalURL sql.NullString
	if raw != nil {
		externalURL = sql.NullString{String: *raw, Valid: true}
	}

	mu.Lock()
	cache = externalAccess{externalURL: externalURL, fetchedAt: time.Now()}
	mu.Unlock()
	return externalURL, nil
}
