package world

import "testing"

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
		w.Tiles[i] = GIDGrass
	}
	w.Tiles[w.tileIndex(2, 2)] = GIDCracked

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
		w.Tiles[i] = GIDGrass
	}
	w.Tiles[w.tileIndex(1, 2)] = GIDTree
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
		w.Tiles[i] = GIDGrass
	}
	keys := map[string]struct{}{MapTilePersistKey("m", 1, 1): {}}
	ApplyDestroyedTiles(w, keys)
	if w.DestroyedTiles[w.tileIndex(1, 1)] {
		t.Fatal("should not mark non-destroyable tile")
	}
}
