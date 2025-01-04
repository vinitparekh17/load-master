package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	MatchStatic = 1
	MatchProxy  = 2
	NotFound    = 3
)

func (sm *ShardManager) UseRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serveStaticFile(w, r) // Default handler for root path
			return
		}

		caseNo := Name(r.URL.Path)
		fmt.Println(caseNo, r.URL.Path)

		switch caseNo {
		case MatchProxy:
			sm.handleProxyReq(r.URL.Path, w, r)
		case MatchStatic:
			serveStaticFile(w, r)
		case NotFound:
			serveErrorPage(w, r)
		}
	}))
	return mux
}

// Determine the case for a given path
func Name(path string) int {
	if path == "" {
		return NotFound
	}

	for endpoint := range SlbConfig.Locations {
		if strings.HasPrefix(path, endpoint) {
			if SlbConfig.Locations[path].Upstream == nil {
				return MatchStatic
			}
			return MatchProxy
		}
	}
	return NotFound
}

// Serve static files
func serveStaticFile(w http.ResponseWriter, r *http.Request) {
	requestedPath := filepath.Join(DefaultStaticRootPath, r.URL.Path)
	if _, err := os.Stat(requestedPath); err != nil {
		if os.IsNotExist(err) {
			serveErrorPage(w, r)
		} else {
			serveErrorPage(w, r)
		}
		return
	}
	StaticHandler(w, r)
}

// Serve an error page
func serveErrorPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, DefaultError404Page)
}
