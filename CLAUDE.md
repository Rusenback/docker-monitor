# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Docker Monitor (dockermon) is a TUI-based (Terminal User Interface) Docker container management and monitoring application built with Go, using Bubbletea for the interface and Lipgloss for styling.

## Build and Run Commands

```bash
# Build the application
go build -o dockermon ./cmd/dockermon
# Or use make
make build

# Run the application
./dockermon
# Or run directly with go
go run ./cmd/dockermon
# Or use make
make run

# Run tests
go test ./...
# Or use make
make test

# Install/update dependencies
go mod download
go mod tidy
# Or use make
make deps

# Clean build artifacts
make clean
```

## Architecture

### High-Level Structure

The application follows a clean architecture pattern with three main layers:

1. **TUI Layer** (`internal/tui/`): Implements the Bubbletea Model-View-Update pattern
   - `app.go` contains the main Model struct and implements Init/Update/View methods
   - Manages four-panel layout: Container List (top-left), Stats (top-right), Graph (bottom-left), Logs (bottom-right)
   - Handles keyboard input: arrow keys/j/k for navigation, s/x/r for container actions, PgUp/PgDown for log scrolling

2. **Docker Integration Layer** (`internal/docker/`): Abstracts Docker API operations
   - `interface.go` defines the `DockerClient` interface for testability
   - `client.go` provides the Docker client implementation
   - `container.go` handles container operations (list, start, stop, restart)
   - `stats.go` implements real-time stats streaming
   - `logs.go` implements real-time log streaming

3. **Model Layer** (`internal/model/`): Domain models
   - `container.go`: Container and Port structs
   - `stats.go`: Stats struct for CPU, memory, network, and disk metrics
   - `logs.go`: LogEntry struct for container logs

### Key Design Patterns

- **Interface-based Docker client**: The `DockerClient` interface allows for easy mocking in tests and decouples the TUI from Docker SDK implementation
- **Channel-based streaming**: Stats and logs use Go channels for real-time updates from Docker API, with cancel functions for cleanup
- **Bubbletea commands**: Async operations (container actions, fetching data) are modeled as Bubbletea commands that return messages
- **Message-passing architecture**: All state changes flow through the Update function via typed messages (tickMsg, containersMsg, actionMsg, statsMsg, logsMsg)

### TUI Message Flow

The TUI uses several message types to coordinate async operations:
- `tickMsg`: Triggers periodic container list refresh (every 2 seconds)
- `containersMsg`: Returns updated container list
- `actionMsg`: Returns result of container actions (start/stop/restart)
- `statsMsg`: Returns streaming stats data for selected container
- `logsMsg`: Returns streaming log entries for selected container

When the cursor changes, `updateStatsAndLogsForCursor()` cancels previous streams and starts new ones for the selected container.

### Docker Client Configuration

The Docker client connects via Unix socket (`/var/run/docker.sock` by default) and negotiates API version automatically. Configuration is in `internal/docker/client.go` with `DefaultConfig()` providing sensible defaults.

## Development Notes

- The application requires Docker to be running and accessible via the Docker socket
- Stats and logs streaming only work for running containers
- The TUI maintains a 60/40 split layout (left/right and top/bottom)
- Log buffer is limited to 1000 entries to prevent memory issues
- Auto-scroll can be toggled with 'a' key and is automatically disabled when manually scrolling up
