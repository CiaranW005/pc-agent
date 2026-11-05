package telemetry

import (
	"github.com/CiaranW005/go-protocol"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Collect() (protocol.Message, error) {
	// Collect currently reports agent availability only.
	// A running agent returns pc=true; no hardware metrics are collected.
	return protocol.NewMessage(
		protocol.TypeTelemetry,
		struct {
			PC bool `json:"pc"`
		}{
			PC: true,
		},
	), nil
}
