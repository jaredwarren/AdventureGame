package systems

import (
	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world"
)

// HazardSystem handles environment hazard lifecycles (e.g. active flames)
// and player hazard damage/knockback.
type HazardSystem struct{}

func (HazardSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	var activeFlames []world.ActiveFlame
	for _, f := range w.Flames {
		f.Timer--
		if f.Timer <= 0 {
			// Burn complete: destroy the tree tile!
			if broke, saveKey := w.BreakTileAt(f.TX, f.TY, world.DamageFire); broke {
				tryPush(bus, TileDestroyedEvent{SaveKey: saveKey})
			}
		} else {
			// Check player collision
			if w.Player.Invuln == 0 && w.Player.DodgeTimer == 0 {
				flameRect := geom.Rect{
					X: float64(f.TX * world.TileSize),
					Y: float64(f.TY * world.TileSize),
					W: float64(world.TileSize),
					H: float64(world.TileSize),
				}
				if w.PlayerRect().Overlaps(flameRect) {
					w.HP -= 1
					if w.HP < 0 {
						w.HP = 0
					}
					w.Player.Invuln = w.Player.EffectiveInvulnFrames()
					fcx := float64(f.TX*world.TileSize) + float64(world.TileSize)*0.5
					fcy := float64(f.TY*world.TileSize) + float64(world.TileSize)*0.5
					nx, ny := knockbackSlide(w, w.Player.X, w.Player.Y, w.Player.W, w.Player.H, fcx, fcy, w.Player.EffectivePlayerHazardKnockbackForce(), true)
					w.Player.X, w.Player.Y = nx, ny
					tryPush(bus, PlayerHurtEvent{Damage: 1})
				}
			}
			activeFlames = append(activeFlames, f)
		}
	}
	w.Flames = activeFlames
	return nil
}
