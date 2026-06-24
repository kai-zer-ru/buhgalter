package importexport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type Handler struct {
	Store  *db.Handle
	Audit  *audit.Logger
	Logger *slog.Logger
}

func (h *Handler) Headers(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	_, filename, data, err := parseImportRequest(r)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	table, err := ParseFile(filename, data)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"headers": table.Headers})
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
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
	opts.Confirm = false

	report, err := Preview(r.Context(), h.Store.DB(), info.User.ID, filename, data, opts)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
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
	if !opts.Confirm {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_IMPORT_CONFIRM")
		return
	}
	opts.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	report, err := Import(r.Context(), h.Store.DB(), info.User.ID, filename, data, opts)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("import.commit", info.User.ID, info.User.Login, ip, map[string]any{
		"filename":             filename,
		"total_rows":           report.TotalRows,
		"created_transactions": report.CreatedTransactions,
		"skipped_duplicates":   report.SkippedDuplicates,
	})

	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	q := r.URL.Query()
	filters := ExportFilters{
		From:       strings.TrimSpace(q.Get("from")),
		To:         strings.TrimSpace(q.Get("to")),
		AccountID:  strings.TrimSpace(q.Get("account_id")),
		CategoryID: strings.TrimSpace(q.Get("category_id")),
	}

	displayName := info.User.DisplayName
	if displayName == "" {
		displayName = info.User.Login
	}

	data, filename, err := ExportCSV(r.Context(), h.Store.DB(), info.User.ID, displayName, filters)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("export.csv", info.User.ID, info.User.Login, ip, map[string]any{
		"from": filters.From, "to": filters.To, "account_id": filters.AccountID,
	})

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func parseImportRequest(r *http.Request) (ImportOptions, string, []byte, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return ImportOptions{}, "", nil, err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return ImportOptions{}, "", nil, err
	}
	defer file.Close()

	data, err := ReadAll(file, 32<<20)
	if err != nil {
		return ImportOptions{}, "", nil, err
	}

	opts := ImportOptions{
		Preset:          strings.TrimSpace(r.FormValue("preset")),
		Deduplicate:     parseBool(r.FormValue("deduplicate"), true),
		AutoSubcategory: parseBool(r.FormValue("auto_subcategory"), true),
		Confirm:         parseBool(r.FormValue("confirm"), false),
	}
	if opts.Preset == "" {
		opts.Preset = "cubux"
	}
	if cm := strings.TrimSpace(r.FormValue("column_map")); cm != "" {
		if err := json.Unmarshal([]byte(cm), &opts.ColumnMap); err != nil {
			return ImportOptions{}, "", nil, err
		}
	}
	if am := strings.TrimSpace(r.FormValue("account_map")); am != "" {
		if err := json.Unmarshal([]byte(am), &opts.AccountMap); err != nil {
			return ImportOptions{}, "", nil, err
		}
	}
	if cm := strings.TrimSpace(r.FormValue("category_map")); cm != "" {
		if err := json.Unmarshal([]byte(cm), &opts.CategoryMap); err != nil {
			return ImportOptions{}, "", nil, err
		}
	}
	if sm := strings.TrimSpace(r.FormValue("subcategory_map")); sm != "" {
		if err := json.Unmarshal([]byte(sm), &opts.SubcategoryMap); err != nil {
			return ImportOptions{}, "", nil, err
		}
	}
	return opts, header.Filename, data, nil
}

func parseBool(s string, def bool) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
