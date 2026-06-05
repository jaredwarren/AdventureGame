// combat.go — sword- and torch-vs-enemy damage, plus torch burning the
// facing tile (fire damage kind).
//
// Sword and torch share the same arc geometry and active swing window
// (see world.SwordHitbox / world.TorchHitbox) but never run in the same
// tick because TrySwingSword / TrySwingTorch are mutually exclusive.
//
// Torch hits apply immediate damage (same base as sword) and ignite the
// enemy for BurnSystem DoT. Each active torch frame also attempts
// world.TryDamageFaceTile(DamageFire) so trees and other fire-vulnerable
// tiles break like bomb timing (first matching frame wins; subsequent
// frames no-op once DestroyedTiles is set).
//
// One HitEvent is emitted per enemy hit per frame. TileDestroyedEvent is
// emitted when a fire-facing tile is destroyed this frame. Melee hits apply
// slight knockback away from the player (wall-slid).
package systems

import "github.com/jaredwarren/game-test/internal/world"

// CombatSystem resolves sword and torch impacts.
type CombatSystem struct{}

// Update tests the sword or torch hitbox against live enemies.
func (CombatSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	pcx, pcy := w.Player.Center()
	if hb, ok := w.SwordHitbox(); ok {
		dmg := w.SwordDamage()
		for i := range w.Enemies {
			e := &w.Enemies[i]
			if e.HP <= 0 {
				continue
			}
			if !hb.Overlaps(e.Rect()) {
				continue
			}
			before := e.HP
			w.DamageEnemy(i, dmg)
			nx, ny := knockbackSlide(w, e.X, e.Y, e.W, e.H, pcx, pcy, enemyMeleeKnockbackPx, false)
			e.X, e.Y = nx, ny
			killed := before > 0 && e.HP <= 0
			tryPush(bus, HitEvent{
				EnemyID: e.ID,
				Damage:  dmg,
				Killed:  killed,
				IsBoss:  e.IsBoss,
			})
		}
		return nil
	}
	if hb, ok := w.TorchHitbox(); ok {
		dmg := w.SwordDamage()
		for i := range w.Enemies {
			e := &w.Enemies[i]
			if e.HP <= 0 {
				continue
			}
			if !hb.Overlaps(e.Rect()) {
				continue
			}
			before := e.HP
			w.DamageEnemy(i, dmg)
			nx, ny := knockbackSlide(w, e.X, e.Y, e.W, e.H, pcx, pcy, enemyMeleeKnockbackPx, false)
			e.X, e.Y = nx, ny
			w.IgniteEnemy(i)
			killed := before > 0 && e.HP <= 0
			tryPush(bus, HitEvent{
				EnemyID: e.ID,
				Damage:  dmg,
				Killed:  killed,
				IsBoss:  e.IsBoss,
			})
		}
		if okTile, key := w.TryDamageFaceTile(world.DamageFire); okTile {
			tryPush(bus, TileDestroyedEvent{SaveKey: key})
		}
		return nil
	}
	return nil
}
