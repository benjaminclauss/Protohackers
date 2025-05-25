package main

import (
	"bufio"
	"io"
	"net"
)

func SmokeTest(conn net.Conn) error {
	defer CloseOrLog(conn)

	reader := bufio.NewReader(conn)

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		if _, err = conn.Write(buf[:n]); err != nil {
			return err
		}
	}
}
