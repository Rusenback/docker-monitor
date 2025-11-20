// internal/docker/stats.go
package docker

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/rusenback/docker-monitor/internal/model"
)

// GetContainerStats retrieves container resource statistics
func (c *Client) GetContainerStats(id string) (*model.Stats, error) {
	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	// Fetch stats (stream: false = fetch only once)
	resp, err := c.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	return parseStats(&stats), nil
}

// parseStats converts Docker API's StatsJSON structure to model.Stats structure
func parseStats(stats *types.StatsJSON) *model.Stats {
	// Calculate CPU percentage
	cpuPercent := calculateCPUPercent(stats)

	// Memory information
	memUsage := stats.MemoryStats.Usage
	memLimit := stats.MemoryStats.Limit
	memPercent := float64(0)
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	// Memory cache (this is often a large part of "usage" but can be freed)
	memCache := stats.MemoryStats.Stats["cache"]

	// Network information - including packets, errors and dropped
	var networkRx, networkTx uint64
	var networkRxPackets, networkTxPackets uint64
	var networkRxErrors, networkTxErrors uint64
	var networkRxDropped, networkTxDropped uint64

	for _, network := range stats.Networks {
		networkRx += network.RxBytes
		networkTx += network.TxBytes
		networkRxPackets += network.RxPackets
		networkTxPackets += network.TxPackets
		networkRxErrors += network.RxErrors
		networkTxErrors += network.TxErrors
		networkRxDropped += network.RxDropped
		networkTxDropped += network.TxDropped
	}

	// Block I/O (Disk) tiedot
	var blockRead, blockWrite uint64
	for _, bioEntry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch bioEntry.Op {
		case "Read":
			blockRead += bioEntry.Value
		case "Write":
			blockWrite += bioEntry.Value
		}
	}

	// PIDs (number of processes)
	pids := stats.PidsStats.Current

	return &model.Stats{
		CPUPercent:       cpuPercent,
		MemoryUsage:      memUsage,
		MemoryLimit:      memLimit,
		MemoryPercent:    memPercent,
		MemoryCache:      memCache,
		NetworkRx:        networkRx,
		NetworkTx:        networkTx,
		NetworkRxPackets: networkRxPackets,
		NetworkTxPackets: networkTxPackets,
		NetworkRxErrors:  networkRxErrors,
		NetworkTxErrors:  networkTxErrors,
		NetworkRxDropped: networkRxDropped,
		NetworkTxDropped: networkTxDropped,
		BlockRead:        blockRead,
		BlockWrite:       blockWrite,
		PIDs:             pids,
		Timestamp:        stats.Read,
	}
}

// calculateCPUPercent calculates CPU usage as a percentage
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(
			len(stats.CPUStats.CPUUsage.PercpuUsage),
		) * 100.0
		return cpuPercent
	}
	return 0.0
}

// StreamContainerStats streams container statistics
// Returns a channel for reading stats and an error channel
func (c *Client) StreamContainerStats(id string) (<-chan *model.Stats, <-chan error, func()) {
	statsChan := make(chan *model.Stats)
	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(c.Ctx)

	go func() {
		defer close(statsChan)
		defer close(errChan)

		resp, err := c.cli.ContainerStats(ctx, id, true) // stream: true
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		updateCounter := 0
		var lastProcesses []model.Process

		for {
			var stats types.StatsJSON
			if err := decoder.Decode(&stats); err != nil {
				if err == io.EOF || err == context.Canceled {
					return
				}
				errChan <- err
				return
			}

			// Use the shared parseStats function
			parsedStats := parseStats(&stats)

			// Fetch processes on first update and then every 10th update
			updateCounter++
			if updateCounter == 1 || updateCounter%10 == 0 {
				// Fetch processes synchronously but with short timeout
				processCtx, processCancel := context.WithTimeout(ctx, 1*time.Second)
				processes, err := c.getContainerProcessesWithContext(processCtx, id)
				processCancel()

				if err == nil {
					lastProcesses = processes
					parsedStats.Processes = processes
				} else if len(lastProcesses) > 0 {
					// Use cached processes if fetch fails
					parsedStats.Processes = lastProcesses
				}
			} else if len(lastProcesses) > 0 {
				// Use last fetched processes for intermediate updates
				parsedStats.Processes = lastProcesses
			}

			select {
			case statsChan <- parsedStats:
			case <-ctx.Done():
				return
			}
		}
	}()

	return statsChan, errChan, cancel
}

// getContainerProcessesWithContext retrieves processes with a custom context
func (c *Client) getContainerProcessesWithContext(ctx context.Context, id string) ([]model.Process, error) {
	// Call ContainerTop to get process list
	top, err := c.cli.ContainerTop(ctx, id, []string{"aux"})
	if err != nil {
		return nil, err
	}

	processes := make([]model.Process, 0)

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

// Helper function to safely get a value from a slice
func getOrEmpty(slice []string, index int) string {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return ""
}
