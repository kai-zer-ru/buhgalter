package realtime

import "sync"

// NotifyUserDataChanged invalidates server GET cache and publishes to live clients
// when wired from httpserver (scheduler, future activation, etc.).
var (
	notifyMu sync.RWMutex
	notifyFn func(userID string)

	publishMu sync.RWMutex
	publishFn func(userID string, ev Event)
)

// SetUserDataChangedHandler registers the callback used by NotifyUserDataChanged.
func SetUserDataChangedHandler(fn func(userID string)) {
	notifyMu.Lock()
	notifyFn = fn
	notifyMu.Unlock()
}

// NotifyUserDataChanged drops the user's GET cache and pushes an invalidate event.
func NotifyUserDataChanged(userID string) {
	if userID == "" {
		return
	}
	notifyMu.RLock()
	fn := notifyFn
	notifyMu.RUnlock()
	if fn != nil {
		fn(userID)
	}
}

// SetPublisher registers the hub Publish function for typed events (import.*).
func SetPublisher(fn func(userID string, ev Event)) {
	publishMu.Lock()
	publishFn = fn
	publishMu.Unlock()
}

// Publish sends a realtime event to the user's live clients.
func Publish(userID string, ev Event) {
	if userID == "" {
		return
	}
	publishMu.RLock()
	fn := publishFn
	publishMu.RUnlock()
	if fn != nil {
		fn(userID, ev)
	}
}
