package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	SLB_PORT = 80 // simple load balancer's port
)

type LbServer struct {
	httpServer   *http.Server
	mux          *http.ServeMux
	shutdownChan chan struct{}
}

func NewLb(ctx context.Context) *LbServer {
	return &LbServer{
		httpServer: &http.Server{
			Addr:         SlbConfig.Server.Addr,
			ReadTimeout:  SlbConfig.Server.ReadTimeout,
			WriteTimeout: SlbConfig.Server.WriteTimeout,
			IdleTimeout:  SlbConfig.Server.IdleTimeout,
		},
		mux:          http.NewServeMux(),
		shutdownChan: make(chan struct{}),
	}
}

func (lb *LbServer) Run(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		lb.Serve()
		lb.httpServer.Handler = lb.mux
		errChan <- lb.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := lb.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errChan:
		return err
	}
}

func (lb *LbServer) Serve() {
	for _, location := range SlbConfig.Locations {
		lb.mux.Handle(location.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filePath := filepath.Join(location.Root, r.URL.Path)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				errPage := filepath.Join(location.Root, location.ErrorFile)
				http.ServeFile(w, r, errPage)
				return
			}

			http.ServeFile(w, r, location.Root)
		}))
	}
}
