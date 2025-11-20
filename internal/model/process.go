package model

// Process represents a running process in a container
type Process struct {
	PID     string
	User    string
	CPU     string
	Memory  string
	Command string
}
