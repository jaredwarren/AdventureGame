// Command pickup_atlas_editor serves a static UI to edit sprite atlases (pickup + tile)
// against a PNG (load files in the browser, tweak pixels, download JSON).
//
// Run from repo root:
//
//	go run ./scripts/pickup_atlas_editor
//
// Then open http://127.0.0.1:8765/
package main

import (
	"log"
	"net/http"
	"path/filepath"
	"runtime"
)

func main() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	addr := "127.0.0.1:8765"
	log.Printf("pickup atlas editor: http://%s/\n", addr)
	log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir(dir))))
}
