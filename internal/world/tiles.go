// Tile GIDs. Per-tile behavior (collision, destructibility, lock state,
// fallback color) lives in tiledef.go; this file is the list of stable
// numeric IDs used in .tmj map data and Save files.
package world

// GID values match the hand-authored maps under assets/maps (firstgid=1 convention).
const (
	GIDEmpty   = 0
	GIDGrass   = 1
	GIDWall    = 2
	GIDCracked = 3
	GIDDoor    = 4 // warp trigger drawn as floor in collision; doors from objects preferred
	GIDWater   = 5
	GIDLock    = 6 // locked door until small key
	GIDFloor2  = 7
	GIDTree    = 8
)

// SolidAt reads collision from the TileDef registry. Destroyable tiles
// (cracked walls, trees, ...) are solid until their index is marked in
// destroyedTiles; lock tiles are solid until the player has a small key.
func SolidAt(gid int, tileIndex int, destroyedTiles map[int]bool, hasSmallKey bool) bool {
	def := TileDefOf(gid)
	if def.Destroyable() {
		return !destroyedTiles[tileIndex]
	}
	if def.OpenableByKey {
		return !hasSmallKey
	}
	return def.Solid
}
