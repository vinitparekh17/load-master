package main

import (
	"net/http"
	"sync"
	"time"
)

type ProxyRequest struct {
	OriginalRequest *http.Request
	ResponseWriter  http.ResponseWriter
	Endpoint        string
	done            *sync.WaitGroup
}

func (sm *ShardManager) handleProxyReq(endpoint string, w http.ResponseWriter, r *http.Request) {
	sm.wg.Add(1)

	proxyReq := ProxyRequest{
		OriginalRequest: r,
		ResponseWriter:  w,
		Endpoint:        endpoint,
		done:            &sm.wg,
	}

	select {
	case sm.globalProxyReqChan <- proxyReq:
		// Successfully sent to the channel
	case <-r.Context().Done():
		// Client canceled the request, exit early
		http.Error(w, "Request canceled", http.StatusRequestTimeout)
		sm.wg.Done()
		return
	case <-time.After(5 * time.Second):
		// Optional timeout
		http.Error(w, "Gateway timeout", http.StatusGatewayTimeout)
		sm.wg.Done()
		return
	}

	// Wait for the processing to complete, but check for context cancellation
	select {
	case <-r.Context().Done():
		http.Error(w, "Request canceled", http.StatusRequestTimeout)
		return
	case <-time.After(5 * time.Second): // optional timeout
		sm.wg.Wait() // Wait for all work to finish
	}
}
