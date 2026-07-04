package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func grassWorldForPath(w, h int, wallIdx map[int]bool) *World {
	tiles := make([]int, w*h)
	for i := range tiles {
		tiles[i] = tile.GIDGrass
	}
	for idx := range wallIdx {
		if idx >= 0 && idx < len(tiles) {
			tiles[idx] = tile.GIDWall
		}
	}
	return &World{
		MapID:          "test",
		TileW:          tile.Size,
		TileH:          tile.Size,
		MapW:           w,
		MapH:           h,
		Tiles:          tiles,
		DestroyedTiles: map[int]bool{},
		Stats:          progression.DefaultStats(),
	}
}

func TestFindPathAStar_DirectLine(t *testing.T) {
	w := grassWorldForPath(5, 5, nil)
	path := w.findPathAStar(0, 0, 4, 0)
	if len(path) != 5 {
		t.Fatalf("path len = %d, want 5: %v", len(path), path)
	}
}

func TestFindPathAStar_AroundWall(t *testing.T) {
	w := grassWorldForPath(5, 5, nil)
	// Single solid cell on the straight line — path must detour above/below.
	w.Tiles[2*w.MapW+2] = tile.GIDWall
	path := w.findPathAStar(0, 2, 4, 2)
	if path == nil {
		t.Fatal("expected path around wall")
	}
	if len(path) <= 5 {
		t.Fatalf("path should detour (len > 5), got %v", path)
	}
	last := path[len(path)-1]
	if last.tx != 4 || last.ty != 2 {
		t.Fatalf("end = (%d,%d), want (4,2)", last.tx, last.ty)
	}
}

func TestEnemyChaseTarget_NextTile(t *testing.T) {
	w := grassWorldForPath(5, 5, nil)
	tgx, tgy := w.EnemyChaseTarget(
		float64(0)*tile.Size+tile.Size*0.5,
		float64(2)*tile.Size+tile.Size*0.5,
		float64(4)*tile.Size+tile.Size*0.5,
		float64(2)*tile.Size+tile.Size*0.5,
	)
	wantX := float64(1)*tile.Size + tile.Size*0.5
	wantY := float64(2)*tile.Size + tile.Size*0.5
	if tgx != wantX || tgy != wantY {
		t.Fatalf("chase target = (%.1f,%.1f), want (%.1f,%.1f)", tgx, tgy, wantX, wantY)
	}
}
