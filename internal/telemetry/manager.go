package telemetry

import (
	"github.com/CiaranW005/go-protocol"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Collect() (protocol.Message, error) {
	data := "some_data"

	return protocol.NewMessage(
		protocol.TypeTelemetry,
		data,
	), nil
}
