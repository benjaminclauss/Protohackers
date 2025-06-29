package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync"
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
	mu           sync.Mutex

	CameraHandler     *CameraHandler
	DispatcherHandler *DispatcherHandler
	Records           <-chan CameraRecord

	// TODO: Please polish...
	TicketsSent map[TicketOnDay]bool
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

		for _, other := range recordsOnRoad {
			distance := float64(max(r.Camera.Mile, other.Camera.Mile) - min(r.Camera.Mile, other.Camera.Mile))
			duration := float64(max(r.Timestamp, other.Timestamp) - min(r.Timestamp, other.Timestamp))
			mph := (distance / duration) * 3600

			// It is always required to ticket a car exceeding the speed limit by 0.5 mph or more.
			// In cases where the car is exceeding the speed limit by less than 0.5 mph, it is acceptable to omit the ticket.
			if float64(mph) > float64(r.Limit)+0.5 {
				t := ticket(r, other, mph)
				s.sendTicket(t)
			}
		}

		slog.Debug("finished checking tickets", "plate", r.Plate)
	}
	return nil
}

func ticket(r CameraRecord, other CameraRecord, mph float64) TicketMessage {
	var earlier, later CameraRecord
	// mile1 and timestamp1 must refer to the earlier of the 2 observations (the smaller timestamp), and mile2 and timestamp2 must refer to the later of the 2 observations (the larger timestam
	if r.Timestamp < other.Timestamp {
		earlier = r
		later = other
	} else {
		earlier = other
		later = r
	}

	return TicketMessage{
		Plate:      r.Plate,
		Road:       r.Road,
		Mile1:      earlier.Camera.Mile,
		Timestamp1: earlier.Timestamp,
		Mile2:      later.Camera.Mile,
		Timestamp2: later.Timestamp,
		Speed:      uint16(mph * 100),
	}
}

func (s *SpeedLimitEnforcementServer) sendTicket(t TicketMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d1 := day(t.Timestamp1)
	todStart := TicketOnDay{Plate: t.Plate, Day: d1}
	d2 := day(t.Timestamp2)
	todEnd := TicketOnDay{Plate: t.Plate, Day: d2}

	_, ok1 := s.TicketsSent[todStart]
	_, ok2 := s.TicketsSent[todEnd]

	// block if *any* day in the span is already ticketed
	if ok1 || ok2 { // covers same-day and 2-day spans
		slog.Debug("ticket overlaps used day; skipping",
			"plate", t.Plate, "d1", d1, "d2", d2)
		return
	}
	s.TicketsSent[todStart] = true
	s.TicketsSent[todEnd] = true
	s.DispatcherHandler.SendTicket(t)
}

type TicketOnDay struct {
	Plate string
	Day   uint32
}

func day(timestamp uint32) uint32 {
	// Since timestamps do not count leap seconds, days are defined by floor(timestamp / 86400).
	// TODO: Maximize revenues.
	return timestamp / 86400
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
func closeOrLog(conn *Conn) {
	slog.Debug("connection was closed", "ID", conn.ID)
	if err := conn.Close(); err != nil {
		slog.Error("error closing connection", "err", err, "remote_addr", conn.RemoteAddr())
	}
}

func illegalMessage(t uint8) *ErrorMessage {
	return &ErrorMessage{Msg: fmt.Sprintf("illegal message: %02X", t)}
}
