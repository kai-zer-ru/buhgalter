package features

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const cacheTTL = time.Minute

type snapshotCache struct {
	values    map[string]bool
	fetchedAt time.Time
}

var (
	cacheMu sync.RWMutex
	cache   snapshotCache
)

// Invalidate clears the in-memory feature snapshot cache.
func Invalidate() {
	cacheMu.Lock()
	cache = snapshotCache{}
	cacheMu.Unlock()
}

// Snapshot returns enabled state for every registry key (defaults when DB row missing).
func Snapshot(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	cacheMu.RLock()
	if time.Since(cache.fetchedAt) < cacheTTL && cache.values != nil {
		out := copyMap(cache.values)
		cacheMu.RUnlock()
		return out, nil
	}
	cacheMu.RUnlock()

	rows, err := sqlcdb.New(db).ListFeatureFlags(ctx)
	if err != nil {
		return nil, err
	}
	dbMap := make(map[string]bool, len(rows))
	for _, row := range rows {
		if !IsKnown(row.Key) {
			continue
		}
		dbMap[row.Key] = row.Enabled == 1
	}

	out := make(map[string]bool, len(Registry))
	for _, f := range Registry {
		if v, ok := dbMap[f.Key]; ok {
			out[f.Key] = v
		} else {
			out[f.Key] = f.DefaultEnabled
		}
	}

	cacheMu.Lock()
	cache = snapshotCache{values: copyMap(out), fetchedAt: time.Now()}
	cacheMu.Unlock()
	return out, nil
}

// IsEnabled reports whether a known flag is enabled. Unknown keys return false, err.
func IsEnabled(ctx context.Context, db *sql.DB, key string) (bool, error) {
	if !IsKnown(key) {
		return false, fmt.Errorf("unknown feature flag %q", key)
	}
	snap, err := Snapshot(ctx, db)
	if err != nil {
		return false, err
	}
	return snap[key], nil
}

// SetMany upserts overrides for known keys. Unknown keys return an error without writes.
// db may be *sql.DB or *sql.Tx (sqlc DBTX).
func SetMany(ctx context.Context, db sqlcdb.DBTX, updates map[string]bool) error {
	if len(updates) == 0 {
		return nil
	}
	for key := range updates {
		if !IsKnown(key) {
			return fmt.Errorf("unknown feature flag %q", key)
		}
	}
	q := sqlcdb.New(db)
	for key, enabled := range updates {
		v := int64(0)
		if enabled {
			v = 1
		}
		if err := q.UpsertFeatureFlag(ctx, sqlcdb.UpsertFeatureFlagParams{
			Key:     key,
			Enabled: v,
		}); err != nil {
			return err
		}
	}
	Invalidate()
	return nil
}

// SetOne upserts a single known flag.
func SetOne(ctx context.Context, db sqlcdb.DBTX, key string, enabled bool) error {
	return SetMany(ctx, db, map[string]bool{key: enabled})
}

func copyMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
