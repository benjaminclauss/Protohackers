package linereversal

import (
	"fmt"
	"strconv"
	"strings"
)

type Message interface{}

const MessageSeparator = "/"

// ParseMessage parses and validates a LRCP message.
//
// Each message consists of a series of values separated by forward slash characters ("/"), and starts and ends with a
// forward slash character, like so:
//
//	/data/1234567/0/hello/
//
// The first field is a string specifying the message type (here, "data").
// The remaining fields depend on the message type. Numeric fields are represented as ASCII text.
func ParseMessage(s string) (Message, error) {
	// Packet contents must begin with a forward slash and end with a forward slash.
	if !strings.HasPrefix(s, MessageSeparator) {
		return nil, fmt.Errorf("invalid message: must begin with forward slash")
	}
	if !strings.HasSuffix(s, MessageSeparator) {
		return nil, fmt.Errorf("invalid message: must end with forward slash")
	}

	s = strings.TrimSuffix(strings.TrimPrefix(s, MessageSeparator), MessageSeparator)

	fields := SplitEscaped(s, '/', '\\')

	messageType := fields[0]
	switch messageType {
	case "connect":
		return parseConnectMessage(fields[1:])
	case "data":
		return parseDataMessage(fields[1:])
	case "ack":
		return parseAckMessage(fields[1:])
	case "close":
		return parseCloseMessage(fields[1:])
	default:
		return nil, fmt.Errorf("invalid message: unknown message type: %s", messageType)
	}
}

func SplitEscaped(s string, sep, escape rune) []string {
	var fields []string
	var current []rune
	escaped := false

	for _, r := range s {
		if escaped {
			current = append(current, r)
			escaped = false
			continue
		}
		if r == escape {
			escaped = true
			continue
		}
		if r == sep {
			fields = append(fields, string(current))
			current = nil
			continue
		}
		current = append(current, r)
	}

	fields = append(fields, string(current))
	return fields
}

// A ConnectMessage is sent by a client, to a server, to request that a session is opened.
type ConnectMessage struct {
	Session SessionToken
}

func parseConnectMessage(fields []string) (*ConnectMessage, error) {
	if len(fields) != 1 {
		return nil, fmt.Errorf("invalid message: connect message must contain one field")
	}
	session, err := parseNumericField(fields[0])
	if err != nil {
		return nil, fmt.Errorf("error reading SESSION: %w", err)
	}
	return &ConnectMessage{Session: SessionToken(session)}, nil
}

// A DataMessage transmits payload data.
type DataMessage struct {
	Session SessionToken
	// Pos refers to the position in the stream of unescaped application-layer bytes, not the escaped data passed in LRCP.
	Pos  uint32
	Data string
}

func parseDataMessage(fields []string) (*DataMessage, error) {
	if len(fields) != 3 {
		return nil, fmt.Errorf("invalid message: data message must contain three fields")
	}

	m := &DataMessage{}
	session, err := parseNumericField(fields[0])
	if err != nil {
		return nil, fmt.Errorf("error reading SESSION: %w", err)
	}
	m.Session = SessionToken(session)

	position, err := parseNumericField(fields[1])
	if err != nil {
		return nil, fmt.Errorf("error reading POSITION: %w", err)
	}
	m.Pos = position

	m.Data = fields[2]

	return m, nil
}

// An AckMessage acknowledges receipt of payload data.
type AckMessage struct {
	Session SessionToken
	// Length tells the other side how many bytes of payload have been successfully received so far.
	Length uint32
}

func parseAckMessage(fields []string) (*AckMessage, error) {
	if len(fields) != 2 {
		return nil, fmt.Errorf("invalid message: ack message must contain two fields")
	}

	m := &AckMessage{}
	session, err := parseNumericField(fields[0])
	if err != nil {
		return nil, fmt.Errorf("error reading SESSION: %w", err)
	}
	m.Session = SessionToken(session)

	length, err := parseNumericField(fields[1])
	if err != nil {
		return nil, fmt.Errorf("error reading LENGTH: %w", err)
	}
	m.Length = length

	return m, nil
}

// A CloseMessage requests that the session is closed.
// This can be initiated by either the server or the client.
type CloseMessage struct {
	Session SessionToken
}

func parseCloseMessage(fields []string) (*CloseMessage, error) {
	if len(fields) != 1 {
		return nil, fmt.Errorf("invalid message: close message must contain one field")
	}
	session, err := parseNumericField(fields[0])
	if err != nil {
		return nil, fmt.Errorf("error reading SESSION: %w", err)
	}
	return &CloseMessage{Session: SessionToken(session)}, nil
}

// MaximumNumericFieldValue is the maximum numeric field value.
//
// Numeric field values must be smaller than 2,147,483,648.
// This means sessions are limited to 2 billion bytes of data transferred in each direction.
const MaximumNumericFieldValue = 2_147_483_648

func parseNumericField(s string) (uint32, error) {
	u, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric field value: %w", err)
	}
	if u > MaximumNumericFieldValue {
		return 0, fmt.Errorf("numeric field values must be smaller than 2147483648")
	}
	return uint32(u), nil
}
