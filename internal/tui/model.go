package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusenback/docker-monitor/internal/docker"
	"github.com/rusenback/docker-monitor/internal/model"
	"github.com/rusenback/docker-monitor/internal/storage"
)

// Model represents the TUI application state
type Model struct {
	client        docker.DockerClient
	containers    []model.Container
	cursor        int
	err           error
	loading       bool
	message       string
	currentStats     *model.Stats
	previousStats    *model.Stats // For calculating rates
	currentProcesses []model.Process
	statsCancel      func()
	width         int
	height        int

	logs           []model.LogEntry
	logsCancel     func()
	logsScroll     int
	logsAutoScroll bool

	logsChan    <-chan model.LogEntry
	logsErrChan <-chan error

	statsChan    <-chan *model.Stats
	statsErrChan <-chan error

	currentContainerID string // Track current container to avoid resetting logs unnecessarily

	// Historical data for graphs (deprecated - now using storage)
	cpuHistory    []float64
	memoryHistory []float64
	maxDataPoints int

	// Storage and time range
	storage   *storage.Storage
	timeRange storage.TimeRange
}

// Message types for Bubbletea update loop
type tickMsg time.Time

type containersMsg struct {
	containers []model.Container
	err        error
}

type actionMsg struct {
	message string
	err     error
}

type statsMsg struct {
	stats *model.Stats
	err   error
}

type logsMsg struct {
	entry model.LogEntry
	err   error
}

// NewModel creates a new TUI model
func NewModel(client docker.DockerClient, store *storage.Storage) Model {
	maxPoints := 150
	// Pre-fill with zeros so graph is full-width from the start
	cpuHist := make([]float64, maxPoints)
	memHist := make([]float64, maxPoints)

	return Model{
		client:        client,
		loading:       true,
		maxDataPoints: maxPoints,
		cpuHistory:    cpuHist,
		memoryHistory: memHist,
		storage:       store,
		timeRange:     storage.Range30Min, // Default to 30 minutes
	}
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchContainers(m.client), tickCmd())
}
