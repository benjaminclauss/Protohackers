package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func MeansToAnEnd() error {
	listener, err := net.Listen("tcp", ":50003")
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	fmt.Printf("Listening on port: %d\n", addr.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go meansToAnEnd(conn)
	}
}

const messageSize = 9

func meansToAnEnd(conn net.Conn) {
	// TODO: Handle closure.
	defer conn.Close()
	// TODO: Use slog.
	fmt.Println("New connection:", conn.RemoteAddr())

	reader := bufio.NewReaderSize(conn, messageSize)

	buf := make([]byte, 9)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			log.Printf("Connection closed or error: %v", err)
			return
		}

		log.Printf("Received %d bytes", n)

		messageType := buf[0]
		fmt.Println(messageType)

		var result int32
		buf := bytes.NewReader(buf[1:5])
		err = binary.Read(buf, binary.LittleEndian, &result)
		if err != nil {
			fmt.Println("Error:", err)
		}

		fmt.Println(result) // Output: 1
	}
}
