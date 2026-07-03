// pipeline.go — ordered runner for Systems.
//
// Order matters. The Default() pipeline mirrors the legacy UpdateSim
// ordering so behavior is identical after Phase 3:
//
//  1. Timers    — decrement per-tick counters (Invuln, Swing, TorchSwing, ...)
//  2. Combat    — sword/torch vs enemies; torch vs fire-vulnerable tiles
//  3. Burn      — torch DoT on ignited enemies
//  4. Bomb      — bomb fuse decay, tile destruction, radial damage
//  5. Hazard    — active flame decay, tree destruction, hazard player collision
//  6. Pickup    — player-pickup overlap; emits PickupEvent
//  7. EnemyAI   — A* chase target + contact damage; emits PlayerHurtEvent
//  8. Lock      — GIDLock conversion under player; emits LockOpenEvent
//  9. Stamina   — sprint stamina drain and recovery
//
// Combat runs AFTER Timers because Timers decrements Player.Swing: combat's
// SwordHitbox read observes the post-decrement value, matching the legacy
// code. EnemyAI runs AFTER Timers for the same reason (HurtCD / ContactCD).
// BombSystem runs AFTER Combat/Burn and BEFORE Pickup so dropped loot from
// bomb kills is collectible same-tick. HazardSystem runs AFTER BombSystem so
// a bomb-lit flame damages same tick as it appears. StaminaSystem runs LAST
// (after Lock) to drain or regen stamina at tick completion.
package systems

import "github.com/jaredwarren/game-test/internal/world"

// Pipeline owns an ordered list of Systems and an internal EventBus.
type Pipeline struct {
	systems []System
	bus     EventBus
}

// NewPipeline builds a pipeline from the provided systems in order. Pass
// Default() for the stock sim; pass a custom order for tests that want to
// exercise one system in isolation.
func NewPipeline(ss ...System) *Pipeline {
	return &Pipeline{systems: append([]System(nil), ss...)}
}

// Default returns a pipeline with the canonical sim order.
func Default() *Pipeline {
	return NewPipeline(
		TimersSystem{},
		CombatSystem{},
		BurnSystem{},
		BombSystem{},
		HazardSystem{},
		PickupSystem{},
		EnemyAISystem{},
		LockSystem{},
		StaminaSystem{},
	)
}

// Tick runs all systems in order and returns the buffered events.
//
// The returned slice is owned by the caller until the next Tick; do NOT
// retain references across ticks. If any system returns an error, Tick
// stops and returns that error immediately — subsequent systems do not run.
// The event buffer is drained regardless so the caller can inspect partial
// progress.
func (p *Pipeline) Tick(w *world.World, dt float64) ([]Event, error) {
	var firstErr error
	for _, s := range p.systems {
		if err := s.Update(w, &p.bus, dt); err != nil {
			firstErr = err
			break
		}
	}
	return p.bus.Drain(), firstErr
}
