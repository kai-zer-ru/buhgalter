package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

// CreditRunner applies due credit payments for users at local midnight.
type CreditRunner struct {
	DB    *sql.DB
	Audit func(action, userID, login, ip string, details map[string]any) error
	Logger *slog.Logger
}

type Scheduler struct {
	Credit  *CreditRunner
	Logger  *slog.Logger
	stop    chan struct{}
	mu      sync.Mutex
	lastRun map[string]string // userID -> date key
}

func New(creditRunner *CreditRunner, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		Credit:  creditRunner,
		Logger:  logger,
		stop:    make(chan struct{}),
		lastRun: make(map[string]string),
	}
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
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			if s.Credit != nil {
				s.runCreditPayments(now)
			}
		}
	}
}

func (s *Scheduler) runCreditPayments(now time.Time) {
	ctx := context.Background()
	users, err := sqlcdb.New(s.Credit.DB).ListUsersWithTimezone(ctx)
	if err != nil {
		s.Logger.Error("credit scheduler: list users", "err", err)
		return
	}
	for _, u := range users {
		tz := u.Timezone
		if tz == "" {
			tz = "Europe/Moscow"
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue
		}
		local := now.In(loc)
		if local.Hour() != 0 || local.Minute() != 0 {
			continue
		}
		dateKey := local.Format("2006-01-02")
		s.mu.Lock()
		if s.lastRun[u.ID] == dateKey {
			s.mu.Unlock()
			continue
		}
		s.lastRun[u.ID] = dateKey
		s.mu.Unlock()

		cutoff, err := endOfTodayUTC(tz, now)
		if err != nil {
			continue
		}
		applied, err := credit.ApplyDuePayments(ctx, s.Credit.DB, u.ID, cutoff)
		if err != nil {
			s.Logger.Error("credit auto-payment failed", "user_id", u.ID, "err", err)
			continue
		}
		if applied > 0 {
			s.Logger.Info("credit auto-payments applied", "user_id", u.ID, "count", applied)
			if s.Credit.Audit != nil {
				_ = s.Credit.Audit("credit.auto_payment", u.ID, "", "", map[string]any{"count": applied})
			}
		}
	}
}

func endOfTodayUTC(tz string, now time.Time) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	inTZ := now.In(loc)
	year, month, day := inTZ.Date()
	end := time.Date(year, month, day, 23, 59, 59, 0, loc).UTC()
	return timeutil.FormatUTC(end), nil
}
