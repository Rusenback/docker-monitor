package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/docker"
	"github.com/rusenback/docker-monitor/internal/model"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B4BEFE"))
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(lipgloss.Color("#CBA6F7"))
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(lipgloss.Color("#89B4FA"))
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	stoppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6ADC8")).Padding(1, 0)
	panelStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#585B70")).
			Padding(1, 2)
)

type Model struct {
	client        docker.DockerClient
	containers    []model.Container
	cursor        int
	err           error
	loading       bool
	message       string
	currentStats  *model.Stats
	previousStats *model.Stats // For calculating rates
	statsCancel   func()
	width         int
	height        int
}

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

func NewModel(client docker.DockerClient) Model {
	return Model{client: client, loading: true}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchContainers(m.client), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func fetchContainers(client docker.DockerClient) tea.Cmd {
	return func() tea.Msg {
		containers, err := client.ListContainers()
		for i := range containers {
			c := &containers[i]
			switch c.State {
			case "running":
				c.DisplayStatus = truncate(c.Status, 30)
			case "exited":
				c.DisplayStatus = "exited"
			default:
				c.DisplayStatus = c.State
			}
		}
		return containersMsg{containers: containers, err: err}
	}
}

func waitForStats(statsChan <-chan *model.Stats, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case stats := <-statsChan:
			return statsMsg{stats: stats, err: nil}
		case err := <-errChan:
			return statsMsg{stats: nil, err: err}
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.statsCancel != nil {
				m.statsCancel()
			}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.updateStatsForCursor()
			}
		case "down", "j":
			if m.cursor < len(m.containers)-1 {
				m.cursor++
				return m, m.updateStatsForCursor()
			}
		case "s":
			if len(m.containers) > 0 {
				return m, startContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}
		case "x":
			if len(m.containers) > 0 {
				return m, stopContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}
		case "r":
			if len(m.containers) > 0 {
				return m, restartContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}
		case "R":
			m.loading = true
			m.message = "Refreshing..."
			return m, fetchContainers(m.client)
		}
	case tickMsg:
		return m, tea.Batch(fetchContainers(m.client), tickCmd())
	case containersMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.containers = msg.containers
			if m.cursor >= len(m.containers) && len(m.containers) > 0 {
				m.cursor = len(m.containers) - 1
			}
			return m, m.updateStatsForCursor()
		}
	case statsMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Stats error: %v", msg.err)
		} else {
			m.currentStats = msg.stats
			m.message = ""
		}
	case actionMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.message = msg.message
		}
		return m, fetchContainers(m.client)
	}
	return m, nil
}

func (m *Model) updateStatsForCursor() tea.Cmd {
	if len(m.containers) == 0 {
		return nil
	}
	container := m.containers[m.cursor]
	if container.State == "running" {
		statsChan, errChan, cancel := m.client.StreamContainerStats(container.ID)
		m.statsCancel = cancel
		return waitForStats(statsChan, errChan)
	} else {
		if m.statsCancel != nil {
			m.statsCancel()
			m.statsCancel = nil
		}
		m.currentStats = nil
		return nil
	}
}

func startContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{message: fmt.Sprintf("Started: %s", name), err: client.StartContainer(id)}
	}
}
func stopContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{message: fmt.Sprintf("Stopped: %s", name), err: client.StopContainer(id)}
	}
}
func restartContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{
			message: fmt.Sprintf("Restarted: %s", name),
			err:     client.RestartContainer(id),
		}
	}
}

func (m Model) View() string {
	return m.renderFourPanelView()
}

func (m Model) renderFourPanelView() string {
	// Calculate dimensions for 4-panel grid
	// 60% left, 40% right for columns
	// 60% top, 40% bottom for rows
	leftWidth := int(float64(m.width) * 0.6)
	rightWidth := m.width - leftWidth

	topHeight := int(float64(m.height) * 0.6)
	bottomHeight := m.height - topHeight

	// Render all four panels
	topLeftPanel := m.renderContainerListPanel(leftWidth, topHeight)
	topRightPanel := m.renderStatsPanel(rightWidth, topHeight)
	bottomLeftPanel := m.renderGraphPanel(leftWidth, bottomHeight)
	bottomRightPanel := m.renderLogPanel(rightWidth, bottomHeight)

	// Join top row
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeftPanel, topRightPanel)

	// Join bottom row
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeftPanel, bottomRightPanel)

	// Join rows vertically
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

func (m Model) renderContainerListPanel(width, height int) string {
	content := m.renderListPanelContent(width, height)
	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

func (m Model) renderStatsPanel(width, height int) string {
	content := m.renderStatsPanelContent(width, height)
	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

func (m Model) renderGraphPanel(width, height int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📈 Resource Usage Graph") + "\n\n")
	s.WriteString("Coming soon...\n")
	s.WriteString("CPU and Memory usage over time")

	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(s.String())
}

func (m Model) renderLogPanel(width, height int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📋 Log Preview") + "\n\n")

	if len(m.containers) == 0 {
		s.WriteString("No container selected")
	} else {
		container := m.containers[m.cursor]
		s.WriteString(fmt.Sprintf("Container: %s\n\n", container.Name))
		s.WriteString("Coming soon...\n")
		s.WriteString("Real-time log streaming")
	}

	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(s.String())
}

func (m Model) renderListPanelContent(width, height int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("🐳 Containers") + "\n\n")

	if m.err != nil {
		s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return s.String()
	}

	if m.loading && len(m.containers) == 0 {
		s.WriteString("Loading...\n")
		return s.String()
	}

	running := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		}
	}
	s.WriteString(fmt.Sprintf("%d total, %d running\n\n", len(m.containers), running))

	// Adjusted column widths for the panel
	colWidth := width - 10
	nameWidth := int(float64(colWidth) * 0.25)
	imageWidth := int(float64(colWidth) * 0.30)
	stateWidth := 10
	statusWidth := colWidth - nameWidth - imageWidth - stateWidth

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		imageWidth, "IMAGE",
		stateWidth, "STATE",
		statusWidth, "STATUS")
	s.WriteString(headerStyle.Render(header) + "\n")

	// Calculate how many containers we can show
	maxContainers := height - 10 // Reserve space for header, help, etc.

	for i, container := range m.containers {
		if i >= maxContainers {
			break
		}

		name := truncate(container.Name, nameWidth)
		image := truncate(container.Image, imageWidth)

		var stateStr string
		if container.State == "running" {
			stateStr = runningStyle.Render("running")
		} else {
			stateStr = stoppedStyle.Render(container.State)
		}

		status := truncate(container.DisplayStatus, statusWidth)

		line := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s",
			nameWidth, name,
			imageWidth, image,
			stateWidth+10, stateStr, // Account for ANSI codes
			statusWidth, status,
		)

		if i == m.cursor {
			s.WriteString(selectedStyle.Render("> " + line))
		} else {
			s.WriteString("  " + line)
		}
		s.WriteString("\n")
	}

	if m.message != "" {
		s.WriteString("\n" + m.message + "\n")
	}

	help := "\n[↑/k] up  [↓/j] down  [s] start  [x] stop  [r] restart  [R] refresh  [q] quit"
	s.WriteString(helpStyle.Render(help))

	return s.String()
}

func (m Model) renderStatsPanelContent(width, height int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📊 Stats") + "\n\n")

	if len(m.containers) == 0 {
		s.WriteString("No containers available")
		return s.String()
	}

	container := m.containers[m.cursor]

	if container.State != "running" {
		s.WriteString(fmt.Sprintf("Container: %s\n\n", container.Name))
		s.WriteString("Container must be running\nto view stats")
		return s.String()
	}

	s.WriteString(RenderStats(&container, m.currentStats))

	return s.String()
}

func RenderStats(container *model.Container, stats *model.Stats) string {
	if stats == nil {
		return helpStyle.Render("No stats available")
	}

	// Convert memory to MB
	memUsageMB := float64(stats.MemoryUsage) / 1024 / 1024
	memLimitMB := float64(stats.MemoryLimit) / 1024 / 1024
	memCacheMB := float64(stats.MemoryCache) / 1024 / 1024

	// Helpers
	renderBar := func(percent float64, length int) string {
		filled := int(percent / 100 * float64(length))
		if filled > length {
			filled = length
		}
		return strings.Repeat("█", filled) + strings.Repeat("─", length-filled)
	}

	colorize := func(percent float64, text string) string {
		var color string
		switch {
		case percent > 80:
			color = "#F38BA8" // red/pink
		case percent > 50:
			color = "#FAB387" // orange
		default:
			color = "#A6E3A1" // green
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
	}

	formatBytes := func(b uint64) string {
		switch {
		case b > 1_000_000_000:
			return fmt.Sprintf("%.2f GB", float64(b)/1_000_000_000)
		case b > 1_000_000:
			return fmt.Sprintf("%.2f MB", float64(b)/1_000_000)
		case b > 1_000:
			return fmt.Sprintf("%.2f KB", float64(b)/1_000)
		default:
			return fmt.Sprintf("%d B", b)
		}
	}

	barLength := 30 // wider bar for vertical layout

	// CPU box
	cpuBar := renderBar(stats.CPUPercent, barLength)
	cpuStr := fmt.Sprintf("%6.2f%% |%s|", stats.CPUPercent, cpuBar)
	cpuBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#89B4FA")).
		Padding(0, 1).
		Render("CPU\n" + colorize(stats.CPUPercent, cpuStr))

	// Memory box
	memBar := renderBar(stats.MemoryPercent, barLength)
	memStr := fmt.Sprintf("%6.2f MB / %6.2f MB (%.2f%%) |%s| Cache: %5.2f MB",
		memUsageMB, memLimitMB, stats.MemoryPercent, memBar, memCacheMB)
	memBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#A6E3A1")).
		Padding(0, 1).
		Render("MEM\n" + colorize(stats.MemoryPercent, memStr))

	// PIDs
	pidsStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9E2AF")).
		Render(fmt.Sprintf("PIDs: %d", stats.PIDs))

	// Network
	netStr := fmt.Sprintf("Rx: %7s | Tx: %7s | RxPkts: %6d | TxPkts: %6d",
		formatBytes(stats.NetworkRx), formatBytes(stats.NetworkTx),
		stats.NetworkRxPackets, stats.NetworkTxPackets)
	netStr = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89B4FA")).
		Render("Network: " + netStr)

	// Disk I/O
	blockStr := fmt.Sprintf("Read: %7s | Write: %7s",
		formatBytes(stats.BlockRead), formatBytes(stats.BlockWrite))
	blockStr = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CBA6F7")).
		Render("Disk I/O: " + blockStr)

	// Container title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F5C2E7")).
		Render("Container: " + container.Name)

	// Build final layout vertically
	result := lipgloss.JoinVertical(lipgloss.Left,
		title,
		cpuBox,
		memBox,
		pidsStr,
		netStr,
		blockStr,
	)

	return result
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	} else {
		return s[:max-3] + "..."
	}
}
