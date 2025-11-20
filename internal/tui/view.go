package tui

import "github.com/charmbracelet/lipgloss"

// View renders the TUI interface
func (m Model) View() string {
	return m.renderFourPanelView()
}

// renderFourPanelView renders the four-panel grid layout
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
