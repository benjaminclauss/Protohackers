package speeddaemon

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
)

type CameraHandler struct {
	mu sync.Mutex

	// TODO: Move this to a separate struct. Dependency inversion principle.
	recordings map[Car][]CameraRecord

	recordsChan chan<- CameraRecord
}

func NewCameraHandler(recordsChan chan<- CameraRecord) *CameraHandler {
	return &CameraHandler{
		recordings:  make(map[Car][]CameraRecord),
		recordsChan: recordsChan,
	}
}

func (h *CameraHandler) handleCamera(conn *Conn) error {
	m, err := readIAmCameraMessage(conn)
	if err != nil {
		// TODO: Remove
		slog.Error("bad connection", "ID", conn.ID, "error", err, "addr", conn.RemoteAddr())
		return fmt.Errorf("error reading IAmCamera message: %w", err)
	}

	camera := Camera{Road: m.Road, Mile: m.Mile, Limit: m.Limit}
	slog.Info("camera connected", "id", conn.ID, "road", camera.Road, "mile", camera.Mile, "limit", camera.Limit)

	for {
		slog.Debug("reading from connection", "ID", conn.ID)
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
		slog.Debug("done reading from connection", "ID", conn.ID)
	}
}

func (h *CameraHandler) recordPlateMessage(c Camera, client *Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	message, err := readPlateMessage(client)
	if err != nil {
		return fmt.Errorf("error reading plate message: %w", err)
	}

	slog.Info("received plate message", "ID", client.ID, "road", c.Road, "mile", c.Mile, "limit", c.Limit,
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
	slog.Debug("wrote record, moving on...")
	return nil
}

func (h *CameraHandler) FetchPlateRecords(plate string) []CameraRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.recordings[Car(plate)]
}
