package main

import (
	"log/slog"
	"net"
)

func CloseOrLog(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("error closing connection", "err", err, "remote_addr", conn.RemoteAddr())
	}
}

func LogReadError(err error) {
	if err != nil {
		slog.Error("read error", "err", err)
	}
}

func LogWriteError(err error) {
	if err != nil {
		slog.Error("write error", "err", err)
	}
}
