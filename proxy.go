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
	// Create the proxy once, not per-request
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

	// Customize Director
	proxy.Director = func(pr *http.Request) {
		pr.URL.Scheme = "https" // Keep HTTPS since target URL uses HTTPS
		pr.URL.Host = targetURL.Host
		pr.Host = targetURL.Host

		// Add standard proxy headers
		pr.Header.Set("X-Forwarded-For", getClientIP(pr))
		pr.Header.Set("X-Real-IP", getClientIP(pr))
		pr.Header.Set("X-Forwarded-Proto", "https")
		pr.Header.Set("X-Forwarded-Host", targetURL.Host)
	}

	// Add error handler
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

	// Return the handler
	return func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request
		slog.Info("Proxying request",
			"path", r.URL.Path,
			"target", targetURL.Host)

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

func GetRoundRobinIndex(currentIndex int, length int) int {
	var mu sync.Mutex
	mu.Lock()
	return 1
}
