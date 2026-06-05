// burn.go — torch burn damage-over-time on ignited enemies.
package systems

import "github.com/jaredwarren/game-test/internal/world"

// BurnSystem ticks BurnTimer / BurnCD on each enemy and applies periodic
// damage. Runs after CombatSystem so a torch hit in the same frame sets
// BurnTimer before the first decay.
type BurnSystem struct{}

// Update applies burn ticks and emits HitEvent with FromBurnDoT set.
func (BurnSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.HP <= 0 || e.BurnTimer <= 0 {
			continue
		}
		e.BurnTimer--
		if e.BurnCD > 0 {
			e.BurnCD--
		}
		if e.BurnTimer <= 0 {
			e.BurnCD = 0
			continue
		}
		if e.BurnCD > 0 {
			continue
		}
		e.BurnCD = world.TorchBurnTickInterval
		before := e.HP
		w.DamageEnemy(i, world.TorchBurnDamagePerTick)
		killed := before > 0 && e.HP <= 0
		tryPush(bus, HitEvent{
			EnemyID:     e.ID,
			Damage:      world.TorchBurnDamagePerTick,
			Killed:      killed,
			IsBoss:      e.IsBoss,
			FromBurnDoT: true,
		})
	}
	return nil
}
