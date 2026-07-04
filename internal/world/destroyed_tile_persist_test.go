package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

func TestApplyDestroyedTiles_Cracked(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID:          "m",
		MapW:           4,
		MapH:           4,
		Tiles:          make([]int, 16),
		DestroyedTiles: make(map[int]bool),
	}
	for i := range w.Tiles {
		w.Tiles[i] = tile.GIDGrass
	}
	w.Tiles[w.tileIndex(2, 2)] = tile.GIDCracked

	keys := map[string]struct{}{
		MapTilePersistKey("m", 2, 2):     {},
		MapTilePersistKey("other", 2, 2): {},
		MapTilePersistKey("m", 9, 9):     {},
	}
	ApplyDestroyedTiles(w, keys)
	if !w.DestroyedTiles[w.tileIndex(2, 2)] {
		t.Fatal("expected cracked tile marked destroyed")
	}
	if len(w.DestroyedTiles) != 1 {
		t.Fatalf("unexpected extra entries: %+v", w.DestroyedTiles)
	}
}

func TestApplyDestroyedTiles_Tree(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID:          "m",
		MapW:           4,
		MapH:           4,
		Tiles:          make([]int, 16),
		DestroyedTiles: make(map[int]bool),
	}
	for i := range w.Tiles {
		w.Tiles[i] = tile.GIDGrass
	}
	w.Tiles[w.tileIndex(1, 2)] = tile.GIDTree
	keys := map[string]struct{}{MapTilePersistKey("m", 1, 2): {}}
	ApplyDestroyedTiles(w, keys)
	if !w.DestroyedTiles[w.tileIndex(1, 2)] {
		t.Fatal("expected tree tile marked destroyed")
	}
}

func TestApplyDestroyedTiles_IgnoresNonDestroyable(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID:          "m",
		MapW:           3,
		MapH:           3,
		Tiles:          make([]int, 9),
		DestroyedTiles: make(map[int]bool),
	}
	for i := range w.Tiles {
		w.Tiles[i] = tile.GIDGrass
	}
	keys := map[string]struct{}{MapTilePersistKey("m", 1, 1): {}}
	ApplyDestroyedTiles(w, keys)
	if w.DestroyedTiles[w.tileIndex(1, 1)] {
		t.Fatal("should not mark non-destroyable tile")
	}
}

func TestBreakTileAt(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID: "m",
		MapW:  3,
		MapH:  3,
		Tiles: []int{
			tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
			tile.GIDGrass, tile.GIDCracked, tile.GIDGrass,
			tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
		},
		DestroyedTiles: make(map[int]bool),
	}

	// 1. Break out of bounds
	if ok, _ := w.BreakTileAt(-1, 0, tile.DamageBomb); ok {
		t.Fatal("expected out of bounds break to fail")
	}

	// 2. Break non-destroyable tile (Grass)
	if ok, _ := w.BreakTileAt(0, 0, tile.DamageBomb); ok {
		t.Fatal("expected breaking grass to fail")
	}

	// 3. Break destroyable tile (Cracked)
	ok, key := w.BreakTileAt(1, 1, tile.DamageBomb)
	if !ok {
		t.Fatal("expected breaking cracked tile to succeed")
	}
	expectedKey := MapTilePersistKey("m", 1, 1)
	if key != expectedKey {
		t.Fatalf("expected key %q, got %q", expectedKey, key)
	}
	if !w.DestroyedTiles[w.tileIndex(1, 1)] {
		t.Fatal("expected cracked tile to be marked destroyed")
	}

	// 4. Break already destroyed tile
	if ok, _ := w.BreakTileAt(1, 1, tile.DamageBomb); ok {
		t.Fatal("expected breaking already destroyed tile to fail")
	}
}
