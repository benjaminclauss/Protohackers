package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
)

type DispatcherHandler struct {
	mu sync.Mutex

	connections map[uint64]*Conn
}

func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		connections: make(map[uint64]*Conn),
	}
}

func (h *DispatcherHandler) handleDispatcher(conn *Conn) error {
	h.mu.Lock()
	h.connections[conn.ID] = conn
	h.mu.Unlock()
	defer h.disconnect(conn)

	m, err := readIAmDispatcherMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading IAmDispatcher message: %w", err)
	}

	d := TicketDispatcher{Roads: m.Roads}
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

func (h *DispatcherHandler) disconnect(conn *Conn) {
	// TODO: Log error.
	_ = conn.Close()
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn.Heartbeat != nil {
		conn.Heartbeat.Ticker.Stop()
		conn.Heartbeat.Done <- true
	}
	delete(h.connections, conn.ID)
}
