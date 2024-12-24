package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	wg := &sync.WaitGroup{}
	lb := NewLb(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := lb.Run(ctx); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	go func() {
		<-sig
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	wg.Wait()
	log.Println("Server exited")
}
