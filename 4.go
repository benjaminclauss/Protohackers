package main

import (
	"log/slog"
	"net"
	"strings"
	"sync"
)

const insertRequestDelimiter = "="

type UnusualDatabaseProgram struct {
	mu   sync.Mutex
	data map[string]string
}

func (p *UnusualDatabaseProgram) Listen(conn net.PacketConn) error {
	for {
		// All requests and responses must be shorter than 1000 bytes.
		// TODO: We are allocating a new array every time!
		buf := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}
		p.handleRequest(conn, addr, buf[:n])
	}
}

func (p *UnusualDatabaseProgram) handleRequest(conn net.PacketConn, addr net.Addr, bytes []byte) error {
	request := string(bytes)
	slog.Info("received request", "request", request)
	parts := strings.SplitN(request, insertRequestDelimiter, 2)
	if len(parts) == 2 {
		p.mu.Lock()
		defer p.mu.Unlock()
		k, v := parts[0], parts[1]
		if k == "version" {
			return nil
		}
		slog.Info("insertion request", "k", k, "v", v)

		p.data[k] = v
	} else {
		p.mu.Lock()
		defer p.mu.Unlock()
		// A request that does not contain an equals sign is a retrieve request.
		k := parts[0]
		// In response to a retrieve request, the server must send back the key and its corresponding value, separated by an equals sign.
		// If a requests is for a key that has been inserted multiple times, the server must return the most recent value.
		v := p.data[k]
		if k == "version" {
			v = "Ken's Key-Value Store 1.0"
		}
		slog.Info("retrieve request", "k", k, "v", v)
		response := []byte(k + insertRequestDelimiter + v)
		// Responses must be sent to the IP address and port number that the request originated from, and must be sent
		// from the IP address and port number that the request was sent to.

		// If a request attempts to retrieve a key for which no value exists, the server can either return a response
		// as if the key had the empty value (e.g. "key="), or return no response at all.
		conn.WriteTo(response, addr)
	}
	return nil
}
