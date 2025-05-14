package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"golang.org/x/sync/errgroup"
)

func main() {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return serve(50001, SmokeTest) })
	g.Go(func() error { return serve(50002, PrimeTime) })
	g.Go(func() error { return serve(50003, MeansToAnEnd) })
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
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handler(conn)
	}
}
