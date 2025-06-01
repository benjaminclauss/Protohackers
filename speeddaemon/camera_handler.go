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

type CameraConnection struct {
	id        uint64
	conn      net.Conn
	heartbeat *Heartbeat
}

type Heartbeat struct {
	Ticker *time.Ticker
	Done   chan bool
}

type CameraHandler struct {
	mu       sync.Mutex
	cameraID atomic.Uint64

	// TODO: Do we need this if we are only writing heartbeats to camera?
	// For Dispatchers, we need to send tickets separate of reading (and anything else - close if received).
	connections map[uint64]*CameraConnection
	// TODO: Move this to a separate struct. Dependency inversion principle.
	recordings map[Car][]CameraRecord
}

func NewCameraHandler() *CameraHandler {
	return &CameraHandler{
		connections: make(map[uint64]*CameraConnection),
		recordings:  make(map[Car][]CameraRecord),
	}
}

func (h *CameraHandler) handleCamera(client net.Conn) error {
	m, err := readIAmCameraMessage(client)
	if err != nil {
		return fmt.Errorf("error reading IAmCamera message: %w", err)
	}

	camera := Camera{Road: m.Road, Mile: m.Mile, Limit: m.Limit}
	slog.Info("camera connected", "road", camera.Road, "mile", camera.Mile, "limit", camera.Limit)

	h.mu.Lock()
	id := h.cameraID.Add(1)
	conn := &CameraConnection{id: id, conn: client}
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
			if err := h.recordPlateMessage(camera, client); err != nil {
				return fmt.Errorf("error recording plate message: %w", err)
			}
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

func (h *CameraHandler) disconnect(conn *CameraConnection) {
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

func (h *CameraHandler) recordPlateMessage(_ Camera, client net.Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	message, err := readPlateMessage(client)
	if err != nil {
		return fmt.Errorf("error reading plate message: %w", err)
	}

	slog.Info("received plate message", "plate", message.Plate, "timestamp", message.Timestamp)

	records, ok := h.recordings[Car(message.Plate)]
	if !ok {
		records = make([]CameraRecord, 0)
	}

	// TODO: Add camera information to record.
	records = append(records, CameraRecord{})
	h.recordings[Car(message.Plate)] = records
	return nil
}

const Decisecond = 100 * time.Millisecond

func (h *CameraHandler) beginHeartbeat(conn *CameraConnection) error {
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

	go heartbeatCamera(conn)
	return nil
}

func heartbeatCamera(conn *CameraConnection) {
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
