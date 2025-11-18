// internal/model/stats.go
package model

// Stats sisältää containerin resurssitiedot
type Stats struct {
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
}
