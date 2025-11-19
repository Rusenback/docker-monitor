// internal/model/stats.go
package model

import "time"

// Stats sisältää containerin resurssitiedot
type Stats struct {
	// CPU
	CPUPercent float64

	// Memory
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	MemoryCache   uint64 // Cache memory (can be freed)

	// Network
	NetworkRx        uint64 // Total bytes received
	NetworkTx        uint64 // Total bytes transmitted
	NetworkRxPackets uint64 // Packets received
	NetworkTxPackets uint64 // Packets transmitted
	NetworkRxErrors  uint64 // RX errors
	NetworkTxErrors  uint64 // TX errors
	NetworkRxDropped uint64 // RX dropped packets
	NetworkTxDropped uint64 // TX dropped packets

	// Block I/O (Disk)
	BlockRead  uint64 // Total bytes read from disk
	BlockWrite uint64 // Total bytes written to disk

	// Processes
	PIDs uint64 // Number of processes/threads

	// Timestamp for rate calculations
	Timestamp time.Time
}
