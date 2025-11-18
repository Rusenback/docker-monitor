// internal/tui/views/stats.go
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/model"
)

var (
	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(60)

	statsTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00"))

	statsLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	statsValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4"))
)

// RenderStats renderöi stats näkymän
func RenderStats(container *model.Container, stats *model.Stats) string {
	if stats == nil {
		return statsBoxStyle.Render("Loading stats...")
	}

	var s strings.Builder

	// Otsikko
	s.WriteString(statsTitleStyle.Render(fmt.Sprintf("📊 Container: %s", container.Name)))
	s.WriteString("\n\n")

	// CPU
	cpuBar := renderProgressBar(stats.CPUPercent, 100, 40)
	s.WriteString(statsLabelStyle.Render("CPU:     "))
	s.WriteString(cpuBar)
	s.WriteString(fmt.Sprintf(" %.2f%%\n", stats.CPUPercent))

	// Memory
	memBar := renderProgressBar(stats.MemoryPercent, 100, 40)
	memMB := float64(stats.MemoryUsage) / 1024 / 1024
	limitMB := float64(stats.MemoryLimit) / 1024 / 1024
	s.WriteString(statsLabelStyle.Render("Memory:  "))
	s.WriteString(memBar)
	s.WriteString(fmt.Sprintf(" %.1f/%.1f MB (%.1f%%)\n", memMB, limitMB, stats.MemoryPercent))

	s.WriteString("\n")

	// Network
	rxMB := float64(stats.NetworkRx) / 1024 / 1024
	txMB := float64(stats.NetworkTx) / 1024 / 1024
	s.WriteString(statsLabelStyle.Render("Network RX: "))
	s.WriteString(statsValueStyle.Render(fmt.Sprintf("%.2f MB\n", rxMB)))
	s.WriteString(statsLabelStyle.Render("Network TX: "))
	s.WriteString(statsValueStyle.Render(fmt.Sprintf("%.2f MB\n", txMB)))

	return statsBoxStyle.Render(s.String())
}

// renderProgressBar luo ASCII progress barin
func renderProgressBar(value, max float64, width int) string {
	if max == 0 {
		max = 1
	}

	percent := value / max
	if percent > 1 {
		percent = 1
	}
	if percent < 0 {
		percent = 0
	}

	filled := int(percent * float64(width))
	empty := width - filled

	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
	return progressBarStyle.Render(bar)
}

// FormatBytes formatoi tavut luettavaan muotoon
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
