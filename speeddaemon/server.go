package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
)

// SpeedLimitEnforcementServer coordinates enforcement of average speed limits on the Freedom Island road network.
//
// Two types of clients are supported: cameras and ticket dispatchers.
// Clients connect over TCP and speak a protocol using a binary format.
//
// When the client does something that this protocol specification declares "an error", the server must send the
// client an appropriate Error message and immediately disconnect that client.
type SpeedLimitEnforcementServer struct {
}

var IllegalMessageType = &ErrorMessage{Msg: "illegal message"}

// Handle handles a client connection.
func (s *SpeedLimitEnforcementServer) Handle(client net.Conn) error {
	defer closeOrLog(client)

	var t uint8
	if err := binary.Read(client, binary.BigEndian, &t); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	switch t {
	case IAmCameraMessageType:
		return s.handleCamera(client)
	case IAmDispatcherMessageType:
		return s.handleDispatcher(client)
	default:
		slog.Error("unexpected message type", "type", t)
		return sendError(client, IllegalMessageType)
	}
}

var AlreadyIdentifiedError = &ErrorMessage{Msg: "client has already identified itself"}

func (s *SpeedLimitEnforcementServer) handleCamera(client net.Conn) error {
	m, err := readIAmCameraMessage(client)
	if err != nil {
		return fmt.Errorf("error reading IAmCamera message: %w", err)
	}

	c := Camera{Road: m.Road, Mile: m.Mile, Limit: m.Limit}
	slog.Info("camera connected", "road", c.Road, "mile", c.Mile, "limit", c.Limit)

	// TODO: Handle camera.

	for {
		var t uint8
		// TODO: Should we ever disconnect client?
		if err := binary.Read(client, binary.BigEndian, &t); err != nil {
			return fmt.Errorf("error reading message type: %w", err)
		}

		switch t {
		case PlateMessageType:
			// TODO: Implement.
			return nil
		case WantHeartbeatMessageType:
			// TODO: Implement.
			//It is an error for a client to send multiple WantHeartbeat messages on a single connection.
			return nil
		case IAmCameraMessageType:
			return sendError(client, AlreadyIdentifiedError)
		case IAmDispatcherMessageType:
			return sendError(client, AlreadyIdentifiedError)
		default:
			return sendError(client, IllegalMessageType)
		}
	}
}

func (s *SpeedLimitEnforcementServer) handleDispatcher(client net.Conn) error {
	m, err := readIAmDispatcherMessage(client)
	if err != nil {
		return fmt.Errorf("error reading IAmCamera message: %w", err)
	}

	d := TicketDispatcher{Roads: m.Roads}
	slog.Info("dispatcher connected", "roads", d.Roads)

	for {
		var t uint8
		// TODO: Should we ever disconnect client?
		if err := binary.Read(client, binary.BigEndian, &t); err != nil {
			return fmt.Errorf("error reading message type: %w", err)
		}

		switch t {
		case PlateMessageType:
			return sendError(client, IllegalMessageType)
		case WantHeartbeatMessageType:
			// TODO: Implement.
			// It is an error for a client to send multiple WantHeartbeat messages on a single connection.
			return nil
		case IAmCameraMessageType:
			return sendError(client, AlreadyIdentifiedError)
		case IAmDispatcherMessageType:
			return sendError(client, AlreadyIdentifiedError)
		default:
			return sendError(client, IllegalMessageType)
		}
	}
}

// sendError sends an ErrorMessage to the client.
func sendError(client net.Conn, m *ErrorMessage) error {
	bytes, err := m.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	_, err = client.Write(bytes)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	return nil
}

// TODO: Move to utility package reusable for other problems.
func closeOrLog(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("error closing connection", "err", err, "remote_addr", conn.RemoteAddr())
	}
}
