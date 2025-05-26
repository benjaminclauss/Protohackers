package speeddaemon

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type MessageType uint8

const (
	ErrorMessageType         uint8 = 0x10
	PlateMessageType               = 0x20
	TicketMessageType              = 0x21
	WantHeartbeatMessageType       = 0x40
	HeartbeatMessageType           = 0x41
	IAmCameraMessageType           = 0x80
	IAmDispatcherMessageType       = 0x81
)

type ErrorMessage struct {
	Msg string
}

func (m *ErrorMessage) MarshalBinary() ([]byte, error) {
	data := []byte{ErrorMessageType}

	msg, err := marshalStr(m.Msg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling message: %w", err)
	}
	data = append(data, msg...)

	return data, nil
}

type PlateMessage struct {
	Plate     string
	Timestamp uint32
}

func readPlateMessage(r io.Reader) (*PlateMessage, error) {
	buf := bufio.NewReader(r)

	lengthByte, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("error reading plate length: %w", err)
	}

	plateBytes := make([]byte, lengthByte)
	if _, err := io.ReadFull(buf, plateBytes); err != nil {
		return nil, fmt.Errorf("error reading plate: %w", err)
	}

	var ts uint32
	if err := binary.Read(buf, binary.BigEndian, &ts); err != nil {
		return nil, fmt.Errorf("error reading timestamp: %w", err)
	}

	return &PlateMessage{Plate: string(plateBytes), Timestamp: ts}, nil
}

type TicketMessage struct {
	// Plate is the number plate of the offending car.
	Plate string
	// Road is the road number of the cameras.
	Road uint16
	// Mile1 is the position of the first camera.
	Mile1 uint16
	// Timestamp1 is the timestamp of the first camera observation.
	Timestamp1 uint32
	// Mile2 is the position of the second camera.
	Mile2 uint16
	// Timestamp2 is the timestamp of the second camera observation.
	Timestamp2 uint32
	// Speed is the inferred average speed of the car multiplied by 100.
	Speed uint16
}

func (m *TicketMessage) MarshalBinary() ([]byte, error) {
	data := []byte{TicketMessageType}

	plate, err := marshalStr(m.Plate)
	if err != nil {
		return nil, fmt.Errorf("error marshaling plate: %w", err)
	}
	data = append(data, plate...)

	fields := []any{
		m.Road,
		m.Mile1,
		m.Timestamp1,
		m.Mile2,
		m.Timestamp2,
		m.Speed,
	}
	for _, field := range fields {
		buf := make([]byte, 4)
		n := 0
		switch v := field.(type) {
		case uint16:
			binary.BigEndian.PutUint16(buf, v)
			n = 2
		case uint32:
			binary.BigEndian.PutUint32(buf, v)
			n = 4
		}
		data = append(data, buf[:n]...)
	}

	return data, nil
}

func marshalStr(s string) ([]byte, error) {
	if len(s) > 255 {
		return nil, fmt.Errorf("string too long: %s", s)
	}
	for _, r := range s {
		if r > 127 {
			return nil, fmt.Errorf("non-ASCII character: %q", r)
		}
	}
	data := append([]byte{uint8(len(s))}, []byte(s)...)
	return data, nil
}

type WantHeartbeatMessage struct {
	// Interval is the interval in deciseconds for which the server should send a heartbeat.
	// An interval of 0 deciseconds means the client does not want to receive heartbeats (this is the default setting).
	Interval uint32
}

func readWantHeartbeatMessage(r io.Reader) (*WantHeartbeatMessage, error) {
	var interval uint32
	if err := binary.Read(r, binary.BigEndian, &interval); err != nil {
		return nil, fmt.Errorf("error reading heartbeat interval: %w", err)
	}
	return &WantHeartbeatMessage{Interval: interval}, nil
}

type HeartbeatMessage struct{}

func (m *HeartbeatMessage) MarshalBinary() ([]byte, error) {
	return []byte{HeartbeatMessageType}, nil
}

type IAmCameraMessage struct {
	//The road field contains the road number that the camera is on.
	Road uint16
	// Mile contains the position of the camera, relative to the start of the road.
	Mile uint16
	// Limit contains the speed limit of the road, in miles per hour.
	Limit uint16
}

func readIAmCameraMessage(r io.Reader) (*IAmCameraMessage, error) {
	var road, mile, limit uint16
	if err := binary.Read(r, binary.BigEndian, &road); err != nil {
		return nil, fmt.Errorf("error reading road: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &mile); err != nil {
		return nil, fmt.Errorf("error reading mile: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &limit); err != nil {
		return nil, fmt.Errorf("error reading limit: %w", err)
	}
	return &IAmCameraMessage{Road: road, Mile: mile, Limit: limit}, nil
}

type IAmDispatcherMessage struct {
	// Roads contains the road numbers.
	Roads []uint16
}

func readIAmDispatcherMessage(r io.Reader) (*IAmDispatcherMessage, error) {
	buf := bufio.NewReader(r)

	numRoadsByte, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("error reading numroads: %w", err)
	}

	roads := make([]uint16, numRoadsByte)
	for i := range roads {
		if err := binary.Read(buf, binary.BigEndian, &roads[i]); err != nil {
			return nil, fmt.Errorf("error reading road[%d]: %w", i, err)
		}
	}
	return &IAmDispatcherMessage{Roads: roads}, nil
}
