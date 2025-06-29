package linereversal

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

const maximumMessageSize = 1000

var ErrExceededMessageSize = errors.New("exceeded message size limit")

// A SessionToken uniquely identifies a client session.
// It is provided by a client upon connection.
// The peer for any given session is at a fixed IP address and port number.
type SessionToken uint32

type LRCPListener struct {
	mu sync.Mutex

	Sessions map[SessionToken]net.Addr
}

func (l *LRCPListener) Handle(conn net.PacketConn) error {
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			// TODO: Should we return if an error occurs reading.
			return err
		}
		// LRCP messages must be smaller than 1000 bytes.
		if n > maximumMessageSize {
			// TODO: Handle error.
			slog.Error(ErrExceededMessageSize.Error())
			return ErrExceededMessageSize
		}

		// Messages are sent in UDP packets.
		// Each UDP packet contains a single LRCP message.
		// TODO: When the server receives an illegal packet it must silently ignore the packet instead of interpreting it as LRCP.
		err = l.handleMessage(buf[:n], conn, addr)
	}
}

const (
	DefaultRetransmissionTimeout = 3 * time.Second
	DefaultSessionExpiryTimeout  = 60 * time.Second
)

// TODO: Make these parameters / configurable.
var (
	// RetransmissionTimeout is the time to wait before retransmitting a message.
	RetransmissionTimeout = DefaultRetransmissionTimeout
	// SessionExpiryTimeout is the time to wait before accepting that a peer has disappeared, in the event that no responses are being received.
	SessionExpiryTimeout = DefaultSessionExpiryTimeout
)

func (l *LRCPListener) handleMessage(buf []byte, conn net.PacketConn, addr net.Addr) error {
	// When the server receives an illegal packet, it must silently ignore the packet instead of interpreting it as LRCP.
	m, err := ParseMessage(string(buf))
	if err != nil {
		return err
	}

	fmt.Println(m)

	return nil
}

const connectionAck = "/ack/SESSION/0/"

func (l *LRCPListener) handleConnectMessage(m *ConnectMessage, conn net.PacketConn, addr net.Addr) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If no session with this token is open: open one, and associate it with the IP address and port number that the UDP packet originated from.
	_, ok := l.Sessions[m.Session]
	if !ok {
		l.Sessions[m.Session] = addr
	}

	_, err := conn.WriteTo([]byte(connectionAck), addr)
	return err
}
