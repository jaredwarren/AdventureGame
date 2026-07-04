// lock.go — GIDLock → floor tile conversion.
//
// When the player is standing on one or more GIDLock tiles AND has a
// SmallKey, the lock tile(s) are converted to GIDGrass (walkable) and one
// key is consumed. One LockOpenEvent is emitted per opened tile; the
// scene typically treats them as a single "lock opened" SFX trigger.
//
// Only one key is consumed per frame even if multiple lock tiles are
// touched simultaneously — matches the legacy behavior where a door-sized
// lock (2x2 tiles) costs one key, not four.
package systems

import (
	"math"

	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// LockSystem converts walk-on lock tiles when the player carries a key.
type LockSystem struct{}

func (LockSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	if w.SmallKey <= 0 {
		return nil
	}
	pr := w.PlayerRect()
	minTX := int(math.Floor(pr.X / tile.Size))
	minTY := int(math.Floor(pr.Y / tile.Size))
	maxTX := int(math.Floor((pr.X + pr.W - 1e-6) / tile.Size))
	maxTY := int(math.Floor((pr.Y + pr.H - 1e-6) / tile.Size))

	opened := false
	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
				continue
			}
			if !w.ConvertLockToFloor(tx, ty) {
				continue
			}
			opened = true
			tryPush(bus, LockOpenEvent{Tile: [2]int{tx, ty}})
		}
	}
	if !opened {
		return nil
	}
	w.SmallKey--
	if w.SmallKey < 0 {
		w.SmallKey = 0
	}
	return nil
}
