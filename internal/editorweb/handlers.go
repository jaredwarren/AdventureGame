package editorweb

import (
	"errors"
	"net/http"
	"os"

	"github.com/jaredwarren/game-test/internal/tiled"
)

// ---- schema and health ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	infos, err := s.store.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list_failed", "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"root":          s.store.Root,
		"mapCount":      len(infos),
		"schemaVersion": SchemaVersion,
		"pid":           os.Getpid(),
	})
}

// handleSchema serves the cached schema with a strong ETag. The payload is
// identical for the process lifetime, so a reload costs one 304.
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("ETag", `"`+s.schema.etag+`"`)
	w.Header().Set("Cache-Control", "no-cache")
	if match := r.Header.Get("If-None-Match"); match == `"`+s.schema.etag+`"` {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(s.schema.body)
}

// ---- maps ----

func (s *Server) handleMapList(w http.ResponseWriter, r *http.Request) {
	infos, err := s.store.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list_failed", "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"root": s.store.Root,
		"grid": map[string]any{
			"cols": GridCols, "rows": GridRows,
			"letterAxis": "row", "numberAxis": "col",
		},
		"maps": infos,
	})
}

// mapPayload is the response for loading one map.
type mapPayload struct {
	ID    string     `json:"id"`
	File  string     `json:"file"`
	Group MapGroup   `json:"group"`
	Grid  *GridPos   `json:"grid,omitempty"`
	ETag  string     `json:"etag"`
	Map   *tiled.Map `json:"map"`

	// Reformats warns that saving this map rewrites its formatting even with no
	// edits (whitespace, or omitempty defaults). See Reformats.
	Reformats bool `json:"reformats"`
	// UnmodeledFields lists JSON a save would silently delete. Non-empty means
	// the file carries Tiled features tiled.Map does not model.
	UnmodeledFields []string `json:"unmodeledFields,omitempty"`

	Issues []Issue `json:"issues"`
}

// handleMapLoad returns the RE-ENCODED map, not the raw file bytes, so the
// document the client holds is already canonical and an immediate save is a
// no-op diff.
func (s *Server) handleMapLoad(w http.ResponseWriter, r *http.Request) {
	id, ok := s.mapID(w, r)
	if !ok {
		return
	}
	m, raw, etag, err := s.store.Read(id)
	if err != nil {
		s.writeStoreErr(w, id, err)
		return
	}
	payload := mapPayload{
		ID:              id,
		Group:           GroupOf(id),
		ETag:            etag,
		Map:             m,
		Reformats:       Reformats(raw, m),
		UnmodeledFields: unmodeledFields(raw),
		Issues:          s.validate(m, id),
	}
	if p, err := s.store.Path(id); err == nil {
		payload.File = p
	}
	if g, ok := ParseGridID(id); ok {
		payload.Grid = &g
	}
	writeJSON(w, http.StatusOK, payload)
}

// saveRequest is the body of PUT /api/maps/{id...}.
type saveRequest struct {
	// BaseETag is the etag the client loaded. Empty skips the staleness check.
	BaseETag string `json:"baseEtag"`
	// Force is the "yes, I am sure" flag behind both save-anyway paths: it
	// writes despite validation errors, and it overwrites a file that changed on
	// disk. The UI only sets it from a dialog that has already explained which
	// case applies. It never bypasses path resolution.
	Force bool       `json:"force"`
	Map   *tiled.Map `json:"map"`
}

func (s *Server) handleMapSave(w http.ResponseWriter, r *http.Request) {
	id, ok := s.mapID(w, r)
	if !ok {
		return
	}
	var req saveRequest
	if err := readJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "%v", err)
		return
	}
	if req.Map == nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "request is missing the \"map\" field")
		return
	}
	if req.BaseETag == "" {
		if m := r.Header.Get("If-Match"); m != "" && m != "*" {
			req.BaseETag = trimQuotes(m)
		}
	}

	issues := s.validate(req.Map, id)
	if !req.Force {
		if errs := filterSeverity(issues, SeverityError); len(errs) > 0 {
			writeErrDetail(w, http.StatusBadRequest, "invalid_map",
				map[string]any{"issues": issues},
				"map has %d error-level problem(s); fix them or save with force", len(errs))
			return
		}
	}

	baseETag := req.BaseETag
	if req.Force {
		baseETag = "" // overwrite whatever is on disk
	}
	etag, err := s.store.Write(id, req.Map, baseETag)
	if errors.Is(err, ErrStale) {
		// Hand back the server's current document so the UI can offer a real
		// choice (reload / overwrite) instead of a dead end. This is the case
		// that fires when the in-game editor saved the same file.
		current, _, _, readErr := s.store.Read(id)
		detail := map[string]any{"currentEtag": etag}
		if readErr == nil {
			detail["serverMap"] = current
		}
		writeErrDetail(w, http.StatusConflict, "stale_etag", detail,
			"%s changed on disk since it was loaded", id)
		return
	}
	if err != nil {
		s.writeStoreErr(w, id, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"etag":   etag,
		"issues": issues,
	})
}

// ---- markers ----

type markerRequest struct {
	Type         string  `json:"type"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Name         string  `json:"name"`
	NextObjectID int     `json:"nextObjectId"`
	PickupKind   string  `json:"pickupKind"`
}

// handleMarker is the authoritative "create a marker" path. The schema ships
// templates for optimistic placement, but grid snapping and default properties
// are computed by internal/world and nowhere else, so door/sign placement cannot
// drift between the game and the editor.
func (s *Server) handleMarker(w http.ResponseWriter, r *http.Request) {
	var req markerRequest
	if err := readJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "%v", err)
		return
	}
	obj, err := NewMarkerObject(req.Type, req.NextObjectID, req.Name, req.X, req.Y, req.PickupKind)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "unknown_marker_type", "%v", err)
		return
	}
	next := req.NextObjectID + 1
	if req.NextObjectID == 0 {
		next = 0 // caller is not tracking ids; do not invent one
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object":       obj,
		"nextObjectId": next,
		"hitRect":      HitRectOf(obj),
	})
}

// ---- helpers ----

// mapID extracts and validates the {id...} path value.
func (s *Server) mapID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := NormalizeMapID(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_map_id", "invalid map id")
		return "", false
	}
	return id, true
}

func (s *Server) writeStoreErr(w http.ResponseWriter, id string, err error) {
	switch {
	case errors.Is(err, ErrMapNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "no map %q", id)
	case errors.Is(err, ErrBadMapID):
		writeErr(w, http.StatusBadRequest, "bad_map_id", "invalid map id")
	default:
		s.logger.Printf("map %s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "io_error", "%v", err)
	}
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
