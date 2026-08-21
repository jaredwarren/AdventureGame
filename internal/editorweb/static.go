package editorweb

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

// The browser app is embedded so `go run ./cmd/mapeditor` works from anywhere
// with no build step and no node_modules. The files are plain ES modules served
// over HTTP, which gives real imports without a bundler.
//
//go:embed all:static
var staticFS embed.FS

// indexTemplate injects the per-launch token into the page. The app keeps it in
// memory and sends it as X-Editor-Token on every API call, so the token never
// has to sit in a query string after the first load.
var indexTemplate = template.Must(template.New("index").Parse(mustReadStatic("static/index.html")))
var tilesTemplate = template.Must(template.New("tiles").Parse(mustReadStatic("static/tiles.html")))

func mustReadStatic(name string) string {
	b, err := staticFS.ReadFile(name)
	if err != nil {
		panic("editorweb: missing embedded asset " + name + ": " + err.Error())
	}
	return string(b)
}

func (s *Server) indexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET / is registered as a catch-all pattern, so anything unmatched
		// lands here. Do not serve the app for those.
		if r.URL.Path != "/" {
			writeErr(w, http.StatusNotFound, "not_found", "no such path")
			return
		}
		var buf bytes.Buffer
		if err := indexTemplate.Execute(&buf, map[string]any{
			"Token": s.opts.Token,
			"Root":  s.store.Root,
		}); err != nil {
			writeErr(w, http.StatusInternalServerError, "template", "%v", err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	})
}

func (s *Server) tilesPageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := tilesTemplate.Execute(&buf, map[string]any{
			"Token":    s.opts.Token,
			"TilesDir": s.opts.TilesDir,
		}); err != nil {
			writeErr(w, http.StatusInternalServerError, "template", "%v", err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	})
}

func (s *Server) staticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("editorweb: static subtree: " + err.Error())
	}
	return http.StripPrefix("/static/", http.FileServerFS(sub))
}
