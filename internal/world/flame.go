package world

import (
	"github.com/jaredwarren/game-test/internal/world/tile"
)

type ActiveFlame struct {
	X, Y   float64
	TX, TY int
	Timer  int
}

// TryIgniteTree ignites a tree tile at tx, ty. Returns true if the tree was successfully ignited.
func (w *World) TryIgniteTree(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false
	}
	g := w.gidAt(tx, ty)
	if g != tile.GIDTree {
		return false
	}
	// Check if already burning
	for _, f := range w.Flames {
		if f.TX == tx && f.TY == ty {
			return false
		}
	}
	bx := float64(tx*tile.Size) + tile.Size*0.5
	by := float64(ty*tile.Size) + tile.Size*0.5
	w.Flames = append(w.Flames, ActiveFlame{
		X:     bx,
		Y:     by,
		TX:    tx,
		TY:    ty,
		Timer: w.EffectiveBalance().Hazards.TreeBurnDuration,
	})
	return true
}
