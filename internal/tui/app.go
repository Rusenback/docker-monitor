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
)

// Model sisältää TUI sovelluksen tilan
type Model struct {
	client     *docker.Client
	containers []model.Container
	cursor     int
	err        error
	loading    bool
	message    string
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

// NewModel luo uuden TUI modelin
func NewModel(client *docker.Client) Model {
	return Model{
		client:  client,
		loading: true,
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
func fetchContainers(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		containers, err := client.ListContainers()
		return containersMsg{containers: containers, err: err}
	}
}

// Update käsittelee viestit
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.containers)-1 {
				m.cursor++
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
func startContainer(client *docker.Client, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.StartContainer(id)
		return actionMsg{
			message: fmt.Sprintf("Started: %s", name),
			err:     err,
		}
	}
}

// stopContainer pysäyttää containerin
func stopContainer(client *docker.Client, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.StopContainer(id)
		return actionMsg{
			message: fmt.Sprintf("Stopped: %s", name),
			err:     err,
		}
	}
}

// restartContainer uudelleenkäynnistää containerin
func restartContainer(client *docker.Client, id, name string) tea.Cmd {
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
	var s strings.Builder

	// Otsikko
	s.WriteString(titleStyle.Render("🐳 Docker Monitor") + "\n\n")

	// Virheviesti
	if m.err != nil {
		s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return s.String()
	}

	// Ladataan...
	if m.loading && len(m.containers) == 0 {
		s.WriteString("Loading containers...\n")
		return s.String()
	}

	// Containerien määrä
	running := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		}
	}
	s.WriteString(fmt.Sprintf("Containers: %d total, %d running\n\n", len(m.containers), running))

	// Taulukon header
	header := fmt.Sprintf("%-20s %-25s %-12s %-30s",
		"NAME", "IMAGE", "STATE", "STATUS")
	s.WriteString(headerStyle.Render(header) + "\n")

	// Containerit
	if len(m.containers) == 0 {
		s.WriteString("\nNo containers found.\n")
		s.WriteString("Start some containers: docker run -d nginx\n")
	} else {
		for i, container := range m.containers {
			// Truncate pitkät stringit
			name := truncate(container.Name, 20)
			image := truncate(container.Image, 25)
			status := truncate(container.Status, 30)

			// State väri
			var stateStr string
			if container.State == "running" {
				stateStr = runningStyle.Render("running")
			} else {
				stateStr = stoppedStyle.Render(container.State)
			}

			line := fmt.Sprintf("%-20s %-25s %-12s %-30s",
				name, image, stateStr, status)

			// Valittu rivi
			if i == m.cursor {
				s.WriteString(selectedStyle.Render("> " + line))
			} else {
				s.WriteString("  " + line)
			}
			s.WriteString("\n")
		}
	}

	// Viesti
	if m.message != "" {
		s.WriteString("\n" + m.message + "\n")
	}

	// Ohje
	help := "\n" + strings.Repeat("─", 80) + "\n"
	help += "[↑/k] up  [↓/j] down  [s] start  [x] stop  [r] restart  [R] refresh  [q] quit"
	s.WriteString(helpStyle.Render(help))

	return s.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
