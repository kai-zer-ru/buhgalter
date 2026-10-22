package realtime

import (
	"encoding/json"
	"time"
)

// Event is a JSON message pushed over the realtime WebSocket.
type Event struct {
	V         int             `json:"v"`
	Type      string          `json:"type"`
	At        string          `json:"at,omitempty"`
	Entities  []string        `json:"entities,omitempty"`
	IDs       []string        `json:"ids,omitempty"`
	HintPaths []string        `json:"hint_paths,omitempty"`
	JobID     string          `json:"job_id,omitempty"`
	Status    string          `json:"status,omitempty"`
	Report    json.RawMessage `json:"report,omitempty"`
	Error     string          `json:"error_message,omitempty"`
}

const (
	TypeHello          = "hello"
	TypePing           = "ping"
	TypeInvalidate     = "invalidate"
	TypeImportProgress = "import.progress"
	TypeImportDone     = "import.done"
	TypeImportFailed   = "import.failed"
)

// NewInvalidate builds an invalidate event.
// Empty hintPaths → coarse client refresh (full clear). Non-empty → path-aware soft-reload.
func NewInvalidate(hintPaths []string, entities []string) Event {
	return Event{
		V:         1,
		Type:      TypeInvalidate,
		At:        time.Now().UTC().Format(time.RFC3339),
		HintPaths: hintPaths,
		Entities:  entities,
	}
}

// NewImportProgress builds a running-job progress event (report is optional JSON).
func NewImportProgress(jobID string, report json.RawMessage) Event {
	return Event{
		V:      1,
		Type:   TypeImportProgress,
		At:     time.Now().UTC().Format(time.RFC3339),
		JobID:  jobID,
		Status: "running",
		Report: report,
	}
}

// NewImportDone builds a successful import completion event.
func NewImportDone(jobID string, report json.RawMessage) Event {
	return Event{
		V:      1,
		Type:   TypeImportDone,
		At:     time.Now().UTC().Format(time.RFC3339),
		JobID:  jobID,
		Status: "done",
		Report: report,
	}
}

// NewImportFailed builds a failed import event.
func NewImportFailed(jobID, errMsg string) Event {
	return Event{
		V:      1,
		Type:   TypeImportFailed,
		At:     time.Now().UTC().Format(time.RFC3339),
		JobID:  jobID,
		Status: "failed",
		Error:  errMsg,
	}
}

func newHello() Event {
	return Event{
		V:    1,
		Type: TypeHello,
		At:   time.Now().UTC().Format(time.RFC3339),
	}
}

func newPing() Event {
	return Event{
		V:    1,
		Type: TypePing,
		At:   time.Now().UTC().Format(time.RFC3339),
	}
}
