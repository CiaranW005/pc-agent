package connection

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/CiaranW005/go-protocol"
)

type Manager struct {
	listener net.Listener
	conn     net.Conn
	reader   *bufio.Reader
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) StartSession(address string, handler func(protocol.PacketVariant) func() error) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	m.listener = listener
	defer listener.Close()

	fmt.Println("Waiting for ESP32 connection on", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		m.conn = conn
		m.reader = bufio.NewReader(conn)

		fmt.Println("ESP32 connected from", conn.RemoteAddr())

		m.listen(handler)

		conn.Close()
		m.conn = nil
		m.reader = nil

		fmt.Println("ESP32 disconnected. Waiting for reconnection...")
	}
}

func (m *Manager) listen(handler func(protocol.PacketVariant) func() error) {
	for {
		packet, err := m.Receive()
		if err != nil {
			if err != io.EOF {
				fmt.Println("Receive error:", err)
			}
			return
		}

		fmt.Printf("Received packet: %T\n", packet)

		operation := handler(packet)

		if operation == nil {
			fmt.Printf("No handler for packet: %T\n", packet)
			continue
		}

		ack := protocol.NewAcknowledgement(
			uint64(time.Now().UnixMilli()),
			protocol.AckOk,
			"",
		)

		if err := m.Send(ack); err != nil {
			fmt.Println("Send error:", err)
			return
		}

		if err := operation(); err != nil {
			fmt.Printf("Command failed: %T: %v\n", packet, err)

			if sendErr := m.Send(protocol.NewAcknowledgement(
				uint64(time.Now().UnixMilli()),
				protocol.AckError,
				err.Error(),
			)); sendErr != nil {
				fmt.Println("Failed to send error acknowledgment:", sendErr)
				return
			}

			continue
		}
	}
}

func (m *Manager) Send(packet protocol.PacketVariant) error {
	data, err := protocol.SerialisePacket(packet)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = m.conn.Write(data)
	return err
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
