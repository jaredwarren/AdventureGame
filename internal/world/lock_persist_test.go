package world

import "testing"

func TestApplyPersistedOpenedLocks(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID: "m",
		MapW:  5, MapH: 3,
		Tiles: make([]int, 15),
	}
	for i := range w.Tiles {
		w.Tiles[i] = GIDGrass
	}
	w.Tiles[w.tileIndex(2, 1)] = GIDLock
	keys := map[string]struct{}{
		MapTilePersistKey("m", 2, 1):     {},
		MapTilePersistKey("other", 2, 1): {},
		MapTilePersistKey("m", 99, 99):   {},
	}
	ApplyPersistedOpenedLocks(w, keys)
	if w.Tiles[w.tileIndex(2, 1)] != GIDGrass {
		t.Fatalf("expected lock converted to grass, gid=%d", w.Tiles[w.tileIndex(2, 1)])
	}
}

func TestApplyPersistedOpenedLocks_SkipsNonLock(t *testing.T) {
	t.Parallel()
	w := &World{MapID: "m", MapW: 3, MapH: 3, Tiles: make([]int, 9)}
	for i := range w.Tiles {
		w.Tiles[i] = GIDWall
	}
	keys := map[string]struct{}{MapTilePersistKey("m", 1, 1): {}}
	ApplyPersistedOpenedLocks(w, keys)
	if w.Tiles[w.tileIndex(1, 1)] != GIDWall {
		t.Fatal("non-lock tile should not change")
	}
}
