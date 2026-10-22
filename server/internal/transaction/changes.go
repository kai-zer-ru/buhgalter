package transaction

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const (
	changesDefaultLimit = 200
	changesMaxLimit     = 500
)

// TransactionChange is one collapsed mutation for Android incremental sync.
type TransactionChange struct {
	ID          int64        `json:"id"`
	Action      string       `json:"action"`
	EntityID    string       `json:"entity_id"`
	OccurredAt  string       `json:"occurred_at"`
	Transaction *Transaction `json:"transaction,omitempty"`
}

// ChangesResult is GET /sync/transaction-changes.
type ChangesResult struct {
	ServerTime string              `json:"server_time"`
	SinceID    int64               `json:"since_id"`
	HasMore    bool                `json:"has_more"`
	Changes    []TransactionChange `json:"changes"`
}

func maxUserChangeEventID(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case nil:
		return 0, nil
	default:
		return 0, errors.New("unexpected max change id type")
	}
}

func clampChangesLimit(limit int) int {
	if limit <= 0 {
		return changesDefaultLimit
	}
	if limit > changesMaxLimit {
		return changesMaxLimit
	}
	return limit
}

// ListChanges returns operations created/updated/deleted for the user after a cursor.
func ListChanges(ctx context.Context, db *sql.DB, userID string, afterID int64, since string, limit int) (ChangesResult, error) {
	limit = clampChangesLimit(limit)
	if afterID < 0 {
		afterID = 0
	}
	rows, err := queries(db).ListUserChangeEventsAfter(ctx, sqlcdb.ListUserChangeEventsAfterParams{
		UserID:  userID,
		AfterID: afterID,
		Since:   since,
		Limit:   int64(limit + 1),
	})
	if err != nil {
		return ChangesResult{}, err
	}

	result := ChangesResult{
		ServerTime: timeutil.FormatUTC(timeutil.NowUTC()),
		SinceID:    afterID,
		Changes:    []TransactionChange{},
	}
	if len(rows) == 0 {
		return result, nil
	}
	if len(rows) > limit {
		result.HasMore = true
		rows = rows[:limit]
	}
	result.SinceID = rows[len(rows)-1].ID

	latest := make(map[string]sqlcdb.UserChangeEvent, len(rows))
	order := make([]string, 0, len(rows))
	for _, row := range rows {
		if _, seen := latest[row.EntityID]; !seen {
			order = append(order, row.EntityID)
		}
		latest[row.EntityID] = row
	}

	for _, entityID := range order {
		row := latest[entityID]
		change := TransactionChange{
			ID:         row.ID,
			EntityID:   row.EntityID,
			OccurredAt: row.OccurredAt,
			Action:     "deleted",
		}
		if row.Action != "deleted" {
			tx, getErr := GetByID(ctx, db, userID, entityID)
			if getErr == nil {
				change.Action = "upsert"
				change.Transaction = &tx
			} else if !errors.Is(getErr, ErrNotFound) {
				return ChangesResult{}, getErr
			}
		}
		result.Changes = append(result.Changes, change)
	}
	return result, nil
}

func (h *Handler) Changes(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	q := r.URL.Query()
	if q.Get("watermark") == "1" {
		maxRaw, err := queries(h.Store.DB()).MaxUserChangeEventID(r.Context(), info.User.ID)
		if err != nil {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
			return
		}
		maxID, err := maxUserChangeEventID(maxRaw)
		if err != nil {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
			return
		}
		writeJSON(w, http.StatusOK, ChangesResult{
			ServerTime: timeutil.FormatUTC(timeutil.NowUTC()),
			SinceID:    maxID,
			HasMore:    false,
			Changes:    []TransactionChange{},
		})
		return
	}
	afterID, _ := strconv.ParseInt(q.Get("since_id"), 10, 64)
	if afterID < 0 {
		afterID = 0
	}
	since := strings.TrimSpace(q.Get("since"))
	if since != "" {
		t, err := timeutil.ParseFlexibleUTC(since)
		if err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_PERIOD_DATE")
			return
		}
		since = timeutil.FormatUTC(t)
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	result, err := ListChanges(r.Context(), h.Store.DB(), info.User.ID, afterID, since, limit)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
