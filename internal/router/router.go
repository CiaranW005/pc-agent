package router

import (
	"github.com/CiaranW005/go-protocol"
	"github.com/CiaranW005/pc-agent/internal/executor"
)

type Router struct {
	executor *executor.Executor
}

func NewRouter(e *executor.Executor) *Router {
	return &Router{executor: e}
}

func (r *Router) Handle(packet protocol.PacketVariant) func() error {
	switch p := packet.(type) {
	case protocol.Message:
		return r.handleMessage(p)

	case protocol.Acknowledgement:
		return nil
	}

	return nil
}

func (r *Router) handleMessage(message protocol.Message) func() error {
	switch message.Type {
	case protocol.TypeReset:
		return r.executor.Restart

	case protocol.TypePowerOff:
		return r.executor.Shutdown

	case protocol.TypeTelemetry:
		return nil

	default:
		return nil
	}
}
