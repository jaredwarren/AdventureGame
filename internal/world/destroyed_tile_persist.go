package world

// ApplyDestroyedTiles marks World.DestroyedTiles for any saved keys
// whose map_id matches this world and whose GID is a destroyable tile
// (any TileDef with a non-empty DamageKinds).
//
// Keys are produced by MapTilePersistKey when a tile is broken and
// serialized via save.GameSave.DestroyedTileKeys.
func ApplyDestroyedTiles(w *World, keys map[string]struct{}) {
	if w == nil || len(keys) == 0 {
		return
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	for k := range keys {
		mid, tx, ty, ok := ParseMapTilePersistKey(k)
		if !ok || mid != w.MapID {
			continue
		}
		if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
			continue
		}
		idx := w.tileIndex(tx, ty)
		if TileDefOf(w.Tiles[idx]).Destroyable() {
			w.DestroyedTiles[idx] = true
		}
	}
}
