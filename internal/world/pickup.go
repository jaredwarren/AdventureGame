package world

import (
	"github.com/jaredwarren/game-test/internal/geom"
)

// Pickup hitbox default. Ground pickups are 12x12 regardless of kind.
const defaultPickupHitbox = 12.0

// Pickup is a ground item the player can collect by overlap. Transform +
// Hitbox give it its AABB; Kind drives the on-pickup effect in
// systems.PickupSystem.
type Pickup struct {
	ID EntityID
	Transform
	Hitbox

	Kind           *PickupKind
	Gone           bool // set true when consumed; kept in the slice for stable indices
	Opened         bool // set true when chest is opened
	PendingCollect bool // set true when chest is opened by player intent until PickupSystem processes it
	IsChest        bool // set true if this is visually and behaviorally a chest

	// PersistentSaveKey is non-empty when this pickup was authored with
	// persistent=true in Tiled; format from PersistentPickupSaveKey. Consumed
	// pickups stay absent across map reloads while this key remains in save.
	PersistentSaveKey string
}

// Rect returns the pickup's AABB.
func (p Pickup) Rect() geom.Rect { return geom.Rect{X: p.X, Y: p.Y, W: p.W, H: p.H} }

// MaxBombsCarry returns the player's effective bomb inventory cap.
func (w *World) MaxBombsCarry() int {
	return w.Player.EffectiveMaxBombs()
}

// ClampBombsCarry clamps n to [0, MaxBombsCarry] for save load and pickups.
func (w *World) ClampBombsCarry(n int) int {
	if n < 0 {
		return 0
	}
	maxB := w.MaxBombsCarry()
	if n > maxB {
		return maxB
	}
	return n
}

// RectOverlapsAnyClosedChest reports whether r overlaps any closed chest pickup.
func (w *World) RectOverlapsAnyClosedChest(r geom.Rect) bool {
	for i := range w.Pickups {
		p := &w.Pickups[i]
		if p.IsChest && !p.Opened && !p.Gone {
			if r.Overlaps(p.Rect()) {
				return true
			}
		}
	}
	return false
}

// NewPickup is the pure factory for Pickup values.
func NewPickup(id EntityID, x, y float64, kind *PickupKind) Pickup {
	return Pickup{
		ID:        id,
		Transform: Transform{X: x, Y: y},
		Hitbox:    Hitbox{W: defaultPickupHitbox, H: defaultPickupHitbox},
		Kind:      kind,
	}
}

// SpawnPickup allocates an ID, builds the Pickup, appends it, and returns
// the ID. persistentSaveKey should be empty except for Tiled pickups marked
// persistent (see BuildFromTiled).
func (w *World) SpawnPickup(x, y float64, kind *PickupKind, persistentSaveKey string, isChest bool) EntityID {
	p := NewPickup(w.allocID(), x, y, kind)
	p.PersistentSaveKey = persistentSaveKey
	p.IsChest = isChest
	w.Pickups = append(w.Pickups, p)
	return p.ID
}
