package main

import (
	"net/http"
	"os"
	"path/filepath"
)

func StaticHandler(w http.ResponseWriter, r *http.Request) {
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
