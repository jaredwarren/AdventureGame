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

import "github.com/jaredwarren/game-test/internal/world"

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
	if len(w.ActiveBuffs) > 0 {
		active := w.ActiveBuffs[:0]
		for i := range w.ActiveBuffs {
			b := &w.ActiveBuffs[i]
			if b.OnTick != nil {
				b.OnTick(w, b)
			}
			if b.Duration > 0 {
				b.Duration--
			}
			if b.Duration == 0 {
				if b.OnExpire != nil {
					b.OnExpire(w, b)
				}
			} else {
				active = append(active, *b)
			}
		}
		w.ActiveBuffs = active
	}
	return nil
}
