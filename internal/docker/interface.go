// internal/docker/interface.go
package docker

import "github.com/rusenback/docker-monitor/internal/model"

// DockerClient interface allows mocking in tests
type DockerClient interface {
	ListContainers() ([]model.Container, error)
	StartContainer(id string) error
	StopContainer(id string) error
	RestartContainer(id string) error
	GetContainerStats(id string) (*model.Stats, error)
	StreamContainerStats(id string) (<-chan *model.Stats, <-chan error, func())

	GetContainerLogs(id string, tail int) ([]model.LogEntry, error)
	StreamContainerLogs(id string) (<-chan model.LogEntry, <-chan error, func())

	Close() error
}

// Ensure Client implements the interface
var _ DockerClient = (*Client)(nil)
