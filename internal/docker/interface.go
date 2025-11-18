// internal/docker/interface.go
package docker

import "github.com/rusenback/docker-monitor/internal/model"

// DockerClient interface mahdollistaa mockauksen testeissä
type DockerClient interface {
	ListContainers() ([]model.Container, error)
	StartContainer(id string) error
	StopContainer(id string) error
	RestartContainer(id string) error
	GetContainerStats(id string) (*model.Stats, error)
	StreamContainerStats(id string) (<-chan *model.Stats, <-chan error, func())
	Close() error
}

// Varmista että Client toteuttaa interfacen
var _ DockerClient = (*Client)(nil)
