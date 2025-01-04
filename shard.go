package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Shard struct {
	id              int
	shardClient     *http.Client
	proxyReqChan    chan ProxyRequest
	upstreamCounter int
	mu              sync.Mutex
}

func NewShard(id int) *Shard {
	return &Shard{
		id: id,
		shardClient: &http.Client{
			Transport: NewCustomTransport(id),
			Timeout:   20 * time.Second,
		},
		proxyReqChan: make(chan ProxyRequest),
	}
}

func (s *Shard) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(s.proxyReqChan)
			return
		case proxyReq, ok := <-s.proxyReqChan:
			if !ok {
				return
			}
			s.startProcessing(proxyReq)
		}
	}
}
func (s *Shard) startProcessing(proxy ProxyRequest) {
	upstream := SlbConfig.Locations[proxy.Endpoint].Upstream
	var originalURL string
	if len(upstream.Addr) == 1 {
		originalURL = upstream.Addr[0]
	} else {
		originalURL = upstream.Addr[s.GetNextAddrIndex(proxy.Endpoint)]
	}

	// Log the initial request details
	slog.Info("Starting proxy request",
		"shard_id", s.id,
		"method", proxy.OriginalRequest.Method,
		"path", proxy.OriginalRequest.URL.Path,
		"upstream", originalURL)

	targetURL := fmt.Sprintf("https://%s%s", originalURL, proxy.OriginalRequest.URL.Path)
	if proxy.OriginalRequest.URL.RawQuery != "" {
		targetURL += "?" + proxy.OriginalRequest.URL.RawQuery
	}

	// Check if context is already canceled
	if proxy.OriginalRequest.Context().Err() != nil {
		slog.Error("Request context already canceled before processing",
			"error", proxy.OriginalRequest.Context().Err(),
			"url", targetURL)
		http.Error(proxy.ResponseWriter, "Request Canceled", http.StatusBadGateway)
		return
	}

	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Monitor context in background
	go func() {
		select {
		case <-reqCtx.Done():
			slog.Error("Request context canceled",
				"error", reqCtx.Err(),
				"url", targetURL,
				"timeout_duration", "30s")
		case <-proxy.OriginalRequest.Context().Done():
			slog.Error("Original request context canceled",
				"error", proxy.OriginalRequest.Context().Err(),
				"url", targetURL)
		}
	}()

	upstreamReq, err := http.NewRequestWithContext(reqCtx, proxy.OriginalRequest.Method, targetURL, proxy.OriginalRequest.Body)
	if err != nil {
		slog.Error("Failed to create upstream request",
			"error", err,
			"url", targetURL)
		http.Error(proxy.ResponseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Clone and set headers
	upstreamReq.Header = proxy.OriginalRequest.Header.Clone()
	upstreamReq.Header.Set("X-Forwarded-For", getClientIP(proxy.OriginalRequest))
	upstreamReq.Header.Set("X-Real-IP", getClientIP(proxy.OriginalRequest))
	upstreamReq.Header.Set("X-Forwarded-Proto", "https")
	upstreamReq.Header.Set("X-Forwarded-Host", proxy.OriginalRequest.Host)

	// Log request start time
	startTime := time.Now()

	// Execute the upstream request
	resp, err := s.shardClient.Do(upstreamReq)
	if err != nil {
		slog.Error("Failed to process upstream request",
			"error", err,
			"url", targetURL,
			"duration", time.Since(startTime))

		// Check if it's a context cancellation
		if err == context.Canceled || err == context.DeadlineExceeded {
			http.Error(proxy.ResponseWriter, "Request Timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(proxy.ResponseWriter, "Bad Gateway", http.StatusBadGateway)
		}
		return
	}
	defer resp.Body.Close()

	slog.Info("Upstream request successful",
		"shard_id", s.id,
		"status", resp.StatusCode,
		"duration", time.Since(startTime),
		"url", targetURL)

	// Copy response headers
	copyHeaders(resp.Header, proxy.ResponseWriter.Header())
	proxy.ResponseWriter.WriteHeader(resp.StatusCode)

	// Copy body with monitoring
	written, err := io.Copy(proxy.ResponseWriter, resp.Body)
	if err != nil {
		slog.Error("Failed to copy response body",
			"error", err,
			"bytes_written", written,
			"url", targetURL)
		return
	}

	slog.Info("Request completed successfully",
		"shard_id", s.id,
		"bytes_written", written,
		"total_duration", time.Since(startTime),
		"url", targetURL)

	proxy.done.Done()
}

// Helper function to get client IP
func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Utility function to copy headers
func copyHeaders(src http.Header, dest http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

func (s *Shard) GetNextAddrIndex(endpoint string) int {
	s.mu.Lock()
	upstreamIndex := s.upstreamCounter % len(SlbConfig.Locations[endpoint].Upstream.Addr)
	s.upstreamCounter = (s.upstreamCounter + 1) % len(SlbConfig.Locations[endpoint].Upstream.Addr)
	s.mu.Unlock()
	return upstreamIndex
}
