package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
)

type DispatcherHandler struct {
	mu sync.Mutex

	connections       map[uint64]*Conn
	roadToDispatchers map[uint16][]*Conn
}

func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		connections:       make(map[uint64]*Conn),
		roadToDispatchers: make(map[uint16][]*Conn),
	}
}

func (h *DispatcherHandler) handleDispatcher(conn *Conn) error {
	h.mu.Lock()
	h.connections[conn.ID] = conn
	h.mu.Unlock()

	m, err := readIAmDispatcherMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading IAmDispatcher message: %w", err)
	}

	d := TicketDispatcher{Roads: m.Roads}
	// TODO: This is messy!
	defer h.disconnect(conn, d)
	h.registerForRoad(d, conn)
	slog.Info("dispatcher connected", "roads", d.Roads)

	for {
		var t uint8
		// TODO: Should we ever disconnect client?
		if err := binary.Read(conn, binary.BigEndian, &t); err != nil {
			return fmt.Errorf("error reading message type: %w", err)
		}

		switch t {
		case PlateMessageType:
			return sendError(conn, illegalMessage(t))
		case WantHeartbeatMessageType:
			// It is an error for a client to send multiple WantHeartbeat messages on a single connection.
			if conn.Heartbeat != nil {
				return sendError(conn, MultipleWantHeartbeatMessagesError)
			}
			if err := beginHeartbeat(conn); err != nil {
				return fmt.Errorf("error beginning heartbeat: %w", err)
			}
		case IAmCameraMessageType:
			return sendError(conn, AlreadyIdentifiedError)
		case IAmDispatcherMessageType:
			return sendError(conn, AlreadyIdentifiedError)
		default:
			return sendError(conn, illegalMessage(t))
		}
	}
}

func (h *DispatcherHandler) registerForRoad(d TicketDispatcher, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, r := range d.Roads {
		dispatchers, ok := h.roadToDispatchers[r]
		if !ok {
			dispatchers = make([]*Conn, 0)
		}
		dispatchers = append(dispatchers, conn)
		h.roadToDispatchers[r] = dispatchers
	}
}

func (h *DispatcherHandler) disconnect(conn *Conn, d TicketDispatcher) {
	// TODO: Log error. Should we do this here?
	_ = conn.Close()
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.connections, conn.ID)
	for _, r := range d.Roads {
		dispatchers, ok := h.roadToDispatchers[r]
		if !ok {
			continue
		}
		for i, c := range dispatchers {
			if c == conn {
				dispatchers = append(dispatchers[:i], dispatchers[i+1:]...)
				break
			}
		}
	}
}

func (h *DispatcherHandler) SendTicket(r CameraRecord, other CameraRecord, mph float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	dispatchers := h.roadToDispatchers[r.Road]
	if len(dispatchers) == 0 {
		// TODO: Handle
		fmt.Println("no dispatchers")
		return
	}

	dispatcher := dispatchers[0]
	fmt.Println(dispatcher.ID)

	var earlier, later CameraRecord
	// mile1 and timestamp1 must refer to the earlier of the 2 observations (the smaller timestamp), and mile2 and timestamp2 must refer to the later of the 2 observations (the larger timestam
	if r.Timestamp < other.Timestamp {
		earlier = r
		later = other
	} else {
		earlier = other
		later = r
	}

	t := TicketMessage{
		Plate:      r.Plate,
		Road:       r.Road,
		Mile1:      earlier.Camera.Mile,
		Timestamp1: earlier.Timestamp,
		Mile2:      later.Camera.Mile,
		Timestamp2: later.Timestamp,
		Speed:      uint16(mph),
	}
	fmt.Println(t)
	// TODO: Handle error.
	marshalBinary, _ := t.MarshalBinary()

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	dispatcher.Conn.Write(marshalBinary)
}
