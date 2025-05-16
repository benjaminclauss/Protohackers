package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"golang.org/x/sync/errgroup"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return serve(50001, SmokeTest) })
	g.Go(func() error { return serve(50002, PrimeTime) })
	g.Go(func() error { return serve(50003, MeansToAnEnd) })

	chat := NewBudgetChat(DefaultWelcomeMessage)
	g.Go(func() error { return serve(50004, chat.Handle) })

	p := &UnusualDatabaseProgram{data: make(map[string]string)}
	g.Go(func() error {
		// TODO: Inject this in deploy.
		pc, err := net.ListenPacket("udp", "fly-global-services:50005")
		if err != nil {
			log.Fatal(err)
		}
		return p.Listen(pc)
	})

	err := g.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func serve(port int, handler func(net.Conn)) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	fmt.Printf("Listening on port: %d\n", addr.Port)

	for {
		conn, err := listener.Accept()
		fmt.Println("New connection:", conn.RemoteAddr())
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		fmt.Println("handling that")
		go handler(conn)
	}
}
