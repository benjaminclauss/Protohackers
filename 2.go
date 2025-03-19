package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"time"
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

type TimestampedPrice struct {
	Timestamp time.Time
	Price     int32
}

func meansToAnEnd(conn net.Conn) {
	defer CloseOrLog(conn)
	// TODO: Use slog.
	fmt.Println("New connection:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	buf := make([]byte, 9)
	var prices []TimestampedPrice
	for {
		n, err := io.ReadFull(reader, buf)
		if err != nil {
			log.Printf("Connection closed or error: %v", err)
			return
		}

		log.Printf("Received %d bytes", n)

		messageType := buf[0]
		// The first byte of a message is a character indicating its type.
		// This will be an ASCII uppercase 'I' or 'Q' character, indicating whether the message inserts or queries prices, respectively.
		// Read the first 4-byte integer (bytes 1-5)

		// The next 8 bytes are two signed two's complement 32-bit integers in network byte order (big endian), whose meaning depends on the message type.
		// We'll refer to these numbers as int32, but note this may differ from your system's native int32 type (if any), particularly with regard to byte order.

		var firstInt int32
		err = binary.Read(bytes.NewReader(buf[1:5]), binary.BigEndian, &firstInt)
		if err != nil {
			return
		}

		// Read the second 4-byte integer (bytes 5-9)
		var secondInt int32
		err = binary.Read(bytes.NewReader(buf[5:9]), binary.BigEndian, &secondInt)
		if err != nil {
			return
		}

		switch {
		case messageType == 'I':
			fmt.Println("Insertion")
			t := time.Unix(int64(firstInt), 0).UTC()
			fmt.Printf("Timestamp: %s\n", t)
			price := secondInt
			fmt.Printf("Price: %d\n", price)
			prices = append(prices, TimestampedPrice{Timestamp: t, Price: price})
		case messageType == 'Q':
			fmt.Println("Query")
			minTime := time.Unix(int64(firstInt), 0).UTC()
			fmt.Printf("mintime: %s\n", minTime)
			maxTime := time.Unix(int64(secondInt), 0).UTC()
			fmt.Printf("maxtime: %s\n", maxTime)

			// TODO:The server must then send the mean to the client as a single int32.
			var pricesInRange []TimestampedPrice
			mean := 0.0
			for _, p := range prices {
				if isInRange(p.Timestamp, minTime, maxTime) {
					pricesInRange = append(pricesInRange, p)
				}
			}
			if len(pricesInRange) > 0 {
				total := 0.0
				for _, p := range pricesInRange {
					total += float64(p.Price)
				}
				mean = total / float64(len(pricesInRange))
			}

			fmt.Println(prices)
			fmt.Println(mean)

			resp := new(bytes.Buffer)
			err = binary.Write(resp, binary.BigEndian, int32(math.Round(mean)))
			if err != nil {
				fmt.Println("Binary write error:", err)
				return
			}

			// Send bytes over the connection
			_, err = conn.Write(resp.Bytes())
			if err != nil {
				fmt.Println("Write error:", err)
				return
			}

		default:
			fmt.Println("Unknown message:", messageType)
			fmt.Println(buf)
			fmt.Errorf("invalid message type")
		}

	}
}

func isInRange(t, start, end time.Time) bool {
	return !t.Before(start) && !t.After(end) // Equivalent to start <= t <= end
}
