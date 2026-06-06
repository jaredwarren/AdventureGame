// timers.go — per-tick counter decay.
//
// All "frames remaining" counters are monotonically decremented here:
//
//   - World.Tick            (monotonic clock used by stamina regen cadence)
//   - World.DoorCooldown    (skip door warp triggers after a spawn)
//   - Player.Swing, SwingCD (sword animation / cooldown)
//   - Player.TorchSwing, TorchSwingCD (torch swing / cooldown)
//   - Player.Invuln         (post-hit i-frames)
//   - Player.DodgeTimer     (enemy-contact ignore window)
//   - Enemy.HurtCD          (cooldown between contact hits against the player)
//
// Keeping all decrements in one system means no other system has to
// remember "did I already tick my own timer this frame?" — answer: someone
// else did it first.
package systems

import (
	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world"
)

// TimersSystem is the unconditional-decay phase of each tick.
type TimersSystem struct{}

// Update decrements every per-tick counter on World. Safe when Enemies is
// empty; iteration simply does no work.
func (TimersSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	w.Tick++
	w.TimeOfDay = (w.TimeOfDay + 1) % world.CycleLength
	if w.DoorCooldown > 0 {
		w.DoorCooldown--
	}
	if w.Player.Swing > 0 {
		w.Player.Swing--
	}
	if w.Player.SwingCD > 0 {
		w.Player.SwingCD--
	}
	if w.Player.TorchSwing > 0 {
		w.Player.TorchSwing--
	}
	if w.Player.TorchSwingCD > 0 {
		w.Player.TorchSwingCD--
	}
	if w.Player.Invuln > 0 {
		w.Player.Invuln--
	}
	if w.Player.DodgeTimer > 0 {
		w.Player.DodgeTimer--
	}
	for i := range w.Enemies {
		if w.Enemies[i].HurtCD > 0 {
			w.Enemies[i].HurtCD--
		}
	}

	// Update active flames
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
					frames := w.Player.InvulnFrames
					if frames <= 0 {
						frames = 45
					}
					w.Player.Invuln = frames
					fcx := float64(f.TX*world.TileSize) + float64(world.TileSize)*0.5
					fcy := float64(f.TY*world.TileSize) + float64(world.TileSize)*0.5
					force := w.Player.PlayerHazardKnockbackForce
					if force <= 0 {
						force = 12
					}
					nx, ny := knockbackSlide(w, w.Player.X, w.Player.Y, w.Player.W, w.Player.H, fcx, fcy, force, true)
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
