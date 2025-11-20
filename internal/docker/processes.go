package docker

import (
	"context"
	"time"

	"github.com/rusenback/docker-monitor/internal/model"
)

// GetContainerProcesses retrieves the top processes running in a container
func (c *Client) GetContainerProcesses(id string) ([]model.Process, error) {
	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	// Call ContainerTop to get process list
	// Args: ps -aux format for better information
	top, err := c.cli.ContainerTop(ctx, id, []string{"aux"})
	if err != nil {
		return nil, err
	}

	// Parse the response
	processes := make([]model.Process, 0)

	// top.Titles contains column names like ["USER", "PID", "%CPU", "%MEM", "COMMAND"]
	// top.Processes contains rows of data

	// Find column indices
	pidIdx, userIdx, cpuIdx, memIdx, cmdIdx := -1, -1, -1, -1, -1
	for i, title := range top.Titles {
		switch title {
		case "PID":
			pidIdx = i
		case "USER":
			userIdx = i
		case "%CPU":
			cpuIdx = i
		case "%MEM":
			memIdx = i
		case "COMMAND", "CMD":
			cmdIdx = i
		}
	}

	// Parse each process
	for _, proc := range top.Processes {
		if len(proc) <= pidIdx || len(proc) <= userIdx ||
			len(proc) <= cpuIdx || len(proc) <= memIdx || len(proc) <= cmdIdx {
			continue
		}

		process := model.Process{
			PID:     getOrEmpty(proc, pidIdx),
			User:    getOrEmpty(proc, userIdx),
			CPU:     getOrEmpty(proc, cpuIdx),
			Memory:  getOrEmpty(proc, memIdx),
			Command: getOrEmpty(proc, cmdIdx),
		}

		processes = append(processes, process)
	}

	// Limit to top 10
	if len(processes) > 10 {
		processes = processes[:10]
	}

	return processes, nil
}
