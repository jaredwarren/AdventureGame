package editorweb

import "net/http"

// routes wires the API. All route strings live here and nowhere else.
//
// Map ids contain slashes ("rooms/boss"), and Go's {id} placeholder matches a
// single segment, so map routes use the trailing multi-segment wildcard {id...}.
// That wildcard is only legal at the end of a pattern, which is why anything
// needing a suffix (validate, thumbs) gets its own prefix instead of hanging off
// /api/maps/{id...}/....
//
// The exact literal /api/maps is more specific than /api/maps/{id...}, so the
// two do not conflict; TestRoutes pins that precedence.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /", s.indexHandler())
	mux.Handle("GET /static/", s.staticHandler())

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/schema", s.handleSchema)
	mux.HandleFunc("POST /api/marker", s.handleMarker)

	mux.HandleFunc("GET /api/maps", s.handleMapList)
	mux.HandleFunc("GET /api/maps/{id...}", s.handleMapLoad)
	mux.HandleFunc("PUT /api/maps/{id...}", s.handleMapSave)

	mux.HandleFunc("GET /api/thumbs/{id...}", s.handleThumb)

	mux.HandleFunc("POST /api/validate", s.handleValidate)
	mux.HandleFunc("GET /api/validate", s.handleValidateAll)

	return s.wrap(mux)
}
