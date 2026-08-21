package editorweb

import (
	"net/http"
	"strconv"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

type tileListItem struct {
	GID        int    `json:"gid"`
	Name       string `json:"name"`
	HasArt     bool   `json:"hasArt"`
	HasSpatial bool   `json:"hasSpatial"`
	Swatch     string `json:"swatch"`
}

type tileArtResponse struct {
	Art         tile.Art `json:"art"`
	Synthesized bool     `json:"synthesized"`
	Path        string   `json:"path,omitempty"`
}

func (s *Server) handleTileList(w http.ResponseWriter, r *http.Request) {
	gids := tile.RegisteredGIDs()
	out := make([]tileListItem, 0, len(gids))
	for _, gid := range gids {
		d := tile.DefOf(gid)
		item := tileListItem{
			GID:    gid,
			Name:   d.Name,
			Swatch: hexColor(d.SwatchColor),
		}
		if art, ok := tile.ArtOf(gid); ok {
			item.HasArt = true
			item.HasSpatial = art.Spatial != nil && len(art.Spatial.Variants) > 0
		} else if _, ok := tile.ArtFilePath(s.opts.TilesDir, gid); ok {
			item.HasArt = true
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tiles": out, "tilesDir": s.opts.TilesDir})
}

func (s *Server) handleTileLoad(w http.ResponseWriter, r *http.Request) {
	gid, err := strconv.Atoi(r.PathValue("gid"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_gid", "gid must be an integer")
		return
	}
	d := tile.DefOf(gid)
	if d.Name == "unknown" {
		writeErr(w, http.StatusNotFound, "not_found", "unknown gid %d", gid)
		return
	}

	if art, ok := tile.ArtOf(gid); ok {
		path, _ := tile.ArtFilePath(s.opts.TilesDir, gid)
		writeJSON(w, http.StatusOK, tileArtResponse{Art: art, Path: path})
		return
	}
	if path, ok := tile.ArtFilePath(s.opts.TilesDir, gid); ok {
		if err := tile.LoadArtDir(s.opts.TilesDir); err == nil {
			if art, ok := tile.ArtOf(gid); ok {
				writeJSON(w, http.StatusOK, tileArtResponse{Art: art, Path: path})
				return
			}
		}
	}

	art, err := synthesizeArtFromDrawer(gid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "synthesize", "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, tileArtResponse{Art: art, Synthesized: true})
}

func (s *Server) handleTileSave(w http.ResponseWriter, r *http.Request) {
	gid, err := strconv.Atoi(r.PathValue("gid"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_gid", "gid must be an integer")
		return
	}
	var body struct {
		Art tile.Art `json:"art"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_json", "%v", err)
		return
	}
	art := body.Art
	art.GID = gid
	if art.Name == "" {
		art.Name = tile.DefOf(gid).Name
	}
	if art.Size == 0 {
		art.Size = tile.Size
	}
	path, err := tile.SaveArtFile(s.opts.TilesDir, art)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_art", "%v", err)
		return
	}
	if err := s.rebuildSchema(); err != nil {
		s.logger.Printf("rebuild schema after tile save: %v", err)
	}
	writeJSON(w, http.StatusOK, tileArtResponse{Art: art, Path: path})
}

func (s *Server) rebuildSchema() error {
	cs, err := newCachedSchema(s.opts.Anim)
	if err != nil {
		return err
	}
	s.schema = cs
	return nil
}
