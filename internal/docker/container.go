// internal/docker/container.go
package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rusenback/docker-monitor/internal/model"
)

// ListContainers palauttaa kaikki containerit (running + stopped)
func (c *Client) ListContainers() ([]model.Container, error) {
	containers, err := c.cli.ContainerList(c.Ctx, container.ListOptions{
		All: true, // Näytä myös pysäytetyt
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.Container, 0, len(containers))
	for _, cont := range containers {
		// Poista "/" container nimen alusta jos on
		name := cont.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}

		// Muunna portit
		ports := make([]model.Port, 0)
		for _, p := range cont.Ports {
			ports = append(ports, model.Port{
				Private: int(p.PrivatePort),
				Public:  int(p.PublicPort),
				Type:    p.Type,
			})
		}

		result = append(result, model.Container{
			ID:      cont.ID[:12], // Lyhyt ID
			Name:    name,
			Image:   cont.Image,
			Status:  cont.Status,
			State:   cont.State,
			Created: time.Unix(cont.Created, 0),
			Ports:   ports,
		})
	}

	return result, nil
}

// StartContainer käynnistää containerin
func (c *Client) StartContainer(id string) error {
	Ctx, cancel := context.WithTimeout(c.Ctx, 10*time.Second)
	defer cancel()

	return c.cli.ContainerStart(Ctx, id, container.StartOptions{})
}

// StopContainer pysäyttää containerin
func (c *Client) StopContainer(id string) error {
	Ctx, cancel := context.WithTimeout(c.Ctx, 10*time.Second)
	defer cancel()

	timeout := 10 // Sekuntia
	return c.cli.ContainerStop(Ctx, id, container.StopOptions{
		Timeout: &timeout,
	})
}

// RestartContainer uudelleenkäynnistää containerin
func (c *Client) RestartContainer(id string) error {
	Ctx, cancel := context.WithTimeout(c.Ctx, 20*time.Second)
	defer cancel()

	timeout := 10
	return c.cli.ContainerRestart(Ctx, id, container.StopOptions{
		Timeout: &timeout,
	})
}
