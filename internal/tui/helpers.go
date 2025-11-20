package tui

// truncate shortens a string to a maximum length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// calculateVisibleLogLines calculates how many log lines can fit in the panel
func (m Model) calculateVisibleLogLines() int {
	// Bottom panel is 40% of height
	bottomHeight := int(float64(m.height) * 0.4)
	// Reserve space for title, container name, help text, and padding
	visibleLines := bottomHeight - 8
	if visibleLines < 1 {
		visibleLines = 1
	}
	return visibleLines
}

// calculateMaxScroll calculates the maximum scroll position
func (m Model) calculateMaxScroll() int {
	visibleLines := m.calculateVisibleLogLines()
	maxScroll := len(m.logs) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
