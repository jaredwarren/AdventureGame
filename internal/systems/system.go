// system.go — the System interface.
//
// A System processes one tick of simulation for a given World. It has no
// memory between ticks (all state lives in World), no side channels (events
// go through EventBus), and no framework dependencies.
//
// Implementors are typically zero-size structs since they are pure
// behavior:
//
//	type TimersSystem struct{}
//	func (TimersSystem) Update(w *world.World, bus *EventBus, dt float64) error { ... }
//
// dt is seconds per tick. Today it is services.TickSeconds (1/60) and most
// systems ignore it; expose it so physics-like systems can integrate in SI
// units as they arrive.
package systems

import "github.com/jaredwarren/game-test/internal/world"

// System is one tick-processing unit.
//
// Update MUST be safe to call with a nil EventBus (test harnesses may pass
// nil when they don't care about events). Implementations guard accordingly
// via bus != nil checks before Push, or by using the tryPush helper below.
type System interface {
	Update(w *world.World, bus *EventBus, dt float64) error
}

// tryPush pushes ev to bus only if bus is non-nil. Used internally by the
// concrete systems so each one is safe to exercise in a test without
// constructing a bus.
func tryPush(bus *EventBus, ev Event) {
	if bus == nil {
		return
	}
	bus.Push(ev)
}
