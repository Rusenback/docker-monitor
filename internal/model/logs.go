// internal/model/logs.go
package model

import "time"

// LogEntry represents a single log line from a container
type LogEntry struct {
	Timestamp time.Time
	Message   string
	Stream    string // "stdout" or "stderr"
}
