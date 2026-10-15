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

		operation, success := handler(packet)

		ackType := protocol.AckOk
		if !success {
			fmt.Printf("No handler for packet: %T\n", packet)
			ackType = protocol.AckError
		}

		if operation == nil {
			continue
		}

		ack := protocol.NewAcknowledgement(
			packet.GetID(),
			ackType,
		)

		if err := m.Send(ack); err != nil {
			fmt.Println("Send error:", err)
			return
		}

		// Send a result if the operation fails
		if err := operation(); err != nil {
			fmt.Printf("Command failed: %T: %v\n", packet, err)

			response := protocol.NewMessage(
				protocol.TypeCommandResult,
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

	data = append(data, '\n')

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
