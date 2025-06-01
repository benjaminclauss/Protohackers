package main

import (
	"context"
	"fmt"
	"github.com/benjaminclauss/protohackers/speeddaemon"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // now debug messages are shown too
	}))
	slog.SetDefault(logger)

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error { return serve(50001, Echo) })
	g.Go(func() error { return serve(50002, PrimeTime) })
	g.Go(func() error { return serve(50003, MeansToAnEnd) })

	chat := NewBudgetChat(DefaultWelcomeMessage)
	g.Go(func() error { return serve(50004, chat.Handle) })

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}

	p := &UnusualDatabaseProgram{data: make(map[string]string)}
	g.Go(func() error {
		// TODO: Inject this in deploy.
		pc, err := net.ListenPacket("udp", host+":50005")
		if err != nil {
			log.Fatal(err)
		}
		return p.Listen(pc)
	})

	g.Go(func() error { return serve(50006, MobInTheMiddle) })

	g.Go(func() error {
		http.HandleFunc("/", landingPageHandler)
		return http.ListenAndServe(":8080", nil)
	})

	server := speeddaemon.SpeedLimitEnforcementServer{
		CameraHandler:     speeddaemon.NewCameraHandler(),
		DispatcherHandler: speeddaemon.NewDispatcherHandler(),
	}
	g.Go(func() error { return serve(50007, server.Handle) })

	err := g.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func serve(port int, handler func(net.Conn) error) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	slog.Info("listening", "port", addr.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("connection error", "err", err)
			continue
		}
		slog.Debug("accepted new connection", "remote", conn.RemoteAddr())
		go func() {
			handlerErr := handler(conn)
			if handlerErr != nil {
				slog.Error("handler error", "err", handlerErr)
			}
		}()
	}
}
