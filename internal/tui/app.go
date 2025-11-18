// internal/tui/app.go
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

// Värit ja tyylit
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#7D56F4"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Padding(1, 0)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
)

// Model sisältää TUI sovelluksen tilan
type Model struct {
	client       docker.DockerClient
	containers   []model.Container
	cursor       int
	err          error
	loading      bool
	message      string
	currentStats *model.Stats
	statsCancel  func()
	showStats    bool
	width        int
	height       int
}

// tickMsg laukaistaan säännöllisesti päivittämään containereita
type tickMsg time.Time

// containersMsg sisältää haetut containerit
type containersMsg struct {
	containers []model.Container
	err        error
}

// actionMsg sisältää toiminnon tuloksen
type actionMsg struct {
	message string
	err     error
}

// statsMsg sisältää containerin stats
type statsMsg struct {
	stats *model.Stats
	err   error
}

// NewModel luo uuden TUI modelin
func NewModel(client docker.DockerClient) Model {
	return Model{
		client:    client,
		loading:   true,
		showStats: false,
	}
}

// Init alustetaan sovellus - hae containerit heti
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchContainers(m.client),
		tickCmd(),
	)
}

// tickCmd palauttaa komennon joka tickaa 2 sekunnin välein
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchContainers hakee containerit taustalla
func fetchContainers(client docker.DockerClient) tea.Cmd {
	return func() tea.Msg {
		containers, err := client.ListContainers()
		return containersMsg{containers: containers, err: err}
	}
}

// waitForStats odottaa stats dataa
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

// Update käsittelee viestit
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

		case "tab":
			// Toggle stats panel
			m.showStats = !m.showStats

			if m.showStats && len(m.containers) > 0 {
				container := m.containers[m.cursor]
				if container.State == "running" {
					// Käynnistä stats stream
					statsChan, errChan, cancel := m.client.StreamContainerStats(container.ID)
					m.statsCancel = cancel
					return m, waitForStats(statsChan, errChan)
				} else {
					m.message = "Container must be running to view stats"
					m.showStats = false
				}
			} else if !m.showStats && m.statsCancel != nil {
				// Sulje stats stream
				m.statsCancel()
				m.statsCancel = nil
				m.currentStats = nil
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Jos stats on auki, päivitä stats uudelle containerille
				if m.showStats {
					if m.statsCancel != nil {
						m.statsCancel()
					}
					container := m.containers[m.cursor]
					if container.State == "running" {
						statsChan, errChan, cancel := m.client.StreamContainerStats(container.ID)
						m.statsCancel = cancel
						return m, waitForStats(statsChan, errChan)
					} else {
						m.currentStats = nil
						m.message = "Container must be running to view stats"
					}
				}
			}

		case "down", "j":
			if m.cursor < len(m.containers)-1 {
				m.cursor++
				// Jos stats on auki, päivitä stats uudelle containerille
				if m.showStats {
					if m.statsCancel != nil {
						m.statsCancel()
					}
					container := m.containers[m.cursor]
					if container.State == "running" {
						statsChan, errChan, cancel := m.client.StreamContainerStats(container.ID)
						m.statsCancel = cancel
						return m, waitForStats(statsChan, errChan)
					} else {
						m.currentStats = nil
						m.message = "Container must be running to view stats"
					}
				}
			}

		case "s":
			// Start container
			if len(m.containers) > 0 {
				container := m.containers[m.cursor]
				return m, startContainer(m.client, container.ID, container.Name)
			}

		case "x":
			// Stop container
			if len(m.containers) > 0 {
				container := m.containers[m.cursor]
				return m, stopContainer(m.client, container.ID, container.Name)
			}

		case "r":
			// Restart container
			if len(m.containers) > 0 {
				container := m.containers[m.cursor]
				return m, restartContainer(m.client, container.ID, container.Name)
			}

		case "R":
			// Refresh manually
			m.loading = true
			m.message = "Refreshing..."
			return m, fetchContainers(m.client)
		}

	case tickMsg:
		// Auto-refresh joka 2 sekunti
		return m, tea.Batch(
			fetchContainers(m.client),
			tickCmd(),
		)

	case containersMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.containers = msg.containers
			// Varmista että cursor on validissa kohdassa
			if m.cursor >= len(m.containers) && len(m.containers) > 0 {
				m.cursor = len(m.containers) - 1
			}
		}

	case statsMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Stats error: %v", msg.err)
		} else {
			m.currentStats = msg.stats
			m.message = ""

			// Jatka streamausta jos stats on auki
			if m.showStats && len(m.containers) > 0 {
				container := m.containers[m.cursor]
				if container.State == "running" {
					statsChan, errChan, _ := m.client.StreamContainerStats(container.ID)
					return m, waitForStats(statsChan, errChan)
				}
			}
		}

	case actionMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.message = msg.message
		}
		// Päivitä lista heti toiminnon jälkeen
		return m, fetchContainers(m.client)
	}

	return m, nil
}

// startContainer käynnistää containerin
func startContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.StartContainer(id)
		return actionMsg{
			message: fmt.Sprintf("Started: %s", name),
			err:     err,
		}
	}
}

// stopContainer pysäyttää containerin
func stopContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.StopContainer(id)
		return actionMsg{
			message: fmt.Sprintf("Stopped: %s", name),
			err:     err,
		}
	}
}

// restartContainer uudelleenkäynnistää containerin
func restartContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.RestartContainer(id)
		return actionMsg{
			message: fmt.Sprintf("Restarted: %s", name),
			err:     err,
		}
	}
}

// View renderöi näkymän
func (m Model) View() string {
	if m.showStats {
		return m.renderSplitView()
	}
	return m.renderListView()
}

func (m Model) renderSplitView() string {
	// Jaa ruutu kahtia: 60% listalle, 40% statsille
	listWidth := int(float64(m.width) * 0.6)
	statsWidth := m.width - listWidth - 4 // -4 for borders and padding

	leftPanel := m.renderListPanel(listWidth)
	rightPanel := m.renderStatsPanel(statsWidth)

	// Yhdistä paneelit vierekkäin
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderListPanel(width int) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🐳 Containers") + "\n\n")

	if m.err != nil {
		s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return panelStyle.Width(width).Render(s.String())
	}

	if m.loading && len(m.containers) == 0 {
		s.WriteString("Loading...\n")
		return panelStyle.Width(width).Render(s.String())
	}

	// Container count
	running := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		}
	}
	s.WriteString(fmt.Sprintf("%d total, %d running\n\n", len(m.containers), running))

	// Header
	header := fmt.Sprintf("%-18s %-20s %-10s", "NAME", "IMAGE", "STATE")
	s.WriteString(headerStyle.Render(header) + "\n")

	// Containers
	if len(m.containers) == 0 {
		s.WriteString("\nNo containers\n")
	} else {
		for i, container := range m.containers {
			name := truncate(container.Name, 18)
			image := truncate(container.Image, 20)

			var stateStr string
			if container.State == "running" {
				stateStr = runningStyle.Render("running")
			} else {
				stateStr = stoppedStyle.Render(container.State)
			}

			line := fmt.Sprintf("%-18s %-20s %-10s", name, image, stateStr)

			if i == m.cursor {
				s.WriteString(selectedStyle.Render("> " + line))
			} else {
				s.WriteString("  " + line)
			}
			s.WriteString("\n")
		}
	}

	// Message
	if m.message != "" {
		s.WriteString("\n" + m.message + "\n")
	}

	// Help
	help := "\n" + strings.Repeat("─", width-4) + "\n"
	help += "[↑/↓] navigate [Tab] stats\n"
	help += "[s]tart [x]stop [r]estart [q]uit"
	s.WriteString(helpStyle.Render(help))

	return panelStyle.Width(width).Render(s.String())
}

func (m Model) renderStatsPanel(width int) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("📊 Stats") + "\n\n")

	if len(m.containers) == 0 {
		return panelStyle.Width(width).Render(s.String())
	}

	container := m.containers[m.cursor]

	if container.State != "running" {
		s.WriteString("Container must be running\nto view stats")
		return panelStyle.Width(width).Render(s.String())
	}

	statsView := views.RenderStats(&container, m.currentStats)
	s.WriteString(statsView)

	// Help
	help := "\n\n[Tab] close stats"
	s.WriteString(helpStyle.Render(help))

	return panelStyle.Width(width).Render(s.String())
}

func (m Model) renderListView() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🐳 Docker Monitor") + "\n\n")

	if m.err != nil {
		s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return s.String()
	}

	if m.loading && len(m.containers) == 0 {
		s.WriteString("Loading containers...\n")
		return s.String()
	}

	// Container count
	running := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		}
	}
	s.WriteString(fmt.Sprintf("Containers: %d total, %d running\n\n", len(m.containers), running))

	// Header
	header := fmt.Sprintf("%-20s %-25s %-12s %-30s",
		"NAME", "IMAGE", "STATE", "STATUS")
	s.WriteString(headerStyle.Render(header) + "\n")

	// Containers
	if len(m.containers) == 0 {
		s.WriteString("\nNo containers found.\n")
		s.WriteString("Start some containers: docker run -d nginx\n")
	} else {
		for i, container := range m.containers {
			name := truncate(container.Name, 20)
			image := truncate(container.Image, 25)
			status := truncate(container.Status, 30)

			var stateStr string
			if container.State == "running" {
				stateStr = runningStyle.Render("running")
			} else {
				stateStr = stoppedStyle.Render(container.State)
			}

			line := fmt.Sprintf("%-20s %-25s %-12s %-30s",
				name, image, stateStr, status)

			if i == m.cursor {
				s.WriteString(selectedStyle.Render("> " + line))
			} else {
				s.WriteString("  " + line)
			}
			s.WriteString("\n")
		}
	}

	// Message
	if m.message != "" {
		s.WriteString("\n" + m.message + "\n")
	}

	// Help
	help := "\n" + strings.Repeat("─", 90) + "\n"
	help += "[↑/k] up  [↓/j] down  [Tab] show stats  [s] start  [x] stop  [r] restart  [R] refresh  [q] quit"
	s.WriteString(helpStyle.Render(help))

	return s.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
