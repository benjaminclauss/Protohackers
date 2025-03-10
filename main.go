package main

import (
	"context"
	"golang.org/x/sync/errgroup"
	"log"
)

func main() {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(SmokeTest)
	g.Go(PrimeTime)
	g.Go(MeansToAnEnd)
	err := g.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
