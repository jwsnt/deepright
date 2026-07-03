package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"static-server/server"
)

func main() {
	port := flag.Int("port", 8080, "listen port")
	siteDir := flag.String("site", "", "static site directory (default: ./site)")
	flag.Parse()

	site := *siteDir
	if site == "" {
		exe, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot resolve executable path: %v", err)
		}
		site = filepath.Join(filepath.Dir(exe), "site")
	}

	mux := http.NewServeMux()
	if err := server.Register(mux, site); err != nil {
		log.Fatalf("static: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("static server started on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
