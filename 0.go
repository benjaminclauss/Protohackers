package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func SmokeTest() error {
	// TODO: Propagate context when listening.
	listener, err := net.Listen("tcp", ":50001")
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	fmt.Printf("Listening on port: %d\n", addr.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			// TODO: Use slog.
			fmt.Println("Connection error:", err)
			continue
		}
		go echo(conn)
	}
}

func echo(conn net.Conn) {
	defer CloseOrLog(conn)
	fmt.Println("New connection:", conn.RemoteAddr())

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
