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
	if !strings.HasPrefix(s, MessageSeparator) {
		return nil, fmt.Errorf("invalid message: must begin with forward slash")
	}
	if !strings.HasSuffix(s, MessageSeparator) {
		return nil, fmt.Errorf("invalid message: must end with forward slash")
	}
	stripped := strings.TrimSuffix(strings.TrimPrefix(s, MessageSeparator), MessageSeparator)

	parts := strings.Split(stripped, MessageSeparator)
	mType := parts[0]

	switch mType {
	case "connect":
		return parseConnectMessage(parts[1:])
	case "data":
		return parseDataMessage(parts[1:])
	default:
		return nil, fmt.Errorf("invalid message: invalid message type")
	}
}

// A ConnectMessage is sent by a client, to a server, to request that a session is opened.
type ConnectMessage struct {
	SessionToken uint32
}

func parseConnectMessage(parts []string) (Message, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("invalid message: connect message must contain one value")
	}
	session, err := parseNumericField(parts[0])
	if err != nil {
		return nil, fmt.Errorf("error reading SESSION: %w", err)
	}
	return &ConnectMessage{SessionToken: session}, nil
}

func parseDataMessage(p []string) (Message, error) {
	return nil, nil
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
