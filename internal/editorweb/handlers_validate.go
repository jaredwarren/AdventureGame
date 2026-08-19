package editorweb

import (
	"net/http"

	"github.com/jaredwarren/game-test/internal/tiled"
)

// validate runs the validator with this server's options and a resolver that can
// load door targets off disk.
func (s *Server) validate(m *tiled.Map, id string) []Issue {
	list := Validate(m, id, ValidateOpts{
		Strict:        s.opts.Strict,
		ResolveTarget: s.resolveTarget,
	})
	if list == nil {
		return []Issue{}
	}
	return list
}

// resolveTarget loads another map for cross-map door checks.
func (s *Server) resolveTarget(id string) (*tiled.Map, bool) {
	m, _, _, err := s.store.Read(id)
	if err != nil {
		return nil, false
	}
	return m, true
}

type validateRequest struct {
	ID  string     `json:"id"`
	Map *tiled.Map `json:"map"`
}

// handleValidate validates the client's IN-MEMORY document, so you can check a
// map before committing it to disk. This is the endpoint the editor debounces
// while you work.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := readJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "%v", err)
		return
	}
	if req.Map == nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "request is missing the \"map\" field")
		return
	}
	id, err := NormalizeMapID(req.ID)
	if err != nil {
		// Validation does not touch this path, so an unnamed document is fine;
		// the id only feeds save-key messages.
		id = "untitled"
	}
	issues := s.validate(req.Map, id)
	errs, warns := CountBySeverity(issues)
	writeJSON(w, http.StatusOK, map[string]any{
		"id": id, "issues": issues, "errorCount": errs, "warnCount": warns,
	})
}

// handleValidateAll sweeps every map on disk. This is what backs `mapeditor
// -check` and gives the map browser its red badges; the existing test suite
// covers only 7 of the ~104 maps.
func (s *Server) handleValidateAll(w http.ResponseWriter, r *http.Request) {
	report, err := s.ValidateAll()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list_failed", "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// ValidateReport is the whole-corpus validation result.
type ValidateReport struct {
	Maps       map[string][]Issue `json:"maps"`
	ErrorCount int                `json:"errorCount"`
	WarnCount  int                `json:"warnCount"`
	MapCount   int                `json:"mapCount"`
}

// ValidateAll validates every map in the store.
func (s *Server) ValidateAll() (*ValidateReport, error) {
	infos, err := s.store.List()
	if err != nil {
		return nil, err
	}
	report := &ValidateReport{Maps: map[string][]Issue{}, MapCount: len(infos)}

	for _, info := range infos {
		var issues []Issue
		if info.ParseError != "" {
			issues = []Issue{{
				Severity: SeverityError, Code: "parse_failed",
				Message: info.ParseError,
			}}
		} else {
			m, _, _, err := s.store.Read(info.ID)
			if err != nil {
				issues = []Issue{{Severity: SeverityError, Code: "io_error", Message: err.Error()}}
			} else {
				issues = s.validate(m, info.ID)
			}
		}
		errs, warns := CountBySeverity(issues)
		report.ErrorCount += errs
		report.WarnCount += warns
		if len(issues) > 0 {
			report.Maps[info.ID] = issues
		}
	}
	return report, nil
}
