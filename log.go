package main

import (
	"log/slog"
	"net"
)

func CloseOrLog(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("error closing connection", "err", err)
	}
}
