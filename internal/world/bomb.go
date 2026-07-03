package world

// ActiveBomb represents a placed bomb with a frame-counted fuse timer.
type ActiveBomb struct {
	X, Y   float64
	TX, TY int
	Timer  int
}

// TryPlaceBomb attempts to place an active bomb in front of the player.
// Returns true if a bomb was placed.
func (w *World) TryPlaceBomb() bool {
	if w.Bombs <= 0 {
		return false
	}
	pc := w.PlayerRect()
	cx := pc.X + pc.W*0.5
	cy := pc.Y + pc.H*0.5
	tx := int(cx / TileSize)
	ty := int(cy / TileSize)
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
	bx := float64(tx*TileSize) + TileSize*0.5
	by := float64(ty*TileSize) + TileSize*0.5
	for _, b := range w.ActiveBombs {
		if b.TX == tx && b.TY == ty {
			return false
		}
	}
	w.Bombs--
	w.ActiveBombs = append(w.ActiveBombs, ActiveBomb{
		X:     bx,
		Y:     by,
		TX:    tx,
		TY:    ty,
		Timer: w.Player.EffectiveBombFuseDuration(),
	})
	return true
}
