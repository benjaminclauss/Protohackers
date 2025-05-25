package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
)

const (
	UpstreamServerAddress = "chat.protohackers.com:16963"
	TonyBoguscoinAddress  = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
)

func MobInTheMiddle(conn net.Conn) error {
	defer CloseOrLog(conn)

	// For each client that connects to proxy server, make a corresponding outward connection to the upstream server.
	upstreamConn, err := net.Dial("tcp", UpstreamServerAddress)
	if err != nil {
		slog.Error("failed to connect to upstream server", "error", err)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	wg := sync.WaitGroup{}

	// When the client sends a message to your proxy, pass it on upstream.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		relayAndOverwrite(conn, upstreamConn)
	}()

	// When the upstream server sends a message to your proxy, pass it on downstream.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		relayAndOverwrite(upstreamConn, conn)
	}()

	<-ctx.Done()
	conn.Close()
	upstreamConn.Close()
	wg.Wait()
	// TODO: Handle above.
	return nil
}

func relayAndOverwrite(source, destination net.Conn) error {
	reader := bufio.NewReader(source)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		msg := rewriteAllBoguscoinAddresses(line)
		if _, err := destination.Write([]byte(msg)); err != nil {
			LogWriteError(err)
			return err
		}
	}
	return nil
}

func rewriteAllBoguscoinAddresses(msg string) string {
	for _, f := range strings.Fields(msg) {
		if strings.HasPrefix(f, "7") && len(f) >= 26 && len(f) <= 35 {
			slog.Debug("identified Boguscoin address", "address", f)
			msg = strings.ReplaceAll(msg, f, TonyBoguscoinAddress)
		}
	}
	return msg
}
