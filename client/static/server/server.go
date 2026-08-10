package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Register registers a static file handler on the given mux.
// It maps the "/site/" HTTP path to the specified directory.
func Register(mux *http.ServeMux, siteDir string) error {
	absDir, err := filepath.Abs(siteDir)
	if err != nil {
		return fmt.Errorf("cannot resolve site path: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("cannot access site directory %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absDir)
	}

	mux.Handle("/site/", http.StripPrefix("/site", http.FileServer(http.Dir(absDir))))
	log.Printf("static: serving /site/ from %s", absDir)
	return nil
}
