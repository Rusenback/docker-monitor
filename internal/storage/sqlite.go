package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// TimeRange represents different time window options
type TimeRange int

const (
	Range30Min TimeRange = iota
	Range1Hour
	Range6Hour
	Range1Day
	Range1Week
)

func (t TimeRange) String() string {
	switch t {
	case Range30Min:
		return "30min"
	case Range1Hour:
		return "1hour"
	case Range6Hour:
		return "6hours"
	case Range1Day:
		return "1day"
	case Range1Week:
		return "1week"
	default:
		return "unknown"
	}
}

// Duration returns the time duration for the range
func (t TimeRange) Duration() time.Duration {
	switch t {
	case Range30Min:
		return 30 * time.Minute
	case Range1Hour:
		return 1 * time.Hour
	case Range6Hour:
		return 6 * time.Hour
	case Range1Day:
		return 24 * time.Hour
	case Range1Week:
		return 7 * 24 * time.Hour
	default:
		return 30 * time.Minute
	}
}

// DataPoint represents a single data point in time
type DataPoint struct {
	Timestamp     time.Time
	CPUPercent    float64
	MemoryPercent float64
}

// Storage handles persistent statistics storage
type Storage struct {
	db        *sql.DB
	writeChan chan *StatsEntry
	closeChan chan struct{}
}

// StatsEntry represents a stats entry to be written
type StatsEntry struct {
	ContainerID   string
	Timestamp     time.Time
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	NetworkRx     uint64
	NetworkTx     uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
}

// NewStorage creates a new storage instance
func NewStorage() (*Storage, error) {
	// Create data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".dockermon")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database
	dbPath := filepath.Join(dataDir, "stats.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	storage := &Storage{
		db:        db,
		writeChan: make(chan *StatsEntry, 1000),
		closeChan: make(chan struct{}),
	}

	// Start background writer
	go storage.writer()

	// Start cleanup routine
	go storage.cleanup()

	return storage, nil
}

// createTables creates the database schema
func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS container_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		container_id TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		cpu_percent REAL,
		memory_percent REAL,
		memory_usage INTEGER,
		memory_limit INTEGER,
		network_rx INTEGER,
		network_tx INTEGER,
		block_read INTEGER,
		block_write INTEGER,
		pids INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_container_time
	ON container_stats(container_id, timestamp);

	CREATE TABLE IF NOT EXISTS containers (
		id TEXT PRIMARY KEY,
		name TEXT,
		image TEXT,
		first_seen INTEGER,
		last_seen INTEGER
	);
	`

	_, err := db.Exec(schema)
	return err
}

// Write queues a stats entry for writing
func (s *Storage) Write(entry *StatsEntry) {
	select {
	case s.writeChan <- entry:
		// Successfully queued
	default:
		// Channel full, drop silently to avoid blocking
		// This is acceptable for metrics collection
	}
}

// writer runs in background and batch writes to database
func (s *Storage) writer() {
	buffer := make([]*StatsEntry, 0, 100)
	ticker := time.NewTicker(5 * time.Second) // Flush more frequently
	defer ticker.Stop()

	for {
		select {
		case entry := <-s.writeChan:
			buffer = append(buffer, entry)

			// Batch write when buffer reaches 50 entries (more frequent writes)
			if len(buffer) >= 50 {
				s.batchWrite(buffer)
				buffer = buffer[:0]
			}

		case <-ticker.C:
			// Periodic flush every 5 seconds
			if len(buffer) > 0 {
				s.batchWrite(buffer)
				buffer = buffer[:0]
			}

		case <-s.closeChan:
			// Final flush on close
			if len(buffer) > 0 {
				s.batchWrite(buffer)
			}
			return
		}
	}
}

// batchWrite writes a batch of entries to the database
func (s *Storage) batchWrite(entries []*StatsEntry) {
	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO container_stats
		(container_id, timestamp, cpu_percent, memory_percent,
		 memory_usage, memory_limit, network_rx, network_tx,
		 block_read, block_write, pids)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, entry := range entries {
		_, err := stmt.Exec(
			entry.ContainerID,
			entry.Timestamp.Unix(),
			entry.CPUPercent,
			entry.MemoryPercent,
			entry.MemoryUsage,
			entry.MemoryLimit,
			entry.NetworkRx,
			entry.NetworkTx,
			entry.BlockRead,
			entry.BlockWrite,
			entry.PIDs,
		)
		if err != nil {
			continue
		}
	}

	tx.Commit()
}

// Query retrieves data points for a container and time range
func (s *Storage) Query(containerID string, timeRange TimeRange) ([]DataPoint, error) {
	cutoff := time.Now().Add(-timeRange.Duration()).Unix()

	var query string
	var bucketSize int64

	// Choose aggregation based on time range
	switch timeRange {
	case Range30Min:
		// Full resolution (no aggregation)
		query = `
			SELECT timestamp, cpu_percent, memory_percent
			FROM container_stats
			WHERE container_id = ? AND timestamp > ?
			ORDER BY timestamp ASC
		`
		rows, err := s.db.Query(query, containerID, cutoff)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return s.scanRows(rows)

	case Range1Hour:
		bucketSize = 30 // 30 second buckets

	case Range6Hour:
		bucketSize = 300 // 5 minute buckets

	case Range1Day:
		bucketSize = 600 // 10 minute buckets

	case Range1Week:
		bucketSize = 3600 // 1 hour buckets
	}

	// Aggregated query
	query = `
		SELECT
			(timestamp / ?) * ? as bucket,
			AVG(cpu_percent) as avg_cpu,
			AVG(memory_percent) as avg_mem
		FROM container_stats
		WHERE container_id = ? AND timestamp > ?
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	rows, err := s.db.Query(query, bucketSize, bucketSize, containerID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// scanRows scans database rows into DataPoints
func (s *Storage) scanRows(rows *sql.Rows) ([]DataPoint, error) {
	var points []DataPoint

	for rows.Next() {
		var timestamp int64
		var cpu, mem float64

		if err := rows.Scan(&timestamp, &cpu, &mem); err != nil {
			continue
		}

		points = append(points, DataPoint{
			Timestamp:     time.Unix(timestamp, 0),
			CPUPercent:    cpu,
			MemoryPercent: mem,
		})
	}

	return points, rows.Err()
}

// cleanup removes old data periodically
func (s *Storage) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Delete data older than 7 days in batches to avoid locking
			cutoff := time.Now().Add(-7 * 24 * time.Hour).Unix()
			s.batchDelete(cutoff)

		case <-s.closeChan:
			return
		}
	}
}

// batchDelete removes old records in batches to prevent long-running locks
func (s *Storage) batchDelete(cutoffTimestamp int64) {
	const batchSize = 1000
	for {
		result, err := s.db.Exec(
			"DELETE FROM container_stats WHERE timestamp < ? LIMIT ?",
			cutoffTimestamp,
			batchSize,
		)
		if err != nil {
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			// No more rows to delete
			return
		}

		// Small sleep to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}
}

// Close closes the storage
func (s *Storage) Close() error {
	close(s.closeChan)
	time.Sleep(100 * time.Millisecond) // Allow goroutines to finish
	return s.db.Close()
}
