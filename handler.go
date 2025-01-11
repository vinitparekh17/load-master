package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type HandlerType int

const (
	TypeNotFound HandlerType = iota
	TypeProxy
	TypeStatic

	ROUND_ROBIN        string = "round-robin"
	LEAST_CONN         string = "least-connections"
	CONSISTENT_HASHING string = "consistent-hashing"
)

type RouteHandler struct {
	mux      *http.ServeMux
	wp       *WorkerPool
	counter  int
	handlers map[string]*LocationHandler
}

type LocationHandler struct {
	handlerType HandlerType
	path        string
}

func NewHandler(ctx context.Context) *RouteHandler {
	rh := &RouteHandler{
		mux:      http.NewServeMux(),
		wp:       NewWorkerPool(SlbConfig.ShardCount, ctx),
		counter:  0,
		handlers: make(map[string]*LocationHandler),
	}
	rh.initializeHandlers()
	return rh
}

func (rh *RouteHandler) initializeHandlers() {
	for path := range SlbConfig.Locations {
		location := SlbConfig.Locations[path]
		if location.Upstream != nil {
			rh.handlers[path] = &LocationHandler{
				handlerType: TypeProxy,
				path:        path,
			}
		} else {
			rh.handlers[path] = &LocationHandler{
				handlerType: TypeStatic,
				path:        path,
			}
		}
	}

	rh.mux.Handle("/", rh.wp.Process(http.HandlerFunc(rh.handleRequest)))
}

func (rh *RouteHandler) handleRequest(w http.ResponseWriter, r *http.Request) {
	slog.Info("Handling request",
		"path", r.URL.Path,
		"method", r.Method)

	handler := rh.findHandler(r.URL.Path)

	switch handler.handlerType {
	case TypeProxy:
		switch SlbConfig.LoadBalancingAlg {
		default:
			addrLen := len(SlbConfig.Locations[handler.path].Upstream.Addr)
			targetAddr := SlbConfig.Locations[handler.path].Upstream.Addr[0]
			if addrLen > 1 {
				rh.counter = GetRoundRobinIndex(rh.counter, addrLen)
				targetAddr = SlbConfig.Locations[handler.path].Upstream.Addr[rh.counter]
				ProxyHandler(targetAddr)(w, r)
			}
			ProxyHandler(targetAddr)(w, r)
		}
	case TypeStatic:
		serveStaticFile(w, r)
	default:
		serveErrorPage(w, r)
	}
}

func (rh *RouteHandler) findHandler(path string) *LocationHandler {
	// Check exact match first
	if handler, exists := rh.handlers[path]; exists {
		return handler
	}

	// Check prefix matches
	var longestMatch string
	for prefix := range rh.handlers {
		if strings.HasPrefix(path, prefix) {
			if len(prefix) > len(longestMatch) {
				longestMatch = prefix
			}
		}
	}

	if longestMatch != "" {
		return rh.handlers[longestMatch]
	}

	return &LocationHandler{handlerType: TypeNotFound}
}

func (rh *RouteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rh.mux.ServeHTTP(w, r)
}
