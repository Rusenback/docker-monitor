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

			// Fetch processes on first update and then every 5th update
			updateCounter++
			if updateCounter == 1 || updateCounter%5 == 0 {
				processes, err := c.GetContainerProcesses(id)
				if err == nil {
					parsedStats.Processes = processes
				}
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
