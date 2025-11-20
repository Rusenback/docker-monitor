package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/storage"
)

var (
	graphTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B4BEFE"))
	graphAxisStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	cpuGraphStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
	memGraphStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
)

// renderGraph creates an ASCII line graph
func renderGraph(data []float64, height int, label string, color lipgloss.Style) string {
	if len(data) == 0 {
		return color.Render("No data yet...")
	}

	var s strings.Builder

	// Find min and max for scaling
	min, max := math.MaxFloat64, 0.0
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// If all values are the same, adjust range slightly
	if max == min {
		min = math.Max(0, max-10)
		max = max + 10
	}

	// Ensure we have a valid range
	if max == 0 {
		max = 100
	}

	dataRange := max - min
	if dataRange == 0 {
		dataRange = 1
	}

	// Chart characters
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	// Render the graph line
	var graphLine strings.Builder
	for _, value := range data {
		// Normalize value to 0-1 range
		normalized := (value - min) / dataRange
		// Map to character index
		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}
		if charIndex < 0 {
			charIndex = 0
		}
		graphLine.WriteString(chars[charIndex])
	}

	// Header with current value and range
	current := data[len(data)-1]
	header := fmt.Sprintf("%s: %.1f%% (min: %.1f%%, max: %.1f%%)", label, current, min, max)
	s.WriteString(graphTitleStyle.Render(header) + "\n\n")

	// The graph
	s.WriteString(color.Render(graphLine.String()) + "\n")

	// Timeline indicators
	dataPoints := len(data)
	timeline := fmt.Sprintf("◄─ %ds ago", dataPoints*2) // Assuming 2s intervals
	s.WriteString(graphAxisStyle.Render(timeline))

	return s.String()
}

// renderSparkline creates a compact sparkline
func renderSparkline(data []float64, width int) string {
	if len(data) == 0 {
		return strings.Repeat("▁", width)
	}

	// Take last 'width' points
	start := 0
	if len(data) > width {
		start = len(data) - width
	}
	displayData := data[start:]

	// Find min and max
	min, max := math.MaxFloat64, 0.0
	for _, v := range displayData {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if max == min {
		min = math.Max(0, max-10)
		max = max + 10
	}

	dataRange := max - min
	if dataRange == 0 {
		dataRange = 1
	}

	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	var result strings.Builder

	for _, value := range displayData {
		normalized := (value - min) / dataRange
		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}
		if charIndex < 0 {
			charIndex = 0
		}
		result.WriteString(chars[charIndex])
	}

	// Pad if needed
	for result.Len() < width {
		result.WriteString("▁")
	}

	return result.String()
}

// renderDualGraphWithRange renders CPU and Memory on a single combined graph with time range indicator
func renderDualGraphWithRange(
	cpuData, memData []float64,
	width, height int,
	timeRange storage.TimeRange,
) string {
	var s strings.Builder

	// Title with time range
	title := fmt.Sprintf("📈 Resource Usage - %s", timeRange.String())
	s.WriteString(graphTitleStyle.Render(title) + "\n")

	// Time range selector hint
	hint := "[1]30m [2]1h [3]6h [4]1d [5]1w"
	s.WriteString(graphAxisStyle.Render(hint) + "\n\n")

	if len(cpuData) == 0 && len(memData) == 0 {
		s.WriteString("Waiting for data...\n")
		s.WriteString("Stats will appear once container starts generating metrics.")
		return s.String()
	}

	// Calculate available height for the combined graph
	graphHeight := height - 14
	if graphHeight < 5 {
		graphHeight = 5
	}

	// Render combined multi-line graph
	combinedGraph := renderCombinedGraph(cpuData, memData, width-8, graphHeight)
	s.WriteString(combinedGraph)

	return s.String()
}

// renderCombinedGraph creates a multi-line ASCII graph with both CPU and Memory
func renderCombinedGraph(cpuData, memData []float64, width, height int) string {
	var s strings.Builder

	// Ensure we have data
	if len(cpuData) == 0 || len(memData) == 0 {
		return "Waiting for data..."
	}

	// Get current values
	cpuCurrent := cpuData[len(cpuData)-1]
	memCurrent := memData[len(memData)-1]

	// Legend with overlap color
	cpuLegend := cpuGraphStyle.Render(
		"█",
	) + " CPU: " + cpuGraphStyle.Render(
		fmt.Sprintf("%.1f%%", cpuCurrent),
	)
	memLegend := memGraphStyle.Render(
		"█",
	) + " Memory: " + memGraphStyle.Render(
		fmt.Sprintf("%.1f%%", memCurrent),
	)
	overlapLegend := lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7")).Render("█") + " Both"
	s.WriteString(cpuLegend + "  " + memLegend + "  " + overlapLegend + "\n\n")

	// Find global min/max for consistent scaling
	minVal, maxVal := 0.0, 100.0

	// Limit data points to available width (leave room for Y-axis labels)
	maxWidth := width - 10
	if maxWidth < 20 {
		maxWidth = 20
	}
	dataPointsToShow := len(cpuData)
	if dataPointsToShow > maxWidth {
		dataPointsToShow = maxWidth
	}

	// Get the data slice we'll display
	startIdx := len(cpuData) - dataPointsToShow
	if startIdx < 0 {
		startIdx = 0
	}
	displayCPU := cpuData[startIdx:]
	displayMem := memData[startIdx:]

	// Render the vertical graph (top to bottom)
	for row := height; row >= 0; row-- {
		var line strings.Builder

		// Determine if this is a grid line row (at 25%, 50%, 75%, 100%)
		isGridLine := false
		if row == height || row == height*3/4 || row == height/2 || row == height/4 || row == 0 {
			isGridLine = true
		}

		// Y-axis label (every few rows)
		if row == height {
			line.WriteString(graphAxisStyle.Render(fmt.Sprintf("%3.0f%% ", maxVal)))
		} else if row == height*3/4 {
			line.WriteString(graphAxisStyle.Render(" 75% "))
		} else if row == height/2 {
			line.WriteString(graphAxisStyle.Render(" 50% "))
		} else if row == height/4 {
			line.WriteString(graphAxisStyle.Render(" 25% "))
		} else if row == 0 {
			line.WriteString(graphAxisStyle.Render(fmt.Sprintf("%3.0f%% ", minVal)))
		} else {
			line.WriteString("     ")
		}

		// Vertical line
		line.WriteString(graphAxisStyle.Render("│"))

		// Calculate threshold for this row
		threshold := minVal + (float64(row)/float64(height))*(maxVal-minVal)

		// Draw data points
		for i := 0; i < len(displayCPU); i++ {
			cpuVal := displayCPU[i]
			memVal := displayMem[i]

			// Determine what to draw at this position
			cpuAbove := cpuVal >= threshold
			memAbove := memVal >= threshold

			// If it's a grid line and no data, show grid character
			if isGridLine && !cpuAbove && !memAbove {
				line.WriteString(graphAxisStyle.Render("·"))
			} else if cpuAbove && memAbove {
				// Both are above threshold - show overlay character
				line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7")).Render("█"))
			} else if cpuAbove {
				// Only CPU above
				line.WriteString(cpuGraphStyle.Render("█"))
			} else if memAbove {
				// Only Memory above
				line.WriteString(memGraphStyle.Render("█"))
			} else {
				// Neither above
				line.WriteString(" ")
			}
		}

		s.WriteString(line.String() + "\n")
	}

	// X-axis
	axisLength := len(displayCPU)
	if axisLength < 1 {
		axisLength = 1
	}
	s.WriteString(
		"     " + graphAxisStyle.Render(
			"└",
		) + graphAxisStyle.Render(
			strings.Repeat("─", axisLength),
		) + "\n",
	)

	// Time labels - show multiple time markers along the axis
	s.WriteString(renderTimeLabels(axisLength, len(displayCPU)) + "\n")

	// Data info
	s.WriteString("\n")
	infoText := fmt.Sprintf("Tracking %d data points | Updates every ~2s", len(cpuData))
	s.WriteString(graphAxisStyle.Render(infoText))

	return s.String()
}

// renderTimeLabels creates time markers along the X-axis
func renderTimeLabels(axisLength, dataPoints int) string {
	if axisLength < 20 {
		// Too narrow for labels
		return graphAxisStyle.Render(fmt.Sprintf("     %ds ago → Now", dataPoints*2))
	}

	// Calculate time span
	totalSeconds := dataPoints * 2

	// Determine number of markers based on width
	numMarkers := 5
	if axisLength < 50 {
		numMarkers = 3
	}

	// Pre-calculate all labels and their positions
	type marker struct {
		position int
		label    string
	}
	markers := make([]marker, numMarkers)

	for i := 0; i < numMarkers; i++ {
		position := (i * axisLength) / (numMarkers - 1)
		if i == numMarkers-1 {
			position = axisLength - 1
		}

		// Calculate time for this position (reverse: leftmost is oldest)
		dataPointIndex := (position * dataPoints) / axisLength
		secondsAgo := totalSeconds - (dataPointIndex * 2)

		// Format time label
		var label string
		if secondsAgo < 60 {
			label = "Now"
		} else if secondsAgo < 3600 {
			label = fmt.Sprintf("%dm ago", secondsAgo/60)
		} else if secondsAgo < 86400 {
			label = fmt.Sprintf("%dh ago", secondsAgo/3600)
		} else {
			days := secondsAgo / 86400
			label = fmt.Sprintf("%dd ago", days)
		}

		markers[i] = marker{position: position, label: label}
	}

	// Build the output string with proper spacing
	var s strings.Builder
	s.WriteString("     ") // Y-axis label space

	currentCol := 0
	for _, m := range markers {
		// Calculate where this label should be centered
		labelStart := m.position - len(m.label)/2
		if labelStart < currentCol {
			labelStart = currentCol
		}

		// Add spacing to reach the label position
		spacesNeeded := labelStart - currentCol
		if spacesNeeded > 0 {
			s.WriteString(strings.Repeat(" ", spacesNeeded))
		}

		// Write the label
		s.WriteString(m.label)
		currentCol = labelStart + len(m.label)
	}

	return graphAxisStyle.Render(s.String())
}
