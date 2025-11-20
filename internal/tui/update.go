package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusenback/docker-monitor/internal/model"
	"github.com/rusenback/docker-monitor/internal/storage"
)

// Update handles messages and updates the model state
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
			if m.logsCancel != nil {
				m.logsCancel()
			}
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.updateStatsAndLogsForCursor()
			}

		case "down", "j":
			if m.cursor < len(m.containers)-1 {
				m.cursor++
				return m, m.updateStatsAndLogsForCursor()
			}

		case "pgup":
			// Scroll logs up by half page for better readability
			if m.logsScroll > 0 {
				visibleLines := m.calculateVisibleLogLines()
				scrollAmount := visibleLines / 2
				if scrollAmount < 1 {
					scrollAmount = 1
				}
				m.logsScroll -= scrollAmount
				if m.logsScroll < 0 {
					m.logsScroll = 0
				}
				m.logsAutoScroll = false
			}

		case "pgdown":
			// Scroll logs down by half page for better readability
			visibleLines := m.calculateVisibleLogLines()
			maxScroll := m.calculateMaxScroll()
			scrollAmount := visibleLines / 2
			if scrollAmount < 1 {
				scrollAmount = 1
			}
			m.logsScroll += scrollAmount
			if m.logsScroll >= maxScroll {
				m.logsScroll = maxScroll
				m.logsAutoScroll = true
			}

		case "home":
			m.logsScroll = 0
			m.logsAutoScroll = false

		case "end":
			m.logsScroll = m.calculateMaxScroll()
			m.logsAutoScroll = true

		case "a":
			// Toggle auto-scroll
			m.logsAutoScroll = !m.logsAutoScroll
			if m.logsAutoScroll {
				m.logsScroll = m.calculateMaxScroll()
			}

		case "c":
			// Clear logs
			m.logs = []model.LogEntry{}
			m.logsScroll = 0

		case "s":
			if len(m.containers) > 0 {
				return m, startContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}

		case "x":
			if len(m.containers) > 0 {
				return m, stopContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}

		case "r":
			if len(m.containers) > 0 {
				return m, restartContainer(m.client, m.containers[m.cursor].ID, m.containers[m.cursor].Name)
			}

		case "R":
			m.loading = true
			m.message = "Refreshing..."
			return m, fetchContainers(m.client)

		case "1":
			m.timeRange = storage.Range30Min
		case "2":
			m.timeRange = storage.Range1Hour
		case "3":
			m.timeRange = storage.Range6Hour
		case "4":
			m.timeRange = storage.Range1Day
		case "5":
			m.timeRange = storage.Range1Week

		case "tab":
			// Cycle through panels: ContainerList -> Stats -> Graph -> Logs -> ContainerList
			m.focusedPanel = (m.focusedPanel + 1) % 4

		case "shift+tab":
			// Cycle backwards through panels
			m.focusedPanel = (m.focusedPanel + 3) % 4 // +3 is same as -1 in mod 4
		}

	case tickMsg:
		return m, tea.Batch(fetchContainers(m.client), tickCmd())

	case containersMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Check if container list actually changed
		containersChanged := containersListChanged(m.containers, msg.containers)

		m.containers = msg.containers
		if m.cursor >= len(m.containers) && len(m.containers) > 0 {
			m.cursor = len(m.containers) - 1
		}

		// Only update stats/logs if containers changed or cursor container changed
		if containersChanged {
			return m, m.updateStatsAndLogsForCursor()
		}

		return m, nil

	case actionMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.message = msg.message
		}
		return m, fetchContainers(m.client)

	case statsMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Stats error: %v", msg.err)
		} else {
			m.currentStats = msg.stats
			m.message = ""

			// Store historical data for graphs (shift left and add new value)
			if msg.stats != nil {
				// Shift CPU data left and add new value at the end
				m.cpuHistory = append(m.cpuHistory[1:], msg.stats.CPUPercent)

				// Shift Memory data left and add new value at the end
				m.memoryHistory = append(m.memoryHistory[1:], msg.stats.MemoryPercent)

				// Write to persistent storage
				if m.storage != nil && len(m.containers) > 0 {
					entry := &storage.StatsEntry{
						ContainerID:   m.containers[m.cursor].ID,
						Timestamp:     time.Now(),
						CPUPercent:    msg.stats.CPUPercent,
						MemoryPercent: msg.stats.MemoryPercent,
						MemoryUsage:   msg.stats.MemoryUsage,
						MemoryLimit:   msg.stats.MemoryLimit,
						NetworkRx:     msg.stats.NetworkRx,
						NetworkTx:     msg.stats.NetworkTx,
						BlockRead:     msg.stats.BlockRead,
						BlockWrite:    msg.stats.BlockWrite,
						PIDs:          msg.stats.PIDs,
					}
					m.storage.Write(entry)
				}

				// Update processes if they were fetched
				if len(msg.stats.Processes) > 0 {
					m.currentProcesses = msg.stats.Processes
				}
			}
		}
		return m, waitForStats(m.statsChan, m.statsErrChan)

	case logsMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Logs error: %v", msg.err)
		} else {
			// Only append if the log entry has a message
			if msg.entry.Message != "" {
				m.logs = append(m.logs, msg.entry)
				if len(m.logs) > 1000 {
					m.logs = m.logs[len(m.logs)-1000:]
				}

				// Auto-scroll
				if m.logsAutoScroll {
					m.logsScroll = m.calculateMaxScroll()
				}
			}
		}
		// Keep waiting for the next log line
		return m, m.waitForLogs()
	}

	return m, nil
}

// updateStatsAndLogsForCursor updates stats and logs streaming when the cursor changes
func (m *Model) updateStatsAndLogsForCursor() tea.Cmd {
	if len(m.containers) == 0 {
		return nil
	}

	container := m.containers[m.cursor]

	// Check if we've switched to a different container
	containerChanged := m.currentContainerID != container.ID

	var cmds []tea.Cmd

	// --- Stats streaming ---
	if container.State == "running" {
		// Only restart stats stream if container changed
		if containerChanged || m.statsCancel == nil {
			if m.statsCancel != nil {
				m.statsCancel()
			}
			statsChan, errChan, cancel := m.client.StreamContainerStats(container.ID)
			m.statsCancel = cancel
			m.statsChan = statsChan
			m.statsErrChan = errChan
			cmds = append(cmds, waitForStats(statsChan, errChan))
		}
	} else {
		if m.statsCancel != nil {
			m.statsCancel()
			m.statsCancel = nil
		}
		m.currentStats = nil
	}

	// --- Logs streaming ---
	// Only restart logs if container actually changed
	if containerChanged {
		// Stop previous log stream if any
		if m.logsCancel != nil {
			m.logsCancel()
			m.logsCancel = nil
			m.logsChan = nil
			m.logsErrChan = nil
		}

		// Reset logs and enable autoscroll for new container
		m.logs = []model.LogEntry{}
		m.logsScroll = 0
		m.logsAutoScroll = true

		// Clear historical graph data for new container (pre-filled with zeros)
		m.cpuHistory = make([]float64, m.maxDataPoints)
		m.memoryHistory = make([]float64, m.maxDataPoints)
		m.currentProcesses = nil

		if container.State == "running" {
			logsChan, errChan, cancel := m.client.StreamContainerLogs(container.ID)
			m.logsCancel = cancel
			m.logsChan = logsChan
			m.logsErrChan = errChan
			cmds = append(cmds, waitForLogs(logsChan, errChan))
		}

		// Update the current container ID
		m.currentContainerID = container.ID
	}

	return tea.Batch(cmds...)
}

// waitForLogs creates a command that waits for the next log entry from the model's channels
func (m *Model) waitForLogs() tea.Cmd {
	return func() tea.Msg {
		select {
		case entry, ok := <-m.logsChan:
			if !ok {
				return nil
			}
			return logsMsg{entry: entry}
		case err := <-m.logsErrChan:
			return logsMsg{err: err}
		}
	}
}

// containersListChanged checks if the container list has meaningfully changed
func containersListChanged(old, new []model.Container) bool {
	// Different length means containers were added/removed
	if len(old) != len(new) {
		return true
	}

	// Check if any container ID or state changed
	for i := range old {
		if old[i].ID != new[i].ID || old[i].State != new[i].State {
			return true
		}
	}

	return false
}
