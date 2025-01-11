package main

import (
	"context"
	"net/http"
	"time"
)

type LbServer struct {
	httpServer   *http.Server
	errChan      chan error
	shutdownChan chan struct{}
}

func NewLb(ctx context.Context) *LbServer {
	return &LbServer{
		httpServer: &http.Server{
			Addr:           SlbConfig.Server.Listen,
			ReadTimeout:    SlbConfig.Server.ReadTimeout,
			WriteTimeout:   SlbConfig.Server.WriteTimeout,
			IdleTimeout:    SlbConfig.Server.IdleTimeout,
			MaxHeaderBytes: 1 << 20, // 1 MB
			Handler:        NewHandler(ctx),
		},
		errChan:      make(chan error),
		shutdownChan: make(chan struct{}),
	}
}
func (lb *LbServer) Run(ctx context.Context) error {

	go func() {
		lb.errChan <- lb.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := lb.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-lb.errChan:
		return err
	}
}
