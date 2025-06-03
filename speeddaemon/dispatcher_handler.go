package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

type DispatcherHandler struct {
	mu sync.Mutex

	connections       map[uint64]*Conn
	roadToDispatchers map[uint16][]*Conn
	ticketQueue       []TicketMessage
}

func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		connections:       make(map[uint64]*Conn),
		roadToDispatchers: make(map[uint16][]*Conn),
		ticketQueue:       make([]TicketMessage, 0),
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
	h.sendQueuedTickets(d, conn)
	slog.Info("dispatcher connected", "ID", conn.ID, "roads", d.Roads)

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
	//_ = conn.Close()
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

func (h *DispatcherHandler) SendTicket(t TicketMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	slog.Debug("sending ticket", "plate", t.Plate, "ticket", t)

	dispatchers := h.roadToDispatchers[t.Road]
	if len(dispatchers) == 0 {
		h.ticketQueue = append(h.ticketQueue, t)
		slog.Debug("no dispatchers for road, queueing ticket", "road", t.Road, "plate", t.Plate, "speed", t.Speed)
		return
	}
	dispatcher := dispatchers[0]
	fmt.Println(dispatcher.ID)

	// TODO: Handle error.
	marshalBinary, _ := t.MarshalBinary()

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	dispatcher.Conn.Write(marshalBinary)
}

func (h *DispatcherHandler) sendQueuedTickets(d TicketDispatcher, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	slog.Debug("checking queued tickets")

	var remainInQueue, toSend []TicketMessage
	for _, t := range h.ticketQueue {
		if slices.Contains(d.Roads, t.Road) {
			toSend = append(toSend, t)
		} else {
			remainInQueue = append(remainInQueue, t)
		}
	}
	h.ticketQueue = remainInQueue

	if len(toSend) == 0 {
		return
	}
	slog.Debug("sending queued tickets", "count", len(toSend))

	conn.mu.Lock()
	defer conn.mu.Unlock()
	for _, m := range toSend {
		marshalBinary, _ := m.MarshalBinary()
		// TODO: Handle error above and below.
		conn.Write(marshalBinary)
	}
}
