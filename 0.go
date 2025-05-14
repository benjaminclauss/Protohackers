package main

import (
	"bufio"
	"net"
)

func SmokeTest(conn net.Conn) {
	defer CloseOrLog(conn)

	reader := bufio.NewReader(conn)

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			LogReadError(err)
			return
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			LogWriteError(err)
			return
		}
	}
}
