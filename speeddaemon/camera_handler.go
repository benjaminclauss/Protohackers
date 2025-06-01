package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

type CameraHandler struct {
	mu sync.Mutex

	// TODO: Do we need this if we are only writing heartbeats to camera?
	// For Dispatchers, we need to send tickets separate of reading (and anything else - close if received).
	connections map[uint64]*Conn
	// TODO: Move this to a separate struct. Dependency inversion principle.
	recordings map[Car][]CameraRecord

	recordsChan chan<- CameraRecord
}

func NewCameraHandler(recordsChan chan<- CameraRecord) *CameraHandler {
	return &CameraHandler{
		connections: make(map[uint64]*Conn),
		recordings:  make(map[Car][]CameraRecord),
		recordsChan: recordsChan,
	}
}

func (h *CameraHandler) handleCamera(conn *Conn) error {
	h.mu.Lock()
	h.connections[conn.ID] = conn
	h.mu.Unlock()
	defer h.disconnect(conn)

	m, err := readIAmCameraMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading IAmCamera message: %w", err)
	}

	camera := Camera{Road: m.Road, Mile: m.Mile, Limit: m.Limit}
	slog.Info("camera connected", "road", camera.Road, "mile", camera.Mile, "limit", camera.Limit)

	for {
		var t uint8
		// TODO: Should we ever disconnect client?
		if err := binary.Read(conn, binary.BigEndian, &t); err != nil {
			return fmt.Errorf("error reading message type: %w", err)
		}

		switch t {
		case PlateMessageType:
			if err := h.recordPlateMessage(camera, conn); err != nil {
				return fmt.Errorf("error recording plate message: %w", err)
			}
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

func (h *CameraHandler) disconnect(conn *Conn) {
	// TODO: Log error. Should we do this here?
	_ = conn.Close()
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.connections, conn.ID)
}

func (h *CameraHandler) recordPlateMessage(c Camera, client net.Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	message, err := readPlateMessage(client)
	if err != nil {
		return fmt.Errorf("error reading plate message: %w", err)
	}

	slog.Info("received plate message", "road", c.Road, "mile", c.Mile, "limit", c.Limit,
		"plate", message.Plate, "timestamp", message.Timestamp)

	records, ok := h.recordings[Car(message.Plate)]
	if !ok {
		records = make([]CameraRecord, 0)
	}
	// TODO: Add camera information to record.
	r := CameraRecord{Camera: c, PlateMessage: *message}
	records = append(records, r)
	h.recordings[Car(message.Plate)] = records
	h.recordsChan <- r
	return nil
}

func (h *CameraHandler) FetchPlateRecords(plate string) []CameraRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.recordings[Car(plate)]
}
