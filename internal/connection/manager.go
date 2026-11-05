package connection

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/CiaranW005/go-protocol"
)

type Manager struct {
	listener net.Listener
	conn     net.Conn
	reader   *bufio.Reader

	writeMu sync.Mutex
	tracker *protocol.AckTracker
}

func NewManager() *Manager {
	m := &Manager{}

	m.tracker = protocol.NewAckTracker(
		protocol.TrackerCallbacks{
			Retry: func(message protocol.Message) error {
				return m.Send(message)
			},

			Failed: func(message protocol.Message, err error) {
				fmt.Printf(
					"Message %s failed %v\n",
					message.ID,
					err,
				)
			},

			Acked: func(message protocol.Message, ack protocol.Acknowledgement) {
				fmt.Printf(
					"Message %s acknowledged %s\n",
					message.ID,
					ack.Status,
				)
			},
		},
	)

	return m
}

func (m *Manager) StartSession(address string, handler func(protocol.PacketVariant) (func() error, bool)) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	m.listener = listener
	defer listener.Close()

	fmt.Println("Waiting for ESP32 connection on", address)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.tracker.Run(ctx)

	// Handle one connection at a time. Accept the next connection only after
	// the current session ends and its pending acknowledgements are cleared.
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		m.writeMu.Lock()
		m.conn = conn
		m.writeMu.Unlock()

		m.reader = bufio.NewReader(conn)

		fmt.Println("ESP32 connected from", conn.RemoteAddr())

		m.listen(handler)

		m.writeMu.Lock()
		conn.Close()
		m.conn = nil
		m.writeMu.Unlock()

		m.reader = nil
		m.tracker.Clear()

		fmt.Println("ESP32 disconnected. Waiting for reconnection...")
	}
}

func (m *Manager) listen(handler func(protocol.PacketVariant) (func() error, bool)) {
	for {
		packet, err := m.Receive()
		if err != nil {
			if err != io.EOF {
				fmt.Println("Receive error:", err)
			}
			return
		}

		fmt.Printf("Received packet: %T\n", packet)

		// The handler returns a deferred operation and whether the packet was accepted.
		// A nil operation skips acknowledgement and execution.
		operation, success := handler(packet)

		ackType := protocol.AckOk
		if !success {
			fmt.Printf("No handler for packet: %T\n", packet)
			ackType = protocol.AckError
		}

		if operation == nil {
			continue
		}

		// Acknowledge before executing because shutdown or reboot may disconnect
		// the agent before another packet can be sent.
		ack := protocol.NewAcknowledgement(
			packet.GetID(),
			ackType,
		)

		if err := m.Send(ack); err != nil {
			fmt.Println("Send error:", err)
			return
		}

		// Report execution failures explicitly. Successful shutdown and reboot
		// requests do not produce a completion response.
		if err := operation(); err != nil {
			fmt.Printf("Command failed: %T: %v\n", packet, err)

			response := protocol.NewMessage(
				packet.(protocol.Message).Type,
				protocol.CommandResult{
					Success: false,
					Error:   err.Error(),
				},
			)

			response.ReplyTo = packet.GetID()

			if err := m.SendTracked(
				response,
				protocol.RetryPolicy{
					Timeout:    3 * time.Second,
					MaxRetries: 2,
				}); err != nil {
				fmt.Println("Failed to send command result:", err)
				return
			}
		}
	}
}

func (m *Manager) Send(packet protocol.PacketVariant) error {
	data, err := protocol.SerialisePacket(packet)
	if err != nil {
		return err
	}

	// Delimit JSON packets with newlines so the receiver can separate messages
	// even when TCP combines or splits writes.
	data = append(data, '\n')

	// Only let one goroutine send at a time so two packets cannot get mixed together.
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if m.conn == nil {
		return fmt.Errorf("No active connection")
	}

	_, err = m.conn.Write(data)
	return err
}

func (m *Manager) SendTracked(
	message protocol.Message,
	policy protocol.RetryPolicy,
) error {
	// Register before sending so an immediate acknowledgement can be matched.
	// Cancel tracking if the initial send fails.
	if err := m.tracker.Track(message, policy); err != nil {
		return err
	}

	if err := m.Send(message); err != nil {
		m.tracker.Cancel(message.ID)
		return err
	}

	return nil
}

func (m *Manager) Receive() (protocol.PacketVariant, error) {
	data, err := m.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return protocol.DeserialisePacket(data)
}

func (m *Manager) CloseSession() error {
	if m.conn != nil {
		m.conn.Close()
	}

	if m.listener != nil {
		return m.listener.Close()
	}

	return nil
}

func (m *Manager) Tracker() *protocol.AckTracker {
	return m.tracker
}
