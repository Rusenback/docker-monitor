package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/model"
)

// RenderStats renders the statistics for a container
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

	// Top Processes
	processesSection := renderProcesses(stats.Processes)

	// Build final layout vertically
	result := lipgloss.JoinVertical(lipgloss.Left,
		title,
		cpuBox,
		memBox,
		pidsStr,
		netStr,
		blockStr,
		processesSection,
	)

	return result
}

// renderProcesses renders the top processes table
func renderProcesses(processes []model.Process) string {
	if len(processes) == 0 {
		return ""
	}

	var s strings.Builder
	s.WriteString("\n")

	// Title
	procTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F9E2AF")).
		Render("Top Processes")
	s.WriteString(procTitle + "\n")

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Bold(true)

	header := fmt.Sprintf("%-8s %-10s %6s %6s %s",
		"PID", "USER", "%CPU", "%MEM", "COMMAND")
	s.WriteString(headerStyle.Render(header) + "\n")

	// Process rows
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	for _, proc := range processes {
		// Truncate command if too long
		cmd := proc.Command
		if len(cmd) > 40 {
			cmd = cmd[:37] + "..."
		}

		row := fmt.Sprintf("%-8s %-10s %6s %6s %s",
			truncateStr(proc.PID, 8),
			truncateStr(proc.User, 10),
			truncateStr(proc.CPU, 6),
			truncateStr(proc.Memory, 6),
			cmd)
		s.WriteString(rowStyle.Render(row) + "\n")
	}

	return s.String()
}

// truncateStr truncates a string to a maximum length
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
