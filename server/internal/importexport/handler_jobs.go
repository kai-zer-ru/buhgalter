package importexport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apicache"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/realtime"
)

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	opts, filename, data, err := parseImportRequest(r)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	opts.Confirm = true
	opts.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	job, err := createImportJobRecord(r.Context(), h.Store.DB(), info.User.ID, filename)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("import.job.create", info.User.ID, info.User.Login, ip, map[string]any{
		"filename": filename,
		"job_id":   job.ID,
	})

	go h.runImportJob(job.ID, info.User.ID, info.User.Login, ip, filename, data, opts)

	writeJSON(w, http.StatusAccepted, job)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	jobID := strings.TrimSpace(chi.URLParam(r, "id"))
	if jobID == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_JOB_ID")
		return
	}

	job, err := getImportJobRecord(r.Context(), h.Store.DB(), info.User.ID, jobID)
	if isNotFound(err) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if isSQLiteBusyError(err) {
		apperror.WriteR(w, r, http.StatusServiceUnavailable, apperror.ServiceUnavailable)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) runImportJob(
	jobID string,
	userID string,
	login string,
	ip string,
	filename string,
	data []byte,
	opts ImportOptions,
) {
	ctx := context.Background()
	if err := setImportJobRunning(ctx, h.Store.DB(), userID, jobID); err != nil {
		if h.Logger != nil {
			h.Logger.Error("import job set running failed", "job_id", jobID, "err", err)
		}
	}

	report, err := ImportWithProgress(
		ctx,
		h.Store.DB(),
		userID,
		filename,
		data,
		opts,
		func(progress Report) {
			if err := setImportJobProgress(ctx, h.Store.DB(), userID, jobID, progress); err != nil && h.Logger != nil {
				h.Logger.Warn("import job progress update failed", "job_id", jobID, "err", err)
			}
			publishImportProgress(userID, jobID, progress)
		},
	)
	if err != nil {
		_ = setImportJobFailed(ctx, h.Store.DB(), userID, jobID, err)
		errMsg := err.Error()
		realtime.Publish(userID, realtime.NewImportFailed(jobID, errMsg))
		if h.Cache != nil {
			h.Cache.InvalidateUserHints(userID, apicache.LedgerHints())
		}
		_ = h.Audit.Log("import.job.failed", userID, login, ip, map[string]any{
			"filename": filename,
			"job_id":   jobID,
			"error":    errMsg,
		})
		if h.Logger != nil {
			h.Logger.Warn("import job failed", "job_id", jobID, "err", err)
		}
		return
	}

	if err := setImportJobDone(ctx, h.Store.DB(), userID, jobID, report); err != nil {
		if h.Logger != nil {
			h.Logger.Error("import job set done failed", "job_id", jobID, "err", err)
		}
		return
	}
	publishImportDone(userID, jobID, report)
	if h.Cache != nil {
		h.Cache.InvalidateUserHints(userID, apicache.LedgerHints())
	}
	_ = h.Audit.Log("import.job.done", userID, login, ip, map[string]any{
		"filename":             filename,
		"job_id":               jobID,
		"total_rows":           report.TotalRows,
		"created_transactions": report.CreatedTransactions,
		"skipped_duplicates":   report.SkippedDuplicates,
	})
}

func publishImportProgress(userID, jobID string, report Report) {
	raw, err := json.Marshal(report)
	if err != nil {
		return
	}
	realtime.Publish(userID, realtime.NewImportProgress(jobID, raw))
}

func publishImportDone(userID, jobID string, report Report) {
	raw, err := json.Marshal(report)
	if err != nil {
		realtime.Publish(userID, realtime.NewImportDone(jobID, nil))
		return
	}
	realtime.Publish(userID, realtime.NewImportDone(jobID, raw))
}
