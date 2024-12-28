package main

import (
	"log/slog"
	"net/http"
	"strings"
)

type ProxyHandler struct {
	sm *ShardManager
}

func NewProxyHandler(sm *ShardManager) *ProxyHandler {
	return &ProxyHandler{
		sm: sm,
	}
}

// Handle is the entry point of the load balancer which transfers client req to ShardManager
func (h *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {

	upstream := h.findMatchingUpstream(r.URL.Path)
	if upstream == nil {
		http.ServeFile(w, r, SlbConfig.ErrorFile)
		slog.Error("upstream addr not available at ", slog.String("path", r.URL.Path))
		return
	}

	h.sm.globalProxyReqChan <- &ProxyRequest{
		OriginalRequest: r,
		ResponseWriter:  w,
		UpstreamAddrs:   upstream.Addr,
	}
}

func (h *ProxyHandler) findMatchingUpstream(path string) *Upstream {
	var matchedUpstream *Upstream
	longestMatch := -1

	for i, upstream := range *SlbConfig.Upstreams {
		if strings.HasPrefix(path, upstream.Path) {
			pathLen := len(upstream.Path)
			if pathLen > longestMatch {
				longestMatch = pathLen
				matchedUpstream = &(*SlbConfig.Upstreams)[i]
			}
		}
	}

	return matchedUpstream
}
