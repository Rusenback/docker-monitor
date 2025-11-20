package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B4BEFE"))

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

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6ADC8")).Padding(1, 0)

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585B70")).
		Padding(1, 2)
)
