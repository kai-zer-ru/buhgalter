package backup

import (
	"log/slog"
	"strings"
	"time"
)

// ShouldRunAt reports whether a scheduled backup should run at the given local time.
func ShouldRunAt(settings Settings, now time.Time, lastRunKey string) (bool, string) {
	if !settings.BackupEnabled {
		return false, lastRunKey
	}
	parts := strings.Split(settings.BackupTime, ":")
	if len(parts) != 2 {
		return false, lastRunKey
	}
	slot := now.Format("15:04")
	if slot != settings.BackupTime {
		return false, lastRunKey
	}
	key := now.Format("2006-01-02") + ":" + slot
	if key == lastRunKey {
		return false, lastRunKey
	}
	return true, key
}

type Scheduler struct {
	Service *Service
	Logger  *slog.Logger
	stop    chan struct{}
}

func NewScheduler(service *Service, logger *slog.Logger) *Scheduler {
	return &Scheduler{Service: service, Logger: logger, stop: make(chan struct{})}
}

func (s *Scheduler) Start() {
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	var lastRun string

	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			settings, err := s.Service.GetSettings()
			if err != nil {
				continue
			}

			should, key := ShouldRunAt(settings, now, lastRun)
			if !should {
				continue
			}
			lastRun = key

			name, err := s.Service.Create()
			if err != nil {
				s.Logger.Error("scheduled backup failed", "err", err)
				continue
			}
			s.Logger.Info("scheduled backup created", "filename", name)
		}
	}
}
