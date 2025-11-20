# Docker Monitor (dockermon)

A powerful Terminal User Interface (TUI) application for monitoring and managing Docker containers in real-time.

![Docker Monitor Screenshot](docs/screenshot.png)

## Features

- **Real-time Container Monitoring**: View all running and stopped containers with live status updates
- **Interactive Container Management**: Start, stop, and restart containers with simple keyboard shortcuts
- **Live Statistics**: Monitor CPU, memory, network I/O, and disk I/O metrics in real-time
- **Visual Performance Graphs**: Track CPU and memory usage trends with embedded graphs
- **Process Monitoring**: View running processes inside containers
- **Container Logs**: Stream and view container logs with auto-scroll functionality
- **Historical Data**: Store and retrieve container statistics with SQLite persistence
- **Intuitive Four-Panel Layout**:
  - Top-left: Container list
  - Top-right: Container statistics
  - Bottom-left: Performance graphs
  - Bottom-right: Container logs

## Prerequisites

- **Docker**: Docker daemon must be running
- **Go**: Version 1.24.0 or later (for building from source)
- **Permissions**: User must have access to Docker socket (`/var/run/docker.sock`)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/rusenback/docker-monitor.git
cd docker-monitor

# Install dependencies
go mod download

# Build the binary
go build -o dockermon ./cmd/dockermon
```

### Using Make

```bash
# Build
make build

# Install dependencies
make deps
```

## Usage

### Running the Application

```bash
# Run the built binary
./dockermon

# Or run directly with Go
go run ./cmd/dockermon

# Or use Make
make run
```

### Keyboard Shortcuts

#### Navigation
- `↑/k` or `j` - Move cursor up
- `↓/j` - Move cursor down
- `PgUp` - Scroll logs up
- `PgDown` - Scroll logs down

#### Container Actions
- `s` - Start selected container
- `x` - Stop selected container
- `r` - Restart selected container

#### View Controls
- `a` - Toggle auto-scroll for logs
- `q` or `Ctrl+C` - Quit application

## Architecture

Docker Monitor follows a clean, layered architecture:

### Project Structure

```
docker-monitor/
├── cmd/
│   └── dockermon/           # Application entry point
│       └── main.go
├── internal/
│   ├── docker/              # Docker API integration layer
│   │   ├── interface.go     # DockerClient interface
│   │   ├── client.go        # Docker client implementation
│   │   ├── container.go     # Container operations
│   │   ├── stats.go         # Real-time stats streaming
│   │   ├── logs.go          # Log streaming
│   │   └── processes.go     # Process monitoring
│   ├── model/               # Domain models
│   │   ├── container.go     # Container data structures
│   │   ├── stats.go         # Statistics models
│   │   ├── logs.go          # Log entry models
│   │   └── process.go       # Process models
│   ├── storage/             # Data persistence layer
│   │   └── sqlite.go        # SQLite storage implementation
│   └── tui/                 # Terminal UI layer (Bubbletea)
│       ├── model.go         # Main TUI model
│       ├── update.go        # Update logic (event handlers)
│       ├── view.go          # View rendering
│       ├── commands.go      # Async command definitions
│       ├── styles.go        # Styling with Lipgloss
│       ├── panels.go        # Panel layout logic
│       ├── stats_view.go    # Statistics panel
│       ├── graph_view.go    # Graph visualization
│       └── logs_view.go     # Logs panel
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Key Design Patterns

- **Model-View-Update (MVU)**: Follows the Elm architecture via Bubbletea
- **Interface-based Design**: Docker operations abstracted behind interfaces for testability
- **Channel-based Streaming**: Real-time data flows through Go channels
- **Message-passing Architecture**: All state changes flow through typed messages
- **Clean Architecture**: Clear separation between UI, business logic, and infrastructure

### Message Flow

The application uses several message types for coordination:
- `tickMsg`: Periodic container list refresh (every 2 seconds)
- `containersMsg`: Updated container list
- `actionMsg`: Container action results (start/stop/restart)
- `statsMsg`: Streaming statistics data
- `logsMsg`: Streaming log entries
- `processMsg`: Container process information

## Technologies

- **[Go](https://golang.org/)** (1.24+) - Core language
- **[Docker SDK](https://github.com/docker/docker)** - Docker API client
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Style definitions and rendering
- **[SQLite](https://modernc.org/sqlite)** - Embedded database for statistics storage

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Or use Make
make test
```

### Building

```bash
# Build binary
go build -o dockermon ./cmd/dockermon

# Or use Make
make build
```

### Cleaning

```bash
# Remove build artifacts
make clean
```

## Configuration

Docker Monitor connects to Docker via Unix socket at `/var/run/docker.sock` by default. The application automatically negotiates the API version with the Docker daemon.

### Docker Permissions

If you encounter permission errors, add your user to the Docker group:

```bash
sudo usermod -aG docker $USER
```

Log out and log back in for the changes to take effect.

## Performance Considerations

- **Stats Streaming**: Only active for running containers
- **Log Buffer**: Limited to 1000 entries to prevent memory issues
- **Auto-refresh**: Container list refreshes every 2 seconds
- **Lazy Loading**: Stats and logs only stream for the selected container

## Troubleshooting

### Docker Connection Failed

```
❌ Failed to connect to Docker: Cannot connect to the Docker daemon
```

**Solution**: Ensure Docker daemon is running:

```bash
sudo systemctl start docker
sudo systemctl enable docker  # Start on boot
```

### Permission Denied

```
❌ Failed to connect to Docker: permission denied
```

**Solution**: Add your user to the Docker group:

```bash
sudo usermod -aG docker $USER
```

Then log out and back in.

### Stats Not Showing

Stats only display for **running** containers. If a container is stopped, the stats panel will be empty.

## Roadmap

### Core Features
- [ ] **Multi-container selection**: Select and perform actions on multiple containers simultaneously
- [ ] **Container search/filter**: Filter containers by name, image, or status
- [ ] **Exec into container**: Open an interactive shell inside a running container
- [ ] **Container inspect**: View detailed container configuration and metadata
- [ ] **Image management**: List, pull, and remove Docker images

### Monitoring & Visualization
- [ ] **Time range selector**: Switch between different time ranges (30min, 1h, 6h, 1d, 1w) for graphs
- [ ] **Export statistics**: Export historical data to CSV or JSON
- [ ] **Network graph**: Visualize network I/O trends over time
- [ ] **Disk I/O graph**: Track block read/write trends
- [ ] **Alert thresholds**: Configure alerts for high CPU/memory usage
- [ ] **Container comparison**: Compare stats between multiple containers side-by-side

### Docker Compose & Networks
- [ ] **Docker Compose support**: Group and manage containers by compose projects
- [ ] **Network visualization**: Display container network topology
- [ ] **Volume management**: List and inspect Docker volumes
- [ ] **Port mapping view**: Enhanced visualization of exposed ports

### UI/UX Enhancements
- [ ] **Themes**: Multiple color schemes (dark, light, custom)
- [ ] **Customizable layout**: Resizable panels and layout presets
- [ ] **Tabs/workspaces**: Multiple views for different container groups
- [ ] **Help overlay**: In-app keyboard shortcut reference (press `?`)
- [ ] **Status bar**: Show system-wide Docker stats (total containers, images, volumes)

### Advanced Features
- [ ] **Remote Docker hosts**: Connect to Docker daemons on remote machines
- [ ] **Container creation**: Create new containers from TUI
- [ **Resource limits**: Set CPU/memory limits on containers
- [ ] **Log filtering**: Search and filter logs with regex patterns
- [ ] **Log export**: Save logs to file
- [ ] **Stats export to Prometheus**: Export metrics for external monitoring

### Performance & Reliability
- [ ] **Configurable refresh rate**: Adjust polling frequency
- [ ] **Offline mode**: View historical data when Docker is unavailable
- [ ] **Data retention policy**: Configurable storage cleanup settings
- [ ] **Performance optimization**: Reduce resource usage for large deployments

### Integration
- [ ] **Configuration file**: YAML/TOML config for settings persistence
- [ ] **Plugin system**: Extensibility for custom features
- [ ] **GitHub Actions monitoring**: Monitor containers in CI/CD environments

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

If you'd like to work on a roadmap item, please open an issue first to discuss the implementation approach.

## License

This project is open source and available under the [MIT License](LICENSE).

## Acknowledgments

- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) by Charm
- Styled with [Lipgloss](https://github.com/charmbracelet/lipgloss) by Charm
- Uses the official [Docker SDK for Go](https://github.com/docker/docker)

## Author

Created by [rusenback](https://github.com/rusenback)
