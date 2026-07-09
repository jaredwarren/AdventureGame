// entity.go — stable entity identity + construction helpers.
//
// EntityID gives every entity a stable, never-reused handle so events
// carry identity instead of slice indices. Indices are fragile — they
// shift on deletion, confuse ordering, and leak implementation detail.
// IDs don't.
//
// Allocation:
//
//   - World.nextID starts at 1. allocID() returns the current value and
//     bumps the counter. The zero value (NoEntity) is a sentinel for
//     "no entity / not yet assigned".
//   - IDs are monotonic across a World's lifetime but NOT globally unique
//     across Worlds. A new map build (BuildFromTiled) produces a fresh
//     counter; IDs are not comparable across reloads.
package world

// EntityID is a stable identifier for one entity in one World. See
// entity.go for allocation semantics.
type EntityID uint32

// NoEntity is the zero value of EntityID; never assigned to a live entity.
const NoEntity EntityID = 0

// allocID returns the next EntityID and advances the counter. Not safe
// for concurrent use; callers are single-threaded (sim update goroutine).
func (w *World) allocID() EntityID {
	if w.nextID == 0 {
		w.nextID = 1
	}
	id := EntityID(w.nextID)
	w.nextID++
	return id
}
