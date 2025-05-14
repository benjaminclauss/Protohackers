package main

import (
	"bufio"
	"log"
	"net"
)

func SmokeTest(conn net.Conn) {
	defer CloseOrLog(conn)

	reader := bufio.NewReader(conn)

	buf := make([]byte, 4096) // Read in 4KB chunks
	for {
		n, err := reader.Read(buf)
		if err != nil {
			log.Printf("Connection closed or error: %v", err)
			return
		}

		log.Printf("Received %d bytes", n)
		// TODO: Handle error.
		conn.Write(buf[:n]) // Echo back
	}
}
