// internal/docker/logs.go
package docker

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rusenback/docker-monitor/internal/model"
)

// GetContainerLogs retrieves container logs
func (c *Client) GetContainerLogs(id string, tail int) ([]model.LogEntry, error) {
	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       strconv.Itoa(tail), // Get last N lines
	}

	reader, err := c.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return parseLogStream(reader)
}

// StreamContainerLogs streams container logs in real-time
func (c *Client) StreamContainerLogs(id string) (<-chan model.LogEntry, <-chan error, func()) {
	logsChan := make(chan model.LogEntry)
	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(c.Ctx)

	go func() {
		defer close(logsChan)
		defer close(errChan)

		options := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     true, // Stream logs continuously
			Tail:       "10", // Start with last 10 lines
		}

		reader, err := c.cli.ContainerLogs(ctx, id, options)
		if err != nil {
			errChan <- err
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		// Increase buffer size for long log lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			entry, valid := parseLogLine(line)
			if !valid {
				continue // Skip empty or invalid lines
			}

			select {
			case logsChan <- entry:
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF && err != context.Canceled {
			errChan <- err
		}
	}()

	return logsChan, errChan, cancel
}

// parseLogStream parses a log stream into a slice of LogEntry
func parseLogStream(reader io.Reader) ([]model.LogEntry, error) {
	var entries []model.LogEntry
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		entry, valid := parseLogLine(line)
		if valid {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return entries, err
	}

	return entries, nil
}

// parseLogLine parses a single log line
// Returns an entry and a boolean indicating if the entry is valid
func parseLogLine(line string) (model.LogEntry, bool) {
	// Docker log format: [8]byte header + timestamp + message
	// Remove Docker's 8-byte header if present (stdout/stderr indicator)
	if len(line) > 8 {
		line = line[8:]
	}

	// Trim whitespace and check if line is empty
	line = strings.TrimSpace(line)
	if line == "" {
		return model.LogEntry{}, false
	}

	entry := model.LogEntry{
		Timestamp: time.Now(),
		Message:   line,
		Stream:    "stdout",
	}

	// Try to parse timestamp from line
	// Format: 2024-01-15T10:30:45.123456789Z message
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 2 {
		if timestamp, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			entry.Timestamp = timestamp
			entry.Message = strings.TrimSpace(parts[1])

			// If message is empty after parsing timestamp, skip it
			if entry.Message == "" {
				return model.LogEntry{}, false
			}
		}
	}

	// Detect stream type from content or color codes
	if strings.Contains(strings.ToLower(line), "error") ||
		strings.Contains(strings.ToLower(line), "fatal") {
		entry.Stream = "stderr"
	}

	return entry, true
}
