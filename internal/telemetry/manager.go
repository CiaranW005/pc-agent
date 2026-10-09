package telemetry

import (
	"time"

	"github.com/CiaranW005/go-protocol"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Collect() (protocol.Message, error) {
	data := "some_data"

	return protocol.NewMessage(
		uint64(time.Now().UnixMilli()),
		protocol.TypeTelemetry,
		data,
	), nil
}
