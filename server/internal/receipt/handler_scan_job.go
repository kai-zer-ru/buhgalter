package receipt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/qrcode"
)

func scanErrorCode(err error, stage string) error {
	if err == nil {
		return nil
	}
	switch stage {
	case "qr":
		if errors.Is(err, qrcode.ErrNotFound) {
			return errors.New("ERR_RECEIPT_QR")
		}
	case "fns":
		switch {
		case errors.Is(err, fns.ErrInvalidQR):
			return errors.New("ERR_RECEIPT_QR")
		case errors.Is(err, fns.ErrNotFound):
			return errors.New("ERR_RECEIPT_NOT_FOUND")
		case errors.Is(err, fns.ErrUnavailable), errors.Is(err, fns.ErrFetchFailed):
			return errors.New("ERR_FNS_UNAVAILABLE")
		}
	}
	return errors.New("ERR_RECEIPT_SCAN")
}

const maxReceiptImageBytes = 16 << 20

func (h *Handler) CreateScanJob(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	if err := r.ParseMultipartForm(maxReceiptImageBytes); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_MULTIPART")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_IMAGE")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxReceiptImageBytes+1))
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_IMAGE")
		return
	}
	if len(data) == 0 || len(data) > maxReceiptImageBytes {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RECEIPT_IMAGE")
		return
	}

	job, err := createScanJobRecord(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	filename := ""
	if header != nil {
		filename = header.Filename
	}
	_ = h.Audit.Log("receipt.scan.create", info.User.ID, info.User.Login, auth.ClientIP(r), map[string]any{
		"job_id":   job.ID,
		"filename": filename,
	})

	go h.runScanJob(job.ID, info.User.ID, info.User.Login, auth.ClientIP(r), data)

	writeJSON(w, http.StatusAccepted, job)
}

func (h *Handler) GetScanJob(w http.ResponseWriter, r *http.Request) {
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
	job, err := getScanJobRecord(r.Context(), h.Store.DB(), info.User.ID, jobID)
	if isScanJobNotFound(err) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) runScanJob(jobID, userID, login, ip string, imageData []byte) {
	ctx := context.Background()
	if err := setScanJobRunning(ctx, h.Store.DB(), userID, jobID); err != nil && h.Logger != nil {
		h.Logger.Error("receipt scan set running failed", "job_id", jobID, "err", err)
	}

	qr, err := qrcode.DecodeFromImage(imageData)
	if err != nil {
		_ = setScanJobFailed(ctx, h.Store.DB(), userID, jobID, scanErrorCode(err, "qr"))
		_ = h.Audit.Log("receipt.scan.failed", userID, login, ip, map[string]any{
			"job_id": jobID,
			"stage":  "qr",
			"error":  err.Error(),
		})
		if h.Logger != nil {
			h.Logger.Warn("receipt scan qr failed", "job_id", jobID, "err", err)
		}
		return
	}

	result, err := Parse(ctx, h.fetcher(), qr)
	if err != nil {
		_ = setScanJobFailed(ctx, h.Store.DB(), userID, jobID, scanErrorCode(err, "fns"))
		_ = h.Audit.Log("receipt.scan.failed", userID, login, ip, map[string]any{
			"job_id": jobID,
			"stage":  "fns",
			"error":  err.Error(),
		})
		if h.Logger != nil {
			h.Logger.Warn("receipt scan fns failed", "job_id", jobID, "err", err)
		}
		return
	}

	if err := setScanJobDone(ctx, h.Store.DB(), userID, jobID, result); err != nil && h.Logger != nil {
		h.Logger.Error("receipt scan set done failed", "job_id", jobID, "err", err)
	}
	_ = h.Audit.Log("receipt.scan.done", userID, login, ip, map[string]any{
		"job_id": jobID,
		"store":  result.StoreName,
	})
}
