package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/admin"
	"github.com/kai-zer-ru/buhgalter/internal/apicache"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/budget"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/docs"
	"github.com/kai-zer-ru/buhgalter/internal/features"
	"github.com/kai-zer-ru/buhgalter/internal/importexport"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/recurring"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
	"github.com/kai-zer-ru/buhgalter/internal/setup"
	"github.com/kai-zer-ru/buhgalter/internal/static"
	"github.com/kai-zer-ru/buhgalter/internal/stats"
	"github.com/kai-zer-ru/buhgalter/internal/subscription"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
	"github.com/kai-zer-ru/buhgalter/internal/transactiontemplate"
	"github.com/kai-zer-ru/buhgalter/internal/ui"
	"github.com/kai-zer-ru/buhgalter/internal/user"
	"github.com/kai-zer-ru/buhgalter/internal/versioncheck"
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
	verboseLogs := s.cfg.LogMode == "dev"
	r.Use(appmw.RequestID)
	r.Use(appmw.RequestIDToContext)
	r.Use(appmw.RejectNoiseProbes)
	r.Use(appmw.Recovery(s.logger))
	r.Use(appmw.Logger(s.logger, verboseLogs))
	r.Use(appmw.CORS(s.cfg.CORSOrigins))
	r.Use(appmw.ExternalAccess(dbHandle, s.cfg.AllowedHosts))
	r.Use(chimw.Compress(5))

	setupHandler := &setup.Handler{DataDir: s.cfg.DataDir, Store: dbHandle, Audit: s.audit, Backup: s.backup}
	loginLimit := 5
	passwordResetLimit := 5
	if os.Getenv("BUHGALTER_E2E") == "1" {
		loginLimit = 1000
		passwordResetLimit = 1000
	}
	loginLimiter := appmw.NewIPRateLimiter(loginLimit, time.Minute)
	passwordResetLimiter := appmw.NewIPRateLimiter(passwordResetLimit, time.Minute)
	authHandler := &auth.Handler{
		Store:                dbHandle,
		Audit:                s.audit,
		Logger:               s.logger,
		LoginLimiter:         loginLimiter,
		PasswordResetLimiter: passwordResetLimiter,
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
	accountArchiveHandler := &accountArchiveHandler{store: dbHandle, audit: s.audit}
	accountDeleteHandler := &accountDeleteHandler{store: dbHandle, audit: s.audit}
	categoryHandler := &category.Handler{Store: dbHandle, Audit: s.audit}
	transactionHandler := &transaction.Handler{Store: dbHandle, Audit: s.audit}
	merchantHandler := &merchant.Handler{Store: dbHandle, Audit: s.audit}
	tagHandler := &tag.Handler{Store: dbHandle, Audit: s.audit}
	templateHandler := &transactiontemplate.Handler{Store: dbHandle, Audit: s.audit}
	debtHandler := &debt.Handler{Store: dbHandle, Audit: s.audit}
	creditHandler := &credit.Handler{Store: dbHandle, Audit: s.audit}
	recurringHandler := &recurring.Handler{Store: dbHandle, Audit: s.audit}
	subscriptionHandler := &subscription.Handler{Store: dbHandle, Audit: s.audit}
	budgetHandler := &budget.Handler{Store: dbHandle, Audit: s.audit}
	importHandler := &importexport.Handler{Store: dbHandle, Audit: s.audit, Logger: s.logger}
	statsHandler := &stats.Handler{Store: dbHandle}
	uiHandler := &ui.Handler{Store: dbHandle, Version: s.cfg.Version}
	versionHandler := &versioncheck.Handler{Checker: versioncheck.NewChecker(s.cfg.Version)}
	apiCache := apicache.New()
	apiCacheMW := apicache.Middleware(apiCache)

	r.Get("/docs", docs.RedocHandler())
	r.Get("/docs/", docs.RedocHandler())
	r.Get("/docs/openapi.yaml", docs.OpenAPIHandler())

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", s.health)
		api.With(apiCacheMW).Get("/setup/status", setupHandler.Status)
		api.With(apiCacheMW).Post("/setup", setupHandler.Setup)
		api.With(apiCacheMW).Post("/setup/restore", setupHandler.Restore)

		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/login", authHandler.Login)
			ar.With(apiCacheMW).Post("/register", authHandler.Register)
			ar.Post("/request-password-reset", authHandler.RequestPasswordReset)
			ar.Get("/verify", authHandler.Verify)
			ar.With(auth.RequireAuth(dbHandle), apiCacheMW).Post("/logout", authHandler.Logout)
			ar.With(auth.RequireAuth(dbHandle), apiCacheMW).Get("/me", authHandler.Me)
		})

		api.With(apiCacheMW).Get("/banks", bankHandler.List)

		api.Group(func(ar chi.Router) {
			ar.Use(auth.RequireAuth(dbHandle))
			ar.Use(apiCacheMW)
			ar.Get("/features", adminHandler.GetFeaturesSnapshot)

			ar.Get("/accounts", accountHandler.List)
			ar.Post("/accounts", accountHandler.Create)
			ar.Get("/accounts/summary", transactionHandler.AccountsSummary)
			ar.Get("/accounts/{id}", accountHandler.Get)
			ar.Get("/accounts/{id}/balance", transactionHandler.AccountBalance)
			ar.Put("/accounts/{id}", accountHandler.Update)
			ar.Put("/accounts/{id}/credit-limit", accountHandler.ChangeCreditLimit)
			ar.Post("/accounts/{id}/archive", accountArchiveHandler.archiveAccount)
			ar.Post("/accounts/{id}/unarchive", accountHandler.Unarchive)
			ar.Post("/accounts/{id}/primary", accountHandler.SetPrimary)
			ar.Delete("/accounts/{id}", accountDeleteHandler.deleteAccount)

			ar.Get("/transactions", transactionHandler.List)
			ar.Post("/transactions", transactionHandler.Create)
			ar.Get("/transactions/{id}", transactionHandler.Get)
			ar.Put("/transactions/{id}", transactionHandler.Update)
			ar.Delete("/transactions/{id}", transactionHandler.Delete)
			ar.Post("/transactions/{id}/activate", transactionHandler.Activate)

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Recurring))
				mod.Get("/recurring-operations", recurringHandler.List)
				mod.Post("/recurring-operations", recurringHandler.Create)
				mod.Put("/recurring-operations/{id}", recurringHandler.Update)
				mod.Delete("/recurring-operations/{id}", recurringHandler.Delete)
				if os.Getenv("BUHGALTER_E2E") == "1" {
					mod.Post("/test/recurring-operations/{id}/run-now", recurringHandler.E2ERunNow)
				}
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Subscriptions))
				mod.Get("/subscriptions", subscriptionHandler.List)
				mod.Get("/subscriptions/summary", subscriptionHandler.Summary)
				mod.Post("/subscriptions/preview-upcoming", subscriptionHandler.PreviewUpcoming)
				mod.Post("/subscriptions", subscriptionHandler.Create)
				mod.Post("/subscriptions/from-recurring/{recurring_id}", subscriptionHandler.ConvertFromRecurring)
				mod.Put("/subscriptions/{id}", subscriptionHandler.Update)
				mod.Delete("/subscriptions/{id}", subscriptionHandler.Delete)
				mod.Get("/subscriptions/{id}/candidate-transactions", subscriptionHandler.Candidates)
				mod.Post("/subscriptions/{id}/attach-transactions", subscriptionHandler.Attach)
				if os.Getenv("BUHGALTER_E2E") == "1" {
					mod.Post("/test/subscriptions/{id}/run-now", subscriptionHandler.E2ERunNow)
				}
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Budget))
				mod.Get("/budgets/summary", budgetHandler.Summary)
				mod.Get("/budgets/spent-preview", budgetHandler.SpentPreview)
				mod.Get("/budgets", budgetHandler.List)
				mod.Post("/budgets/copy-from-previous", budgetHandler.CopyFromPrevious)
				mod.Post("/budgets", budgetHandler.Create)
				mod.Post("/budgets/{id}/copy-next", budgetHandler.CopyNext)
				mod.Patch("/budgets/{id}", budgetHandler.Update)
				mod.Delete("/budgets/{id}", budgetHandler.Delete)
			})

			ar.Post("/transfers", transactionHandler.CreateTransfer)
			ar.Put("/transfers/{group_id}", transactionHandler.UpdateTransfer)
			ar.Delete("/transfers/{group_id}", transactionHandler.DeleteTransfer)

			ar.Get("/dashboard", transactionHandler.Dashboard)
			ar.Get("/version/check", versionHandler.Check)

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.MerchantsTags))
				mod.Get("/merchants", merchantHandler.List)
				mod.Post("/merchants", merchantHandler.Create)
				mod.Get("/merchants/{id}", merchantHandler.Get)
				mod.Put("/merchants/{id}", merchantHandler.Update)
				mod.Delete("/merchants/{id}", merchantHandler.Delete)

				mod.Get("/tags", tagHandler.List)
				mod.Post("/tags", tagHandler.Create)
				mod.Get("/tags/{id}", tagHandler.Get)
				mod.Put("/tags/{id}", tagHandler.Update)
				mod.Delete("/tags/{id}", tagHandler.Delete)
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.TransactionTemplates))
				mod.Get("/transaction-templates", templateHandler.List)
				mod.Post("/transaction-templates", templateHandler.Create)
				mod.Put("/transaction-templates/reorder", templateHandler.Reorder)
				mod.Get("/transaction-templates/{id}", templateHandler.Get)
				mod.Put("/transaction-templates/{id}", templateHandler.Update)
				mod.Delete("/transaction-templates/{id}", templateHandler.Delete)
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Debts))
				mod.Get("/debtors", debtHandler.ListDebtors)
				mod.Post("/debtors", debtHandler.CreateDebtor)
				mod.Get("/debtors/{id}", debtHandler.GetDebtor)
				mod.Put("/debtors/{id}", debtHandler.UpdateDebtor)
				mod.Delete("/debtors/{id}", debtHandler.DeleteDebtor)

				mod.Get("/debts/summary", debtHandler.Summary)
				mod.Get("/debts", debtHandler.ListDebts)
				mod.Post("/debts", debtHandler.CreateDebt)
				mod.Get("/debts/{id}", debtHandler.GetDebt)
				mod.Post("/debts/{id}/settle", debtHandler.SettleDebt)
				mod.Delete("/debts/{id}", debtHandler.DeleteDebt)
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Credits))
				mod.Get("/credits", creditHandler.List)
				mod.Post("/credits", creditHandler.Create)
				mod.Post("/credits/schedule/preview", creditHandler.PreviewSchedule)
				mod.Get("/credits/{id}", creditHandler.Get)
				mod.Put("/credits/{id}", creditHandler.Update)
				mod.Post("/credits/{id}/payments", creditHandler.AddPayment)
				mod.Delete("/credits/{id}/payments/{paymentId}", creditHandler.DeletePayment)
				mod.Patch("/credits/{id}/schedule", creditHandler.UpdateSchedule)
				mod.Post("/credits/{id}/close", creditHandler.Close)
				mod.Delete("/credits/{id}", creditHandler.Delete)
				mod.Get("/credits/{id}/schedule", creditHandler.Schedule)
				if os.Getenv("BUHGALTER_E2E") == "1" {
					mod.Post("/test/credits/{id}/apply-due", creditHandler.E2EApplyDue)
				}
			})

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

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.ImportExport))
				mod.Post("/import/preview", importHandler.Preview)
				mod.Post("/import/headers", importHandler.Headers)
				mod.Post("/import", importHandler.Import)
				mod.Post("/import/jobs", importHandler.CreateJob)
				mod.Get("/import/jobs/{id}", importHandler.GetJob)
				mod.Get("/export", importHandler.Export)
			})

			ar.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Stats))
				mod.Get("/stats/summary", statsHandler.Summary)
				mod.Get("/stats/by-category", statsHandler.ByCategory)
				mod.Get("/stats/by-subcategory", statsHandler.BySubcategory)
				mod.Get("/stats/by-period", statsHandler.ByPeriod)
				mod.Get("/stats/search", statsHandler.Search)
				mod.Get("/stats/context", statsHandler.Context)
			})

			ar.Get("/ui/meta", uiHandler.Meta)
			ar.Get("/ui/i18n/{lang}", uiHandler.I18n)
		})

		api.Route("/user", func(ur chi.Router) {
			ur.Use(auth.RequireAuth(dbHandle))
			ur.Use(apiCacheMW)
			ur.Get("/settings", userHandler.GetSettings)
			ur.Put("/settings", userHandler.PutSettings)
			ur.Group(func(mod chi.Router) {
				mod.Use(features.RequireFeature(dbHandle, features.Notifications))
				mod.Get("/notifications", userHandler.GetNotifications)
				mod.Put("/notifications", userHandler.PutNotifications)
				mod.Post("/notifications/test", userHandler.SendNotificationTest)
				mod.Post("/notifications/templates/preview", userHandler.PreviewNotificationTemplate)
				mod.Post("/notifications/templates/reset", userHandler.ResetNotificationTemplates)
			})
			ur.Put("/password", userHandler.ChangePassword)
			ur.Get("/tokens", userHandler.ListTokens)
			ur.Post("/tokens", userHandler.CreateToken)
			ur.Delete("/tokens/{id}", userHandler.DeleteToken)
		})

		api.Route("/admin", func(ad chi.Router) {
			ad.Use(auth.RequireAuth(dbHandle))
			ad.Use(auth.RequireAdmin)
			ad.Use(apiCacheMW)
			ad.Get("/settings", adminHandler.GetSettings)
			ad.Put("/settings", adminHandler.PutSettings)
			ad.Put("/settings/notification-secret", adminHandler.PutNotificationSecretKey)
			ad.Get("/features", adminHandler.GetFeatures)
			ad.Put("/features", adminHandler.PutFeatures)
			ad.Get("/users", adminHandler.ListUsers)
			ad.Post("/users", adminHandler.CreateUser)
			ad.Put("/users/{id}/status", adminHandler.UpdateUserStatus)
			ad.Put("/users/{id}/password", adminHandler.ResetUserPassword)
			ad.Delete("/users/{id}", adminHandler.DeleteUser)
			ad.Get("/password-reset-requests", adminHandler.ListPasswordResetRequests)
			ad.Post("/password-reset-requests/{id}/ack", adminHandler.AckPasswordResetRequest)
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

	sqlDB := s.manager.DB()
	if sqlDB == nil {
		status = "error"
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	} else if err := sqlDB.PingContext(r.Context()); err != nil {
		status = "error"
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	payload := map[string]string{
		"status":  status,
		"version": s.cfg.Version,
		"db":      dbStatus,
	}
	if sqlDB != nil {
		if externalURL, err := settingscache.ExternalURL(r.Context(), sqlDB); err == nil && externalURL.Valid {
			if u := strings.TrimSpace(externalURL.String); u != "" {
				payload["external_url"] = u
			}
		}
	}

	writeJSON(w, httpStatus, payload)
}

func BackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func InitLogger(logDir, mode string) (*slog.Logger, io.Closer, error) {
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
	level := slog.LevelInfo
	if mode == "dev" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(multi, &slog.HandlerOptions{Level: level}))
	return logger, f, nil
}
