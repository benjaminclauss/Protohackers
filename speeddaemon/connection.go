package speeddaemon

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Conn struct {
	// mu protects concurrent write access.
	mu sync.Mutex

	net.Conn
	ID        uint64
	Heartbeat *Heartbeat
}

func (c *Conn) Read(p []byte) (n int, err error) {
	return c.Conn.Read(p)
}

func (c *Conn) Write(p []byte) (n int, err error) {
	return c.Conn.Write(p)
}

const Decisecond = 100 * time.Millisecond

func beginHeartbeat(conn *Conn) error {
	m, err := readWantHeartbeatMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading WantHeartbeat message: %w", err)
	}

	done := make(chan bool)
	if m.Interval == 0 {
		conn.Heartbeat = &Heartbeat{Done: done}
		return nil
	}

	ticker := time.NewTicker(time.Duration(m.Interval) * Decisecond)
	conn.Heartbeat = &Heartbeat{Ticker: ticker, Done: done}

	go heartbeat(conn)
	return nil
}

func heartbeat(conn *Conn) {
	for {
		select {
		case <-conn.Heartbeat.Done:
			return
		case t := <-conn.Heartbeat.Ticker.C:
			slog.Info("heartbeat", "time", t, "camera", conn.ID)
			m := HeartbeatMessage{}
			bytes, _ := m.MarshalBinary()
			conn.Write(bytes)
		}
	}
}

type Heartbeat struct {
	Ticker *time.Ticker
	Done   chan bool
}
