package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/docker"
	"github.com/rusenback/docker-monitor/internal/model"
	"github.com/rusenback/docker-monitor/internal/tui/views"
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
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#585B70")).
			Padding(0, 1)
)

type Model struct {
	client       docker.DockerClient
	containers   []model.Container
	cursor       int
	err          error
	loading      bool
	message      string
	currentStats *model.Stats
	statsCancel  func()
	width        int
	height       int
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

func (m Model) View() string { return m.renderSplitView() }

func (m Model) renderSplitView() string {
	listWidth := int(float64(m.width) * 0.6)
	statsWidth := m.width - listWidth
	leftPanel := panelStyle.Width(listWidth).Render(m.renderListPanelContent(listWidth))
	rightPanel := panelStyle.Width(statsWidth).Render(m.renderStatsPanelContent(statsWidth))
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderListPanelContent(width int) string {
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
	header := fmt.Sprintf("%-20s %-25s %-12s %-30s", "NAME", "IMAGE", "STATE", "STATUS")
	s.WriteString(headerStyle.Render(header) + "\n")
	for i, container := range m.containers {
		name := truncate(container.Name, 20)
		image := truncate(container.Image, 25)
		var stateStr string
		if container.State == "running" {
			stateStr = runningStyle.Render("running")
		} else {
			stateStr = stoppedStyle.Render(container.State)
		}
		line := fmt.Sprintf(
			"%-20s %-25s %-12s %-30s",
			name,
			image,
			stateStr,
			container.DisplayStatus,
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
	help := "\n" + strings.Repeat(
		"─",
		width-4,
	) + "\n" + "[↑/k] up  [↓/j] down  [s] start  [x] stop  [r] restart  [R] refresh  [q] quit"
	s.WriteString(helpStyle.Render(help))
	return s.String()
}

func (m Model) renderStatsPanelContent(width int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📊 Stats") + "\n\n")
	if len(m.containers) == 0 {
		return s.String()
	}
	container := m.containers[m.cursor]
	if container.State != "running" {
		s.WriteString("Container must be running\nto view stats")
		return s.String()
	}
	s.WriteString(views.RenderStats(&container, m.currentStats))
	return s.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	} else {
		return s[:max-3] + "..."
	}
}
