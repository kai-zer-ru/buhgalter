package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp  string         `json:"@timestamp"`
	Action     string         `json:"action"`
	ActorID    string         `json:"actor_id,omitempty"`
	ActorLogin string         `json:"actor_login,omitempty"`
	IP         string         `json:"ip,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

type Logger struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) *Logger {
	return &Logger{dir: dir}
}

func (l *Logger) Log(action, actorID, actorLogin, ip string, details map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := Entry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Action:     action,
		ActorID:    actorID,
		ActorLogin: actorLogin,
		IP:         ip,
		Details:    details,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s.jsonl", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(l.dir, name)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}
