package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

// Echo implements the TCP Echo Service from [RFC 862].
//
// [RFC 862]: https://www.rfc-editor.org/rfc/rfc862.html
func Echo(conn net.Conn) error {
	// Accept TCP connections.
	defer CloseOrLog(conn)

	reader := bufio.NewReader(conn)

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		// Whenever data is received from a client, send it back unmodified.
		if _, err = conn.Write(buf[:n]); err != nil {
			return fmt.Errorf("write error: %w", err)
		}
	}
}
