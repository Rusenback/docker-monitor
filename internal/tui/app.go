package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Model sisältää TUI sovelluksen tilan
type Model struct {
	// TODO: Lisää kentät
}


// NewModel luo uuden TUI modelin
func NewModel() Model {
	return Model{}
}

// Init alustetaan sovellus
func (m Model) Init() tea.Cmd {
	return nil
}

// Update käsittelee viestit
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renderöi näkymän
func (m Model) View() string {
	return "Docker Monitor MVP\n\nTulossa pian...\n"
}
