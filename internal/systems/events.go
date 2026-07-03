// events.go — the event bus and concrete event types.
//
// Events decouple systems (producers) from scenes (consumers). A system
// that lands a sword hit pushes HitEvent; the scene translates that into
// audio + camera shake + hit-stop. Neither side knows about the other.
//
// Bus model:
//
//   - Single-producer-side (systems) + single-consumer-side (scene) per
//     tick. We do not need concurrent safety: Update runs on the Ebiten
//     update goroutine, Drain runs right after on the same goroutine.
//
//   - Events are small value types; push is a slice append. Drain returns
//     the buffered slice and resets; the caller owns it until the next
//     Push. That avoids a per-tick allocation of the result slice at the
//     cost of requiring callers not to retain the slice across a tick.
//
//   - No subscribe/notify mechanism. If a new consumer appears we pass the
//     drained slice to it too; cross-scene fan-out has not been a need.
package systems

import "github.com/jaredwarren/game-test/internal/world"

// Event is the marker interface all events satisfy. Kept lightweight; no
// methods other than the unexported marker so we can use type-switch at
// the consumer.
type Event interface{ isEvent() }

// HitEvent is emitted when an enemy takes damage from the sword, torch
// swing, or torch burn DoT. Damage is the amount applied this frame;
// Killed is true only if this hit reduced the enemy from >0 to 0 HP.
// FromBurnDoT distinguishes burn ticks (lighter scene feedback).
//
// EnemyID references the enemy by stable id (see world.EntityID). Use
// it instead of slice-index references: indices shift when entities are
// compacted, ids do not.
type HitEvent struct {
	EnemyID world.EntityID
	Damage  int
	Killed  bool
	IsBoss  bool
	// FromBurnDoT is true for torch burn ticks (not the swing hit).
	// Scenes use this to avoid full hit-stop / heavy shake every tick.
	FromBurnDoT bool
}

func (HitEvent) isEvent() {}

// TileDestroyedEvent is emitted when a tile is destroyed by sim logic
// (e.g. torch fire or bomb) so the scene can persist MapTilePersistKey in Session.
type TileDestroyedEvent struct {
	SaveKey string
}

func (TileDestroyedEvent) isEvent() {}

// ExplosionEvent is emitted when a bomb fuse reaches zero and explodes.
type ExplosionEvent struct {
	X, Y   float64
	TX, TY int
}

func (ExplosionEvent) isEvent() {}

// PickupEvent is emitted when the player overlaps and consumes a pickup.
// Kind distinguishes coin/heart/bomb/key so the scene can play differ SFX
// (today they share one clip — kept for future differentiation).
type PickupEvent struct {
	PickupID          world.EntityID
	Kind              *world.PickupKind
	PersistentSaveKey string // empty unless Tiled pickup had persistent=true
}

func (PickupEvent) isEvent() {}

// PlayerHurtEvent is emitted when an enemy's contact damage lands on the
// player. Damage is the HP delta applied this frame (always positive).
type PlayerHurtEvent struct {
	EnemyID world.EntityID
	Damage  int
}

func (PlayerHurtEvent) isEvent() {}

// LockOpenEvent is emitted when the player standing on a GIDLock tile
// converts it to floor by consuming a SmallKey. Tile is (tx, ty) of one
// opened tile; a single step can open multiple tiles, in which case one
// event is emitted per opened tile.
type LockOpenEvent struct {
	Tile [2]int
}

func (LockOpenEvent) isEvent() {}

// EventBus is the per-tick event queue. Not safe for concurrent use.
type EventBus struct {
	events []Event
}

// Push appends ev to the bus. Typical usage inside a System.
func (b *EventBus) Push(ev Event) { b.events = append(b.events, ev) }

// Drain returns all pending events and clears the internal buffer. The
// returned slice is owned by the caller until the next Push; retain with
// care.
func (b *EventBus) Drain() []Event {
	out := b.events
	b.events = nil
	return out
}

// Len reports how many events are currently buffered. Handy in tests.
func (b *EventBus) Len() int { return len(b.events) }
