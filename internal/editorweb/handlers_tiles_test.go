package editorweb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

func TestTileListAndLoadSave(t *testing.T) {
	dir := t.TempDir()
	mapsDir := filepath.Join(dir, "maps")
	tilesDir := filepath.Join(dir, "tiles")
	if err := os.MkdirAll(mapsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	srv, err := New(Options{
		MapsDir:     mapsDir,
		TilesDir:    tilesDir,
		Token:       "test-token",
		Addr:        "127.0.0.1:0",
		AllowRemote: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/tiles", nil)
	req.Header.Set("X-Editor-Token", "test-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tiles/1", nil)
	req.Header.Set("X-Editor-Token", "test-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("load status %d: %s", rec.Code, rec.Body.String())
	}
	var loaded tileArtResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Art.GID != 1 || loaded.Art.Name != "grass" {
		t.Fatalf("unexpected art %#v", loaded.Art)
	}
	if loaded.Art.Spatial == nil || len(loaded.Art.Spatial.Variants) < 2 {
		t.Fatal("expected spatial variants on grass")
	}

	art := tile.Art{
		GID:  4,
		Name: "door",
		Size: 16,
		Layers: []tile.Shape{
			{ID: "fill", Type: "rect", X: 0, Y: 0, W: 16, H: 16, Fill: "#3a5a3a"},
		},
	}
	body, err := json.Marshal(map[string]any{"art": art})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, "/api/tiles/4", bytes.NewReader(body))
	req.Header.Set("X-Editor-Token", "test-token")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status %d: %s", rec.Code, rec.Body.String())
	}
	if _, ok := tile.ArtFilePath(tilesDir, 4); !ok {
		t.Fatal("expected door art file on disk")
	}
	if !tile.HasArt(4) {
		t.Fatal("expected art applied after save")
	}
}

func TestSynthesizeArtFromDrawer(t *testing.T) {
	art, err := synthesizeArtFromDrawer(tile.GIDWall)
	if err != nil {
		t.Fatal(err)
	}
	if art.GID != tile.GIDWall || len(art.Layers) == 0 {
		t.Fatalf("unexpected %#v", art)
	}
	if err := art.Validate(); err != nil {
		t.Fatal(err)
	}
}
