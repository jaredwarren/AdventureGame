package world

import (
	"math"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func (w *World) solidTile(tx, ty int) bool {
	g := w.gidAt(tx, ty)
	idx := w.tileIndex(tx, ty)
	return tile.FullySolidAt(g, idx, w.DestroyedTiles, w.SmallKey > 0)
}

// RectHitsSolid returns true if rect overlaps any solid tile or sub-tile solid region across all layers.
func (w *World) RectHitsSolid(r geom.Rect) bool {
	minTX := int(math.Floor(r.X / tile.Size))
	maxTX := int(math.Floor((r.X + r.W - 0.000001) / tile.Size))
	minTY := int(math.Floor(r.Y / tile.Size))
	maxTY := int(math.Floor((r.Y + r.H - 0.000001) / tile.Size))

	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
				tileRect := geom.Rect{X: float64(tx * tile.Size), Y: float64(ty * tile.Size), W: tile.Size, H: tile.Size}
				if r.Overlaps(tileRect) {
					return true
				}
				continue
			}
			idx := w.tileIndex(tx, ty)
			if len(w.Layers) > 0 {
				for k := 0; k < len(w.Layers); k++ {
					if idx < len(w.Layers[k]) {
						gid := w.Layers[k][idx]
						if gid == tile.GIDEmpty {
							continue
						}
						solidRects := tile.SolidRectsAt(gid, idx, w.DestroyedTiles, w.SmallKey > 0)
						for _, sr := range solidRects {
							worldSR := geom.Rect{
								X: float64(tx*tile.Size) + sr.X,
								Y: float64(ty*tile.Size) + sr.Y,
								W: sr.W,
								H: sr.H,
							}
							if r.Overlaps(worldSR) {
								return true
							}
						}
					}
				}
			} else {
				g := w.gidAt(tx, ty)
				solidRects := tile.SolidRectsAt(g, idx, w.DestroyedTiles, w.SmallKey > 0)
				for _, sr := range solidRects {
					worldSR := geom.Rect{
						X: float64(tx*tile.Size) + sr.X,
						Y: float64(ty*tile.Size) + sr.Y,
						W: sr.W,
						H: sr.H,
					}
					if r.Overlaps(worldSR) {
						return true
					}
				}
			}
		}
	}
	return false
}

func slideAABBAgnostic(x, y, ww, hh, dx, dy float64, blocked func(geom.Rect) bool) (float64, float64) {
	full := geom.Rect{X: x + dx, Y: y + dy, W: ww, H: hh}
	if !blocked(full) {
		return x + dx, y + dy
	}
	nx, ny := x, y
	if dx != 0 {
		if !blocked(geom.Rect{X: x + dx, Y: y, W: ww, H: hh}) {
			nx = x + dx
		}
	}
	if dy != 0 {
		if !blocked(geom.Rect{X: nx, Y: y + dy, W: ww, H: hh}) {
			ny = y + dy
		}
	}
	return nx, ny
}

// SlideAABB tries to move a rectangle by (dx, dy) with axis-separated sliding
// on solid tiles only (e.g. knockback that should not consider entities).
func (w *World) SlideAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, w.RectHitsSolid)
}

// RectOverlapsAnyLiveEnemy reports whether r overlaps any enemy with HP > 0.
func (w *World) RectOverlapsAnyLiveEnemy(r geom.Rect) bool {
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.HP <= 0 {
			continue
		}
		if r.Overlaps(e.Rect()) {
			return true
		}
	}
	return false
}

// SlidePlayerAABB slides the player's would-be AABB against solids, live
// enemies, and closed chests (no enemy-enemy blocking here).
func (w *World) SlidePlayerAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, func(r geom.Rect) bool {
		return w.RectHitsSolid(r) || w.RectOverlapsAnyLiveEnemy(r) || w.RectOverlapsAnyClosedChest(r)
	})
}

// SlideEnemyAABB slides an enemy AABB against solids, the player hitbox, and closed chests.
func (w *World) SlideEnemyAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	pr := w.PlayerRect()
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, func(r geom.Rect) bool {
		return w.RectHitsSolid(r) || r.Overlaps(pr) || w.RectOverlapsAnyClosedChest(r)
	})
}

// ConvertLockToFloor swaps a single GIDLock tile for walkable floor.
// Returns true when the tile at (tx, ty) was a lock and was converted;
// false (no mutation) otherwise. Out-of-bounds coordinates are a no-op.
//
// This is the structural primitive systems.LockSystem uses; keeping the
// Tile[] index math inside World preserves the invariant that external
// code never writes w.Tiles directly.
func (w *World) ConvertLockToFloor(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false
	}
	idx := w.tileIndex(tx, ty)
	if w.Tiles[idx] != tile.GIDLock {
		return false
	}
	if len(w.Layers) > 0 {
		w.Tiles[idx] = tile.GIDEmpty
	} else {
		w.Tiles[idx] = tile.GIDGrass
	}
	return true
}

// BreakTileAt breaks the tile at tx, ty with the given damage kind.
func (w *World) BreakTileAt(tx, ty int, kind tile.DamageKind) (ok bool, saveKey string) {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false, ""
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false, ""
	}
	g := w.gidAt(tx, ty)
	def := tile.DefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	return true, tile.MapTilePersistKey(w.MapID, tx, ty)
}

// TryDamageFaceTile applies the given damage kind to the tile directly
// in front of the player. On success it marks the tile destroyed and
// returns MapTilePersistKey for session/save persistence. A tile is
// only broken when its TileDef.DamageKinds contains kind, so bombs do
// not harm fire-only tiles (e.g. trees) and vice versa.
//
// DamageBomb requires Bombs > 0 and decrements Bombs on success.
//
// NOTE: Uses player center for the "from" tile math, then steps one
// tile in Dir—works for current hitbox sizes.
func (w *World) TryDamageFaceTile(kind tile.DamageKind) (ok bool, saveKey string) {
	if kind == tile.DamageBomb && w.Bombs <= 0 {
		return false, ""
	}
	pc := w.PlayerRect()
	cx := pc.X + pc.W*0.5
	cy := pc.Y + pc.H*0.5
	tx := int(cx / tile.Size)
	ty := int(cy / tile.Size)
	switch w.Player.Dir {
	case DirDown:
		ty++
	case DirUp:
		ty--
	case DirLeft:
		tx--
	case DirRight:
		tx++
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false, ""
	}
	g := w.gidAt(tx, ty)
	def := tile.DefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	if kind == tile.DamageBomb {
		w.Bombs--
		if w.Bombs < 0 {
			w.Bombs = 0
		}
	}
	return true, tile.MapTilePersistKey(w.MapID, tx, ty)
}
