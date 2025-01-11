package main

import (
	"net/http"
	"os"
	"path/filepath"
)

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
	filePath := filepath.Join(DefaultStaticRootPath, r.URL.Path)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			err404Page := filepath.Join(DefaultStaticRootPath, DefaultError404Page)
			http.ServeFile(w, r, err404Page)
			return
		}
		err500Page := filepath.Join(DefaultStaticRootPath, DefaultError500Page)
		http.ServeFile(w, r, err500Page)
		return
	}

	http.FileServer(http.Dir("static")).ServeHTTP(w, r)
}

// Serve an error page
func serveErrorPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, DefaultError404Page)
}
