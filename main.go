package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

func main() {
	asciiArt := `
    _                     _ __  __           _            
   | |    ___   __ _   __| |  \/  | __ _ ___| |_ ___ _ __ 
   | |   / _ \ / _` + "`" + ` | / _` + "`" + ` | |\/| |/ _` + "`" + ` / __| __/ _ \ '__|
   | |__| (_) | (_| || (_| | |  | | (_| \__ \ ||  __/ |   
   |_____\___/ \__,_| \__,_|_|  |_|\__,_|___/\__\___|_|  
	 `
	fmt.Println(asciiArt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	if SlbConfig.ShardCount <= runtime.NumCPU() {
		runtime.GOMAXPROCS(SlbConfig.ShardCount)
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := new(sync.WaitGroup)
	lb := NewLb(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := lb.Run(ctx); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err.Error())
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sig
		slog.Info("Shutting down gracefully...")
		cancel()
	}()

	wg.Wait()
	slog.Info("Server exited")
}
