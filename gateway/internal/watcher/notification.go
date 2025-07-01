package watcher

import "time"

type Notification struct {
	UserID      string    `json:"user_id"`
	EventType   string    `json:"event_type"` // "error", "success", etc.
	VideoID     string    `json:"video_id,omitempty"`
	ServiceName string    `json:"service_name"` // e.g., "transcoder", "storage"
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}
