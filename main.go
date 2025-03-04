package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	// Listen on a random available port
	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Get and print assigned port
	addr := listener.Addr().(*net.TCPAddr)
	fmt.Printf("Listening on port: %d\n", addr.Port)

	// Accept and handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
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
		conn.Write(buf[:n]) // Echo back
	}
}
