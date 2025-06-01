package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
)

// SpeedLimitEnforcementServer coordinates enforcement of average speed limits on the Freedom Island road network.
//
// Two types of clients are supported: cameras and ticket dispatchers.
// Clients connect over TCP and speak a protocol using a binary format.
//
// When the client does something that this protocol specification declares "an error", the server must send the
// client an appropriate Error message and immediately disconnect that client.
type SpeedLimitEnforcementServer struct {
	ConnectionID atomic.Uint64

	CameraHandler     *CameraHandler
	DispatcherHandler *DispatcherHandler
	Records           <-chan CameraRecord
}

var MultipleWantHeartbeatMessagesError = &ErrorMessage{Msg: "multiple WantHeartbeat messages"}

// Handle handles a client connection.
func (s *SpeedLimitEnforcementServer) Handle(conn net.Conn) error {
	client := &Conn{Conn: conn, ID: s.ConnectionID.Add(1)}
	slog.Info("client connected", "connection", client.ID)
	defer closeOrLog(client)

	for {
		var t uint8
		if err := binary.Read(client, binary.BigEndian, &t); err != nil {
			return fmt.Errorf("read error: %w", err)
		}
		switch t {
		case IAmCameraMessageType:
			return s.CameraHandler.handleCamera(client)
		case IAmDispatcherMessageType:
			return s.DispatcherHandler.handleDispatcher(client)
		case WantHeartbeatMessageType:
			// It is an error for a client to send multiple WantHeartbeat messages on a single connection.
			if client.Heartbeat != nil {
				return sendError(client, MultipleWantHeartbeatMessagesError)
			}
			if err := beginHeartbeat(client); err != nil {
				return fmt.Errorf("error beginning heartbeat: %w", err)
			}
		default:
			slog.Error("unexpected message type", "type", t)
			return sendError(client, illegalMessage(t))
		}
	}
}

// TODO: Accept context and terminate on shutdown.
func (s *SpeedLimitEnforcementServer) EnforceSpeedLimit() error {
	for r := range s.Records {
		slog.Debug("checking tickets", "plate", r.Plate)
		plate := r.PlateMessage.Plate
		records := s.CameraHandler.FetchPlateRecords(plate)

		var recordsOnRoad []CameraRecord
		for _, other := range records {
			if r == other {
				continue
			}
			if other.Camera.Road == r.Camera.Road {
				recordsOnRoad = append(recordsOnRoad, other)
			}
		}
		fmt.Println(r)

		for _, other := range recordsOnRoad {
			fmt.Println(other)

			distance := float64(max(r.Camera.Mile, other.Camera.Mile) - min(r.Camera.Mile, other.Camera.Mile))
			fmt.Println(distance)
			duration := float64(max(r.Timestamp, other.Timestamp) - min(r.Timestamp, other.Timestamp))
			fmt.Println(duration)
			mph := (distance / duration) * 3600
			fmt.Println(mph)
			// TODO: .5 over is ok, right?
			if mph > float64(r.Limit) {
				fmt.Println("sending ticket")
				s.DispatcherHandler.SendTicket(r, other, mph)
			}
		}

	}
	return nil
}

var AlreadyIdentifiedError = &ErrorMessage{Msg: "client has already identified itself"}

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

func illegalMessage(t uint8) *ErrorMessage {
	return &ErrorMessage{Msg: fmt.Sprintf("illegal message: %02X", t)}
}
