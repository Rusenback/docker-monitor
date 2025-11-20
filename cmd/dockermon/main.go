// cmd/dockermon/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusenback/docker-monitor/internal/docker"
	"github.com/rusenback/docker-monitor/internal/storage"
	"github.com/rusenback/docker-monitor/internal/tui"
)

func main() {
	// Create Docker client
	cfg := docker.DefaultConfig()
	client, err := docker.NewClient(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Docker: %v\n", err)
		fmt.Println("\nMake sure Docker is running:")
		fmt.Println("  sudo systemctl start docker")
		fmt.Println("  sudo usermod -aG docker $USER")
		os.Exit(1)
	}
	defer client.Close()

	// Create storage
	store, err := storage.NewStorage()
	if err != nil {
		fmt.Printf("❌ Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Create TUI model
	m := tui.NewModel(client, store)

	// Start TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
