package db

import "database/sql"

// Handle is a stable reference to the database that survives Manager.Reopen().
// Handlers should keep a Handle instead of caching *sql.DB at startup.
type Handle struct {
	m *Manager
}

func NewHandle(m *Manager) *Handle {
	return &Handle{m: m}
}

func (h *Handle) DB() *sql.DB {
	return h.m.DB()
}
