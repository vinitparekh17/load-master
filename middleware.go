package main

import (
	"net/http"
	"path/filepath"
)

func CheckInvalidEndpoint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested path matches any valid endpoint
		for endpoint := range SlbConfig.Locations {
			if r.URL.Path == endpoint {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Serve 404 error page if no endpoint matches
		err404Page := filepath.Join(DefaultStaticRootPath, DefaultError404Page)
		http.ServeFile(w, r, err404Page)
	})
}
