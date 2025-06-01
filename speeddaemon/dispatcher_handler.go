package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type DispatcherConnection struct {
	// mu protects concurrent write access.
	mu sync.Mutex

	id        uint64
	conn      net.Conn
	heartbeat *Heartbeat
}

type DispatcherHandler struct {
	mu           sync.Mutex
	dispatcherID atomic.Uint64

	connections map[uint64]*DispatcherConnection
}

func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		connections: make(map[uint64]*DispatcherConnection),
	}
}

func (h *DispatcherHandler) handleDispatcher(client net.Conn) error {
	m, err := readIAmDispatcherMessage(client)
	if err != nil {
		return fmt.Errorf("error reading IAmDispatcher message: %w", err)
	}

	d := TicketDispatcher{Roads: m.Roads}
	slog.Info("dispatcher connected", "roads", d.Roads)

	h.mu.Lock()
	id := h.dispatcherID.Add(1)
	conn := &DispatcherConnection{id: id, conn: client}
	h.connections[id] = conn
	h.mu.Unlock()
	defer h.disconnect(conn)

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
			// It is an error for a client to send multiple WantHeartbeat messages on a single connection.
			if conn.heartbeat != nil {
				return sendError(client, MultipleWantHeartbeatMessagesError)
			}
			if err := h.beginHeartbeat(conn); err != nil {
				return fmt.Errorf("error beginning heartbeat: %w", err)
			}
		case IAmCameraMessageType:
			return sendError(client, AlreadyIdentifiedError)
		case IAmDispatcherMessageType:
			return sendError(client, AlreadyIdentifiedError)
		default:
			return sendError(client, IllegalMessageType)
		}
	}
}

func (h *DispatcherHandler) disconnect(conn *DispatcherConnection) {
	// TODO: Log error.
	_ = conn.conn.Close()
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn.heartbeat != nil {
		conn.heartbeat.Ticker.Stop()
		conn.heartbeat.Done <- true
	}
	delete(h.connections, conn.id)
}

func (h *DispatcherHandler) beginHeartbeat(conn *DispatcherConnection) error {
	m, err := readWantHeartbeatMessage(conn.conn)
	if err != nil {
		return fmt.Errorf("error reading WantHeartbeat message: %w", err)
	}

	done := make(chan bool)
	if m.Interval == 0 {
		conn.heartbeat = &Heartbeat{Done: done}
		return nil
	}

	ticker := time.NewTicker(time.Duration(m.Interval) * Decisecond)
	conn.heartbeat = &Heartbeat{Ticker: ticker, Done: done}

	go heartbeatDispatcher(conn)
	return nil
}

func heartbeatDispatcher(conn *DispatcherConnection) {
	for {
		select {
		case <-conn.heartbeat.Done:
			return
		case t := <-conn.heartbeat.Ticker.C:
			slog.Info("heartbeat", "time", t, "camera", conn.id)
			m := HeartbeatMessage{}
			bytes, _ := m.MarshalBinary()
			conn.conn.Write(bytes)
		}
	}
}
