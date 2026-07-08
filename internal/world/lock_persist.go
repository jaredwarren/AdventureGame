package world

import "github.com/jaredwarren/game-test/internal/world/tile"

// ApplyPersistedOpenedLocks converts GIDLock tiles back to walkable floor for
// any saved keys that match this world's MapID (after BuildFromTiled reload).
func ApplyPersistedOpenedLocks(w *World, keys map[string]struct{}) {
	if w == nil || len(keys) == 0 {
		return
	}
	for k := range keys {
		mid, tx, ty, ok := tile.ParseMapTilePersistKey(k)
		if !ok || mid != w.MapID {
			continue
		}
		if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
			continue
		}
		idx := w.tileIndex(tx, ty)
		if w.Tiles[idx] == tile.GIDLock {
			if len(w.Layers) > 0 {
				w.Tiles[idx] = tile.GIDEmpty
			} else {
				w.Tiles[idx] = tile.GIDGrass
			}
		}
	}
}
