package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/admin"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/docs"
	"github.com/kai-zer-ru/buhgalter/internal/importexport"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/setup"
	"github.com/kai-zer-ru/buhgalter/internal/static"
	"github.com/kai-zer-ru/buhgalter/internal/stats"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
	"github.com/kai-zer-ru/buhgalter/internal/user"
)

type Server struct {
	cfg     config.Config
	manager *db.Manager
	logger  *slog.Logger
	audit   *audit.Logger
	backup  *backup.Service
}

func New(cfg config.Config, manager *db.Manager, logger *slog.Logger, auditLogger *audit.Logger, backupSvc *backup.Service) *Server {
	return &Server{cfg: cfg, manager: manager, logger: logger, audit: auditLogger, backup: backupSvc}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	dbHandle := db.NewHandle(s.manager)
	r.Use(appmw.RequestID)
	r.Use(appmw.RequestIDToContext)
	r.Use(appmw.Recovery(s.logger))
	r.Use(appmw.Logger(s.logger))
	r.Use(appmw.CORS(s.cfg.CORSOrigins))
	r.Use(appmw.ExternalAccess(dbHandle))
	r.Use(chimw.Compress(5))

	setupHandler := &setup.Handler{DataDir: s.cfg.DataDir, Store: dbHandle, Audit: s.audit}
	loginLimiter := appmw.NewIPRateLimiter(5, time.Minute)
	authHandler := &auth.Handler{
		Store:        dbHandle,
		Audit:        s.audit,
		Logger:       s.logger,
		LoginLimiter: loginLimiter,
	}
	userHandler := &user.Handler{Store: dbHandle, Audit: s.audit}
	adminHandler := &admin.Handler{
		Store:   dbHandle,
		Audit:   s.audit,
		Config:  s.cfg,
		Started: time.Now(),
	}
	backupHandler := &backup.Handler{Service: s.backup, Audit: s.audit}
	bankHandler := &bank.Handler{Store: dbHandle}
	accountHandler := &account.Handler{Store: dbHandle, Audit: s.audit}
	categoryHandler := &category.Handler{Store: dbHandle, Audit: s.audit}
	transactionHandler := &transaction.Handler{Store: dbHandle, Audit: s.audit}
	debtHandler := &debt.Handler{Store: dbHandle, Audit: s.audit}
	creditHandler := &credit.Handler{Store: dbHandle, Audit: s.audit}
	importHandler := &importexport.Handler{Store: dbHandle, Audit: s.audit, Logger: s.logger}
	statsHandler := &stats.Handler{Store: dbHandle}

	r.Get("/docs", docs.RedocHandler())
	r.Get("/docs/", docs.RedocHandler())
	r.Get("/docs/openapi.yaml", docs.OpenAPIHandler())

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", s.health)
		api.Get("/setup/status", setupHandler.Status)
		api.Post("/setup", setupHandler.Setup)

		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/login", authHandler.Login)
			ar.Post("/register", authHandler.Register)
			ar.Get("/verify", authHandler.Verify)
			ar.With(auth.RequireAuth(dbHandle)).Post("/logout", authHandler.Logout)
			ar.With(auth.RequireAuth(dbHandle)).Get("/me", authHandler.Me)
		})

		api.Get("/banks", bankHandler.List)

		api.Group(func(ar chi.Router) {
			ar.Use(auth.RequireAuth(dbHandle))
			ar.Get("/accounts", accountHandler.List)
			ar.Post("/accounts", accountHandler.Create)
			ar.Get("/accounts/summary", transactionHandler.AccountsSummary)
			ar.Get("/accounts/{id}", accountHandler.Get)
			ar.Get("/accounts/{id}/balance", transactionHandler.AccountBalance)
			ar.Put("/accounts/{id}", accountHandler.Update)
			ar.Post("/accounts/{id}/archive", accountHandler.Archive)
			ar.Post("/accounts/{id}/unarchive", accountHandler.Unarchive)
			ar.Post("/accounts/{id}/primary", accountHandler.SetPrimary)
			ar.Delete("/accounts/{id}", accountHandler.Delete)

			ar.Get("/transactions", transactionHandler.List)
			ar.Post("/transactions", transactionHandler.Create)
			ar.Get("/transactions/{id}", transactionHandler.Get)
			ar.Put("/transactions/{id}", transactionHandler.Update)
			ar.Delete("/transactions/{id}", transactionHandler.Delete)
			ar.Post("/transactions/{id}/activate", transactionHandler.Activate)

			ar.Post("/transfers", transactionHandler.CreateTransfer)
			ar.Put("/transfers/{group_id}", transactionHandler.UpdateTransfer)
			ar.Delete("/transfers/{group_id}", transactionHandler.DeleteTransfer)

			ar.Get("/dashboard", transactionHandler.Dashboard)

			ar.Get("/debtors", debtHandler.ListDebtors)
			ar.Post("/debtors", debtHandler.CreateDebtor)
			ar.Get("/debtors/{id}", debtHandler.GetDebtor)
			ar.Put("/debtors/{id}", debtHandler.UpdateDebtor)
			ar.Delete("/debtors/{id}", debtHandler.DeleteDebtor)

			ar.Get("/debts/summary", debtHandler.Summary)
			ar.Get("/debts", debtHandler.ListDebts)
			ar.Post("/debts", debtHandler.CreateDebt)
			ar.Get("/debts/{id}", debtHandler.GetDebt)
			ar.Post("/debts/{id}/settle", debtHandler.SettleDebt)
			ar.Delete("/debts/{id}", debtHandler.DeleteDebt)

			ar.Get("/credits", creditHandler.List)
			ar.Post("/credits", creditHandler.Create)
			ar.Post("/credits/schedule/preview", creditHandler.PreviewSchedule)
			ar.Get("/credits/{id}", creditHandler.Get)
			ar.Put("/credits/{id}", creditHandler.Update)
			ar.Post("/credits/{id}/payments", creditHandler.AddPayment)
			ar.Delete("/credits/{id}/payments/{paymentId}", creditHandler.DeletePayment)
			ar.Post("/credits/{id}/close", creditHandler.Close)
			ar.Delete("/credits/{id}", creditHandler.Delete)
			ar.Get("/credits/{id}/schedule", creditHandler.Schedule)

			ar.Get("/categories", categoryHandler.List)
			ar.Post("/categories", categoryHandler.Create)
			ar.Put("/categories/order", categoryHandler.Reorder)
			ar.Put("/categories/{id}", categoryHandler.Update)
			ar.Delete("/categories/{id}", categoryHandler.Delete)
			ar.Post("/categories/{id}/primary", categoryHandler.SetPrimary)
			ar.Get("/categories/{id}/subcategories", categoryHandler.ListSubcategories)
			ar.Put("/categories/{id}/subcategories/order", categoryHandler.ReorderSubcategories)
			ar.Post("/categories/{id}/subcategories", categoryHandler.CreateSubcategory)

			ar.Put("/subcategories/{id}", categoryHandler.UpdateSubcategory)
			ar.Delete("/subcategories/{id}", categoryHandler.DeleteSubcategory)

			ar.Post("/import/preview", importHandler.Preview)
			ar.Post("/import/headers", importHandler.Headers)
			ar.Post("/import", importHandler.Import)
			ar.Post("/import/jobs", importHandler.CreateJob)
			ar.Get("/import/jobs/{id}", importHandler.GetJob)
			ar.Get("/export", importHandler.Export)
			ar.Get("/stats/summary", statsHandler.Summary)
			ar.Get("/stats/by-category", statsHandler.ByCategory)
			ar.Get("/stats/by-subcategory", statsHandler.BySubcategory)
			ar.Get("/stats/by-period", statsHandler.ByPeriod)
			ar.Get("/stats/search", statsHandler.Search)
			ar.Get("/stats/context", statsHandler.Context)
		})

		api.Route("/user", func(ur chi.Router) {
			ur.Use(auth.RequireAuth(dbHandle))
			ur.Get("/settings", userHandler.GetSettings)
			ur.Put("/settings", userHandler.PutSettings)
			ur.Get("/notifications", userHandler.GetNotifications)
			ur.Put("/notifications", userHandler.PutNotifications)
			ur.Post("/notifications/test", userHandler.SendNotificationTest)
			ur.Post("/notifications/templates/preview", userHandler.PreviewNotificationTemplate)
			ur.Post("/notifications/templates/reset", userHandler.ResetNotificationTemplates)
			ur.Put("/password", userHandler.ChangePassword)
			ur.Get("/tokens", userHandler.ListTokens)
			ur.Post("/tokens", userHandler.CreateToken)
			ur.Delete("/tokens/{id}", userHandler.DeleteToken)
		})

		api.Route("/admin", func(ad chi.Router) {
			ad.Use(auth.RequireAuth(dbHandle))
			ad.Use(auth.RequireAdmin)
			ad.Get("/settings", adminHandler.GetSettings)
			ad.Put("/settings", adminHandler.PutSettings)
			ad.Put("/settings/notification-secret", adminHandler.PutNotificationSecretKey)
			ad.Get("/users", adminHandler.ListUsers)
			ad.Post("/users", adminHandler.CreateUser)
			ad.Delete("/users/{id}", adminHandler.DeleteUser)
			ad.Get("/diagnostics", adminHandler.GetDiagnostics)

			ad.Get("/backups", backupHandler.List)
			ad.Get("/backups/settings", backupHandler.GetSettings)
			ad.Put("/backups/settings", backupHandler.PutSettings)
			ad.Post("/backups/run", backupHandler.Run)
			ad.Post("/backups/restore", backupHandler.Restore)
			ad.Get("/backups/download", backupHandler.DownloadCurrent)
			ad.Get("/backups/{filename}/download", backupHandler.Download)
		})
	})

	if s.cfg.StaticEmbed {
		r.Handle("/*", static.Handler())
	}

	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "connected"
	httpStatus := http.StatusOK

	if sqlDB := s.manager.DB(); sqlDB == nil {
		status = "error"
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	} else if err := sqlDB.PingContext(r.Context()); err != nil {
		status = "error"
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, map[string]string{
		"status":  status,
		"version": s.cfg.Version,
		"db":      dbStatus,
	})
}

func BackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func InitLogger(logDir string) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(filepath.Join(logDir, "audit"), 0o755); err != nil {
		return nil, nil, err
	}

	logPath := filepath.Join(logDir, "app.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	multi := io.MultiWriter(os.Stdout, f)
	logger := slog.New(slog.NewJSONHandler(multi, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return logger, f, nil
}
