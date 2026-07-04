package tile

import "github.com/jaredwarren/game-test/internal/geom"

// SolidAt reads collision from the tile registry. Destroyable tiles are solid
// until their index is marked in destroyedTiles; lock tiles are solid until the
// player has a small key.
func SolidAt(gid int, tileIndex int, destroyedTiles map[int]bool, hasSmallKey bool) bool {
	return len(SolidRectsAt(gid, tileIndex, destroyedTiles, hasSmallKey)) > 0
}

// FullySolidAt returns true if the tile is completely impassable across its entire footprint.
func FullySolidAt(gid int, tileIndex int, destroyedTiles map[int]bool, hasSmallKey bool) bool {
	rects := SolidRectsAt(gid, tileIndex, destroyedTiles, hasSmallKey)
	return len(rects) == 1 && rects[0] == geom.Rect{X: 0, Y: 0, W: Size, H: Size}
}

// SolidRectsAt returns the sub-tile solid rectangles in tile local coordinates [0..16, 0..16].
// Returns nil if the tile is completely walkable.
func SolidRectsAt(gid int, tileIndex int, destroyedTiles map[int]bool, hasSmallKey bool) []geom.Rect {
	def := DefOf(gid)
	if def.Destroyable() {
		if destroyedTiles[tileIndex] {
			return nil
		}
		return []geom.Rect{{X: 0, Y: 0, W: Size, H: Size}}
	}
	if def.OpenableByKey {
		if hasSmallKey {
			return nil
		}
		return []geom.Rect{{X: 0, Y: 0, W: Size, H: Size}}
	}
	if !def.Solid {
		return nil
	}
	if len(def.SolidRects) > 0 {
		return def.SolidRects
	}
	return []geom.Rect{{X: 0, Y: 0, W: Size, H: Size}}
}

