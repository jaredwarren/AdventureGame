package tile

// SolidAt reads collision from the tile registry. Destroyable tiles are solid
// until their index is marked in destroyedTiles; lock tiles are solid until the
// player has a small key.
func SolidAt(gid int, tileIndex int, destroyedTiles map[int]bool, hasSmallKey bool) bool {
	def := DefOf(gid)
	if def.Destroyable() {
		return !destroyedTiles[tileIndex]
	}
	if def.OpenableByKey {
		return !hasSmallKey
	}
	return def.Solid
}
