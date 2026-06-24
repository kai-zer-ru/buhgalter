package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type Service struct {
	Manager  *db.Manager
	BackupDir string
}

type FileInfo struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) Dir() string {
	return s.BackupDir
}

func (s *Service) EnsureDir() error {
	return os.MkdirAll(s.BackupDir, 0o755)
}

func (s *Service) Create() (string, error) {
	if err := s.EnsureDir(); err != nil {
		return "", err
	}

	name := fmt.Sprintf("buhgalter_%s.db", time.Now().Format("20060102_150405"))
	dest := filepath.Join(s.BackupDir, name)
	if err := s.Manager.VacuumInto(dest); err != nil {
		return "", err
	}

	if err := s.applyRetention(); err != nil {
		return name, err
	}
	return name, nil
}

func (s *Service) List() ([]FileInfo, error) {
	if err := s.EnsureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.BackupDir)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Filename:  e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Filename > files[j].Filename
	})
	if files == nil {
		files = []FileInfo{}
	}
	return files, nil
}

func (s *Service) PathFor(filename string) (string, error) {
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", fmt.Errorf("invalid filename")
	}
	path := filepath.Join(s.BackupDir, filename)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) LatestPath() (string, error) {
	files, err := s.List()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		name, err := s.Create()
		if err != nil {
			return "", err
		}
		return filepath.Join(s.BackupDir, name), nil
	}
	return filepath.Join(s.BackupDir, files[0].Filename), nil
}

func (s *Service) Restore(src io.Reader) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	tmpPath := s.Manager.Path() + ".restore.tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	_ = s.Manager.Checkpoint()
	if err := s.Manager.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	dbPath := s.Manager.Path()
	db.RemoveSQLiteSidecars(dbPath)
	_ = os.Rename(dbPath, dbPath+".bak")
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = os.Rename(dbPath+".bak", dbPath)
		_ = s.Manager.Reopen()
		return err
	}
	db.RemoveSQLiteSidecars(dbPath)
	_ = os.Remove(dbPath + ".bak")
	if err := s.Manager.Reopen(); err != nil {
		return err
	}
	return nil
}

func (s *Service) applyRetention() error {
	var retention int
	if err := s.Manager.DB().QueryRow(`SELECT backup_retention FROM system_settings WHERE id = 1`).Scan(&retention); err != nil {
		return err
	}
	if retention <= 0 {
		return nil
	}

	files, err := s.List()
	if err != nil {
		return err
	}
	if len(files) <= retention {
		return nil
	}
	for _, f := range files[retention:] {
		_ = os.Remove(filepath.Join(s.BackupDir, f.Filename))
	}
	return nil
}

type Settings struct {
	BackupEnabled   bool   `json:"backup_enabled"`
	BackupTime      string `json:"backup_time"`
	BackupRetention int    `json:"backup_retention"`
}

func (s *Service) GetSettings() (Settings, error) {
	var enabled, retention int
	var backupTime string
	err := s.Manager.DB().QueryRow(`
		SELECT backup_enabled, backup_time, backup_retention FROM system_settings WHERE id = 1`,
	).Scan(&enabled, &backupTime, &retention)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		BackupEnabled:   enabled == 1,
		BackupTime:      backupTime,
		BackupRetention: retention,
	}, nil
}

func (s *Service) UpdateSettings(enabled bool, backupTime string, retention int) error {
	if retention < 1 {
		retention = 1
	}
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.Manager.DB().Exec(`
		UPDATE system_settings
		SET backup_enabled = ?, backup_time = ?, backup_retention = ?, updated_at = datetime('now')
		WHERE id = 1`, en, backupTime, retention,
	)
	return err
}
