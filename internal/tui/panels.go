package tui

import (
	"fmt"
	"strings"
)

// renderContainerListPanel renders the container list panel
func (m Model) renderContainerListPanel(width, height int) string {
	content := m.renderListPanelContent(width, height)
	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

// renderListPanelContent renders the content of the container list panel
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

// renderGraphPanel renders the graph panel with historical data
func (m Model) renderGraphPanel(width, height int) string {
	var content string

	// Query data from storage if available
	if m.storage != nil && m.currentContainerID != "" {
		dataPoints, err := m.storage.Query(m.currentContainerID, m.timeRange)
		if err == nil && len(dataPoints) > 0 {
			// Convert to separate CPU and Memory slices
			cpuData := make([]float64, len(dataPoints))
			memData := make([]float64, len(dataPoints))
			for i, dp := range dataPoints {
				cpuData[i] = dp.CPUPercent
				memData[i] = dp.MemoryPercent
			}
			content = renderDualGraphWithRange(cpuData, memData, width-4, height-4, m.timeRange)
		} else {
			// Fallback to in-memory data
			content = renderDualGraphWithRange(m.cpuHistory, m.memoryHistory, width-4, height-4, m.timeRange)
		}
	} else {
		// Use in-memory data
		content = renderDualGraphWithRange(m.cpuHistory, m.memoryHistory, width-4, height-4, m.timeRange)
	}

	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

// renderLogPanel renders the log panel
func (m Model) renderLogPanel(width, height int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📋 Log Preview") + "\n\n")

	if len(m.containers) == 0 {
		s.WriteString("No container selected")
	} else {
		container := m.containers[m.cursor]
		s.WriteString(fmt.Sprintf("Container: %s", container.Name))

		// Show auto-scroll indicator
		autoScrollIndicator := ""
		if m.logsAutoScroll {
			autoScrollIndicator = " [Auto-scroll: ON]"
		}
		s.WriteString(autoScrollIndicator + "\n\n")

		if len(m.logs) == 0 {
			s.WriteString("No logs yet...")
		} else {
			// Calculate visible lines: reserve space for title, container name, and help text
			visibleLines := height - 8
			if visibleLines < 1 {
				visibleLines = 1
			}

			// Calculate the window of logs to display
			totalLogs := len(m.logs)
			start := m.logsScroll
			end := start + visibleLines

			// Clamp the range
			if start < 0 {
				start = 0
			}
			if end > totalLogs {
				end = totalLogs
			}
			if start >= totalLogs {
				start = totalLogs - visibleLines
				if start < 0 {
					start = 0
				}
			}

			// Render only the visible window of logs
			maxLineWidth := width - 8
			for i := start; i < end && i < totalLogs; i++ {
				log := m.logs[i]
				styledLine := styleLogEntry(log, maxLineWidth)
				s.WriteString(styledLine + "\n")
			}

			// Show scroll indicator if there are more logs
			if totalLogs > visibleLines {
				s.WriteString(fmt.Sprintf("\n[%d/%d] PgUp/PgDown:scroll | a:toggle auto | c:clear",
					start+1, totalLogs))
			}
		}
	}

	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(s.String())
}

// renderStatsPanel renders the stats panel
func (m Model) renderStatsPanel(width, height int) string {
	content := m.renderStatsPanelContent(width, height)
	return panelStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

// renderStatsPanelContent renders the content of the stats panel
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

	// Use current stats with stored processes
	statsWithProcesses := m.currentStats
	if statsWithProcesses != nil && len(m.currentProcesses) > 0 {
		// Create a copy with processes
		statsCopy := *statsWithProcesses
		statsCopy.Processes = m.currentProcesses
		s.WriteString(RenderStats(&container, &statsCopy))
	} else {
		s.WriteString(RenderStats(&container, m.currentStats))
	}

	return s.String()
}
