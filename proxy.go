package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

func ProxyHandler(target string) http.HandlerFunc {
	targetURL, err := url.Parse(fmt.Sprintf("https://%s", target))
	if err != nil {
		slog.Error("Failed to parse target URL", "error", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid backend configuration", http.StatusInternalServerError)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Configure custom transport with timeouts
	proxy.Transport = NewTransport()

	proxy.Director = func(pr *http.Request) {
		pr.URL.Scheme = "https" // TODO: make it dynamic depands on target scheme
		pr.URL.Host = targetURL.Host
		pr.Host = targetURL.Host

		pr.Header.Set("X-Forwarded-For", getClientIP(pr))
		pr.Header.Set("X-Forwarded-Proto", "https")
		pr.Header.Set("X-Forwarded-Host", targetURL.Host)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Proxy error",
			"error", err,
			"path", r.URL.Path,
			"host", targetURL.Host)

		if err == context.Canceled {
			http.Error(w, "Request canceled", http.StatusBadGateway)
			return
		}
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

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

type RoundRobin struct {
	mu           sync.Mutex
	currentIndex int
	length       int
}

func NewRoundRobin(length int) *RoundRobin {
	if length <= 0 {
		panic("length must be greater than 0")
	}
	return &RoundRobin{length: length}
}

func (rr *RoundRobin) GetIndex() int {
	rr.mu.Lock()
	index := rr.currentIndex % rr.length
	rr.currentIndex = (rr.currentIndex + 1) % rr.length
	rr.mu.Unlock()
	return index
}
