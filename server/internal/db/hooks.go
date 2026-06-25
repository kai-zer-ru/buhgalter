package db

import (
	"context"
	"database/sql"
)

// OpenHook runs once after migrations when a database connection is opened.
type OpenHook func(ctx context.Context, db *sql.DB) error

var openHooks []OpenHook

// RegisterOpenHook adds a startup hook. Intended for init() in feature packages.
func RegisterOpenHook(hook OpenHook) {
	openHooks = append(openHooks, hook)
}

func runOpenHooks(ctx context.Context, db *sql.DB) error {
	for _, hook := range openHooks {
		if err := hook(ctx, db); err != nil {
			return err
		}
	}
	return nil
}
