package telemetry

import (
	"github.com/CiaranW005/go-protocol"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Collect() (protocol.Message, error) {
	return protocol.NewMessage(
		protocol.TypeTelemetry,
		struct {
			PC bool `json:"pc"`
		}{
			PC: true,
		},
	), nil
}
