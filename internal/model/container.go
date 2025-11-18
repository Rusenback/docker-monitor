package model

import "time"

// Container edustaa Docker containeria
type Container struct {
	ID            string
	Name          string
	Image         string
	Status        string
	State         string
	Created       time.Time
	Ports         []Port
	DisplayStatus string
}

// Port edustaa container porttia
type Port struct {
	Private int
	Public  int
	Type    string
}
