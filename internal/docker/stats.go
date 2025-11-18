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

// GetContainerStats hakee containerin resurssitiedot
func (c *Client) GetContainerStats(id string) (*model.Stats, error) {
	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	// Hae stats (stream: false = hae vain kerran)
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

	// Laske CPU prosentti
	cpuPercent := calculateCPUPercent(&stats)

	// Memory tiedot
	memUsage := stats.MemoryStats.Usage
	memLimit := stats.MemoryStats.Limit
	memPercent := float64(0)
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	// Network tiedot
	var networkRx, networkTx uint64
	for _, network := range stats.Networks {
		networkRx += network.RxBytes
		networkTx += network.TxBytes
	}

	return &model.Stats{
		CPUPercent:    cpuPercent,
		MemoryUsage:   memUsage,
		MemoryLimit:   memLimit,
		MemoryPercent: memPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
	}, nil
}

// calculateCPUPercent laskee CPU käytön prosentteina
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

// StreamContainerStats streamaa containerin statsit
// Palauttaa channel josta voi lukea statseja ja error channelin
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
		for {
			var stats types.StatsJSON
			if err := decoder.Decode(&stats); err != nil {
				if err == io.EOF || err == context.Canceled {
					return
				}
				errChan <- err
				return
			}

			cpuPercent := calculateCPUPercent(&stats)
			memUsage := stats.MemoryStats.Usage
			memLimit := stats.MemoryStats.Limit
			memPercent := float64(0)
			if memLimit > 0 {
				memPercent = float64(memUsage) / float64(memLimit) * 100.0
			}

			var networkRx, networkTx uint64
			for _, network := range stats.Networks {
				networkRx += network.RxBytes
				networkTx += network.TxBytes
			}

			select {
			case statsChan <- &model.Stats{
				CPUPercent:    cpuPercent,
				MemoryUsage:   memUsage,
				MemoryLimit:   memLimit,
				MemoryPercent: memPercent,
				NetworkRx:     networkRx,
				NetworkTx:     networkTx,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return statsChan, errChan, cancel
}
