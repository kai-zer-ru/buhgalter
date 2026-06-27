package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/httpserver"
	"github.com/kai-zer-ru/buhgalter/internal/importexport"
	"github.com/kai-zer-ru/buhgalter/internal/locale"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
	appsched "github.com/kai-zer-ru/buhgalter/internal/scheduler"
	"github.com/kai-zer-ru/buhgalter/internal/setup"
)

var (
	version       = "1.2.1"
	installMethod = "dev"
	buildCommit   = "unknown"
	buildTime     = ""
)

func main() {
	cfg := config.Load(version, installMethod, buildCommit, buildTime)

	if err := locale.Load(cfg.LocalesDir); err != nil {
		log.Fatalf("locales (%s): %v", cfg.LocalesDir, err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	logger, logCloser, err := httpserver.InitLogger(cfg.LogDir, cfg.LogMode)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer logCloser.Close()
	if len(cfg.AllowedHosts) > 0 {
		logger.Info("direct access hosts", "hosts", cfg.AllowedHosts, "env_file", cfg.EnvFilePath)
	}

	manager, err := db.NewManager(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer manager.Close()

	if err := syncAppVersion(manager.DB(), cfg.Version); err != nil {
		log.Fatalf("sync app version: %v", err)
	}

	if err := setup.SyncMarkerFromDB(cfg.DataDir, manager.DB()); err != nil {
		log.Fatalf("setup marker: %v", err)
	}

	if err := bank.SeedIfEmpty(context.Background(), manager.DB()); err != nil {
		log.Fatalf("bank seed: %v", err)
	}
	if affected, err := importexport.RecoverInterruptedJobs(context.Background(), manager.DB()); err != nil {
		log.Fatalf("recover import jobs: %v", err)
	} else if affected > 0 {
		logger.Info("recovered interrupted import jobs", "count", affected)
	}

	auditLogger := audit.New(cfg.LogDir + "/audit")
	backupSvc := &backup.Service{
		Manager:   manager,
		BackupDir: httpserver.BackupDir(cfg.DataDir),
	}
	backupScheduler := backup.NewScheduler(backupSvc, logger)
	backupScheduler.Start()
	defer backupScheduler.Stop()

	creditSched := appsched.New(&appsched.CreditRunner{
		DB: manager.DB(),
		Audit: func(action, userID, login, ip string, details map[string]any) error {
			return auditLogger.Log(action, userID, login, ip, details)
		},
		Logger: logger,
	}, &appsched.RecurringRunner{
		DB:     manager.DB(),
		Logger: logger,
	}, logger)
	creditSched.Start()
	defer creditSched.Stop()

	notifyWorker := notify.NewWorker(manager.DB(), logger)
	notifyWorker.Start()
	defer notifyWorker.Stop()

	srv := httpserver.New(cfg, manager, logger, auditLogger, backupSvc)

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", cfg.Addr, "version", cfg.Version)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "err", err)
	}
	logger.Info("server stopped")
}

func syncAppVersion(db *sql.DB, version string) error {
	if version == "" {
		return nil
	}

	var current sql.NullString
	if err := db.QueryRow(`SELECT app_version FROM system_settings WHERE id = 1`).Scan(&current); err != nil {
		return err
	}
	currentVersion := strings.TrimSpace(current.String)
	if currentVersion == version {
		return nil
	}
	if currentVersion == "" {
		_, err := db.Exec(`
			UPDATE system_settings
			SET app_version = ?, previous_app_version = NULL, updated_at = datetime('now')
			WHERE id = 1`,
			version,
		)
		return err
	}
	_, err := db.Exec(`
		UPDATE system_settings
		SET previous_app_version = ?, app_version = ?, updated_at = datetime('now')
		WHERE id = 1`,
		currentVersion, version,
	)
	return err
}
