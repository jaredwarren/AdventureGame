// pickup.go — player vs pickup-item overlap.
//
// Pickup AABBs are 12x12 positioned at (p.X, p.Y). On overlap with the
// player rect:
//
//   - coin → currency++
//   - heart → HP++ (clamped to MaxHP)
//   - bomb → Bombs++ (capped at World.MaxBombsCarry)
//   - small key → SmallKey++
//   - torch → HasTorch = true
//
// The pickup is marked Gone so the renderer skips it and subsequent ticks
// ignore it. One PickupEvent per pickup per frame.
package systems

import (
	"github.com/jaredwarren/game-test/internal/world"
)

// PickupSystem consumes pickups the player is standing on.
type PickupSystem struct{}

// Update walks w.Pickups once per frame. Scales O(n) in pickup count;
// there are typically <50 per map so this is fine.
func (PickupSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	pr := w.Player.Rect()
	for i := range w.Pickups {
		p := &w.Pickups[i]
		if p.Gone {
			continue
		}
		if p.PendingCollect {
			p.PendingCollect = false
			w.ApplyPickupReward(p.Kind)
			tryPush(bus, PickupEvent{PickupID: p.ID, Kind: p.Kind, PersistentSaveKey: p.PersistentSaveKey})
			continue
		}
		if p.Opened || p.PersistentSaveKey != "" {
			continue
		}
		if !pr.Overlaps(p.Rect()) {
			continue
		}
		p.Gone = true
		w.ApplyPickupReward(p.Kind)
		tryPush(bus, PickupEvent{PickupID: p.ID, Kind: p.Kind, PersistentSaveKey: p.PersistentSaveKey})
	}
	return nil
}
