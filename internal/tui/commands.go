package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusenback/docker-monitor/internal/docker"
	"github.com/rusenback/docker-monitor/internal/model"
)

// tickCmd creates a command that sends a tick message every 2 seconds
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchContainers creates a command to fetch the container list
func fetchContainers(client docker.DockerClient) tea.Cmd {
	return func() tea.Msg {
		containers, err := client.ListContainers()
		for i := range containers {
			c := &containers[i]
			switch c.State {
			case "running":
				c.DisplayStatus = truncate(c.Status, 30)
			case "exited":
				c.DisplayStatus = "exited"
			default:
				c.DisplayStatus = c.State
			}
		}
		return containersMsg{containers: containers, err: err}
	}
}

// waitForStats creates a command that waits for the next stats message
func waitForStats(statsChan <-chan *model.Stats, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case stats := <-statsChan:
			return statsMsg{stats: stats, err: nil}
		case err := <-errChan:
			return statsMsg{stats: nil, err: err}
		}
	}
}

// waitForLogs creates a command that waits for the next log entry
func waitForLogs(logsChan <-chan model.LogEntry, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case entry, ok := <-logsChan:
			if !ok {
				return nil
			}
			return logsMsg{entry: entry, err: nil}
		case err := <-errChan:
			return logsMsg{err: err}
		}
	}
}

// waitForLogsStream creates a command that continuously waits for log entries
func waitForLogsStream(logsChan <-chan model.LogEntry, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case entry := <-logsChan:
				return logsMsg{entry: entry, err: nil}
			case err := <-errChan:
				return logsMsg{err: err}
			}
		}
	}
}

// startContainer creates a command to start a container
func startContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{
			message: fmt.Sprintf("Started: %s", name),
			err:     client.StartContainer(id),
		}
	}
}

// stopContainer creates a command to stop a container
func stopContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{
			message: fmt.Sprintf("Stopped: %s", name),
			err:     client.StopContainer(id),
		}
	}
}

// restartContainer creates a command to restart a container
func restartContainer(client docker.DockerClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{
			message: fmt.Sprintf("Restarted: %s", name),
			err:     client.RestartContainer(id),
		}
	}
}
