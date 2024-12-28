package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Shard struct {
	id              int
	shardClient     *http.Client
	proxyReqChan    chan *ProxyRequest
	upstreamCounter int
	mu              sync.Mutex
	done            chan struct{}
	errChan         chan error
}

func NewShard(id int) *Shard {
	return &Shard{
		id:           id,
		shardClient:  &http.Client{},
		proxyReqChan: make(chan *ProxyRequest),
		done:         make(chan struct{}),
		errChan:      make(chan error),
	}
}

func (s *Shard) Run(shardCtx context.Context) {
	for {
		select {
		case <-s.done:
			slog.Info("shutting down shard ", slog.Int("ShardID", s.id))

		case proxyReq, ok := <-s.proxyReqChan:
			if !ok {
				return
			}

			proxyReqCtx, cancel := context.WithTimeout(shardCtx, 30*time.Second)

			if err := s.startProcessing(proxyReqCtx); err != nil {
				proxyReq.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
				proxyReq.ResponseWriter.WriteHeader(http.StatusInternalServerError)
				http.ServeFile(proxyReq.ResponseWriter, proxyReq.OriginalRequest, "static/error.html")
			}
			cancel()
		}
	}
}

func (s *Shard) startProcessing(reqCtx context.Context) error {
	for proxy := range s.proxyReqChan {
		slog.Debug(proxy.OriginalRequest.URL.Path)

		originalURL, err := url.Parse((*SlbConfig.Upstreams)[1].Addr[s.GetNextUpstreamIndex()])
		if err != nil {
			return err
		}

		r, err := http.NewRequestWithContext(reqCtx, proxy.OriginalRequest.Method, fmt.Sprintf("%s%s", originalURL,
			proxy.OriginalRequest.URL.Path),
			proxy.OriginalRequest.Body)
		if err != nil {
			return err
		}

		defer r.Body.Close()

		r.Header = proxy.OriginalRequest.Header
		_, err = io.Copy(proxy.ResponseWriter, r.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Shard) GetNextUpstreamIndex() int {
	s.mu.Lock()
	upstreamIndex := s.upstreamCounter % len(*SlbConfig.Upstreams)
	s.upstreamCounter = (s.upstreamCounter + 1) % len(*SlbConfig.Upstreams)
	s.mu.Unlock()
	return upstreamIndex
}
