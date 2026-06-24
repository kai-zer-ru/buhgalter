package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Manager struct {
	mu   sync.RWMutex
	path string
	db   *sql.DB
}

func NewManager(path string) (*Manager, error) {
	sqlDB, err := Open(path)
	if err != nil {
		return nil, err
	}
	return &Manager{path: path, db: sqlDB}, nil
}

func (m *Manager) DB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Reopen() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		_ = m.db.Close()
	}
	sqlDB, err := Open(m.path)
	if err != nil {
		return err
	}
	m.db = sqlDB
	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return nil
	}
	err := m.db.Close()
	m.db = nil
	return err
}

func (m *Manager) VacuumInto(dest string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	escaped := strings.ReplaceAll(dest, "'", "''")
	_, err := m.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, escaped))
	return err
}

func (m *Manager) Checkpoint() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.db == nil {
		return nil
	}
	_, err := m.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func RemoveSQLiteSidecars(path string) {
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
}
