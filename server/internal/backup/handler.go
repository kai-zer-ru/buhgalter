package backup

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

type Handler struct {
	Service *Service
	Audit   *audit.Logger
}

type settingsRequest struct {
	BackupEnabled   bool   `json:"backup_enabled"`
	BackupTime      string `json:"backup_time"`
	BackupRetention int    `json:"backup_retention"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	files, err := h.Service.List()
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) DownloadCurrent(w http.ResponseWriter, r *http.Request) {
	path, err := h.Service.LatestPath()
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	h.serveFile(w, r, path, filepath.Base(path))
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	path, err := h.Service.PathFor(filename)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}
	h.serveFile(w, r, path, filename)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, path, filename string) {
	info, _ := auth.FromContext(r.Context())
	ip := auth.ClientIP(r)
	_ = h.Audit.Log("backup.download", info.User.ID, info.User.Login, ip, map[string]any{
		"filename": filename,
	})

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, path)
}

func (h *Handler) PutSettings(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	if req.BackupTime == "" {
		req.BackupTime = "03:00"
	}
	if _, err := time.Parse("15:04", req.BackupTime); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_BACKUP_TIME")
		return
	}

	if err := h.Service.UpdateSettings(req.BackupEnabled, req.BackupTime, req.BackupRetention); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("backup.settings.update", info.User.ID, info.User.Login, ip, map[string]any{
		"backup_enabled":   req.BackupEnabled,
		"backup_time":      req.BackupTime,
		"backup_retention": req.BackupRetention,
	})

	settings, _ := h.Service.GetSettings()
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	name, err := h.Service.Create()
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("backup.download", info.User.ID, info.User.Login, ip, map[string]any{
		"filename": name,
		"manual":   true,
	})

	writeJSON(w, http.StatusCreated, map[string]string{"filename": name})
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_MULTIPART")
		return
	}

	if strings.TrimSpace(r.FormValue("confirm")) != "RESTORE" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RESTORE_CONFIRM")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_FILE_REQUIRED")
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".db") {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DB_FILE_REQUIRED")
		return
	}

	if err := h.Service.Restore(file); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("backup.restore", info.User.ID, info.User.Login, ip, map[string]any{
		"filename": header.Filename,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "restore_complete"})
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Service.GetSettings()
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
