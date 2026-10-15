package router

import (
	"fmt"
	"time"

	"github.com/CiaranW005/go-protocol"
	"github.com/CiaranW005/pc-agent/internal/executor"
	"github.com/CiaranW005/pc-agent/internal/telemetry"
)

type Router struct {
	executor    *executor.Executor
	tracker     *protocol.AckTracker
	dedup       *protocol.Deduplicator
	telemetry   *telemetry.Manager
	sendTracked func(protocol.Message, protocol.RetryPolicy) error
}

func NewRouter(
	e *executor.Executor,
	t *protocol.AckTracker,
	sendTracked func(protocol.Message, protocol.RetryPolicy) error,
) *Router {
	return &Router{
		executor:    e,
		tracker:     t,
		dedup:       protocol.NewDeduplicator(5 * time.Minute),
		telemetry:   telemetry.NewManager(),
		sendTracked: sendTracked,
	}
}

func (r *Router) Handle(packet protocol.PacketVariant) (func() error, bool) {
	switch p := packet.(type) {
	case protocol.Message:
		operation := r.handleMessage(p)

		if operation == nil {
			return func() error { return nil }, false
		}

		if r.dedup.IsDuplicate(p.ID) {
			return func() error { return nil }, true
		}

		return operation, true

	case protocol.Acknowledgement:
		r.tracker.Acknowledgement(p)
		return nil, true

	default:
		return nil, false
	}
}

func (r *Router) handleMessage(message protocol.Message) func() error {
	fmt.Printf("Received Message: %T\n", message.Type)

	switch message.Type {
	case protocol.TypeReset:
		return r.executor.Restart

	case protocol.TypePowerOff:
		return r.executor.Shutdown

	case protocol.TypeTelemetry:
		return func() error {
			response, err := r.telemetry.Collect()
			if err != nil {
				return err
			}

			response.ReplyTo = message.ID

			return r.sendTracked(
				response,
				protocol.RetryPolicy{
					Timeout:    time.Second,
					MaxRetries: 1,
				},
			)
		}

	default:
		return nil
	}
}
