package systems

import (
	"math"

	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// BombSystem ticks fuses on placed bombs, breaking tiles, damaging enemies,
// and emitting ExplosionEvent upon expiration.
type BombSystem struct{}

func (BombSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	var active []world.ActiveBomb
	for _, b := range w.ActiveBombs {
		b.Timer--
		if b.Timer <= 0 {
			if broke, saveKey := w.BreakTileAt(b.TX, b.TY, tile.DamageBomb); broke {
				tryPush(bus, TileDestroyedEvent{SaveKey: saveKey})
			}

			bombRadius := w.Player.EffectiveBombRadius()
			bombDamage := w.Player.EffectiveBombDamage()
			for i := range w.Enemies {
				enemy := &w.Enemies[i]
				if enemy.HP <= 0 {
					continue
				}
				ecx, ecy := enemy.Center()
				dist := math.Hypot(ecx-b.X, ecy-b.Y)
				if dist <= bombRadius {
					applyEnemyDamage(w, bus, i, bombDamage, b.X, b.Y, 0, false)
				}
			}

			tryPush(bus, ExplosionEvent{
				X:  b.X,
				Y:  b.Y,
				TX: b.TX,
				TY: b.TY,
			})
		} else {
			active = append(active, b)
		}
	}
	w.ActiveBombs = active
	return nil
}
