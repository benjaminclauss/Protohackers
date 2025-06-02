package speeddaemon

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Heartbeat struct {
	Ticker *time.Ticker
	Done   chan bool
}

// A Conn represents a unique client connection to the server.
type Conn struct {
	// mu protects concurrent write access to the underlying net.Conn.
	mu sync.Mutex

	ID uint64
	net.Conn
	Heartbeat *Heartbeat
}

func (c *Conn) Close() error {
	slog.Debug("closing connection", "id", c.ID)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Heartbeat != nil {
		// A client may have registered a "zero" heartbeat interval, in which case we don't need to stop the ticker.
		if c.Heartbeat.Ticker != nil {
			c.Heartbeat.Ticker.Stop()
			c.Heartbeat.Done <- true
		}
	}

	return c.Conn.Close()
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
			slog.Info("heartbeat done", "connection", conn.ID)
			return
		case t := <-conn.Heartbeat.Ticker.C:
			slog.Info("heartbeat", "time", t, "connection", conn.ID)
			m := HeartbeatMessage{}
			bytes, _ := m.MarshalBinary()
			if _, err := conn.Write(bytes); err != nil {
				slog.Warn("error writing heartbeat", "err", err)
			}
		}
	}
}
