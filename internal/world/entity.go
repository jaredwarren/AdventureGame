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
//
// Factories:
//
//   - NewEnemy / NewPickup are pure functions that take the ID explicitly
//     so tests can construct entities without a *World. World method
//     versions (w.SpawnEnemy / w.SpawnPickup) auto-allocate the ID and
//     append to the appropriate slice.
package world

import "github.com/jaredwarren/game-test/internal/world/enemy"

// EntityID is a stable identifier for one entity in one World. See
// entity.go for allocation semantics.
type EntityID uint32

// NoEntity is the zero value of EntityID; never assigned to a live entity.
const NoEntity EntityID = 0

const (
	defaultEnemyW = enemy.HitboxW
	defaultEnemyH = enemy.HitboxH
)

// Pickup hitbox default. Ground pickups are 12x12 regardless of kind.
const defaultPickupHitbox = 12.0

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

// NewEnemy is the pure factory for Enemy values. hp <= 0 uses defaultEnemyHP.
func NewEnemy(id EntityID, x, y float64, hp int, isBoss bool) Enemy {
	cfg := DefaultEnemyConfig()
	if hp > 0 {
		cfg.HP = hp
	}
	cfg.IsBoss = isBoss
	return NewEnemyWithConfig(id, x, y, cfg)
}

// NewEnemyWithConfig builds an enemy from a fully resolved EnemyConfig.
func NewEnemyWithConfig(id EntityID, x, y float64, cfg EnemyConfig) Enemy {
	return Enemy{
		ID:        id,
		Transform: Transform{X: x, Y: y},
		Hitbox:    Hitbox{W: defaultEnemyW, H: defaultEnemyH},
		Health:    Health{HP: cfg.HP, MaxHP: cfg.HP},
		AI: AIHomer{
			Speed:       cfg.Speed,
			AggroRadius: cfg.AggroRadius,
		},
		ContactHurt: ContactHurt{
			Damage: cfg.ContactDamage,
		},
		IsBoss: cfg.IsBoss,
	}
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

// SpawnEnemy allocates an ID, builds the Enemy, appends it, and returns
// the freshly-assigned ID for callers that want to remember it.
func (w *World) SpawnEnemy(x, y float64, hp int, isBoss bool) EntityID {
	cfg := DefaultEnemyConfig()
	if hp > 0 {
		cfg.HP = hp
	}
	cfg.IsBoss = isBoss
	return w.SpawnEnemyConfig(x, y, cfg)
}

// SpawnEnemyConfig spawns an enemy using per-instance tuning from Tiled or tests.
func (w *World) SpawnEnemyConfig(x, y float64, cfg EnemyConfig) EntityID {
	e := NewEnemyWithConfig(w.allocID(), x, y, cfg)
	w.Enemies = append(w.Enemies, e)
	return e.ID
}

// SpawnPickup allocates an ID, builds the Pickup, appends it, and returns
// the ID. persistentSaveKey should be empty except for Tiled pickups marked
// persistent (see BuildFromTiled).
func (w *World) SpawnPickup(x, y float64, kind *PickupKind, persistentSaveKey string) EntityID {
	p := NewPickup(w.allocID(), x, y, kind)
	p.PersistentSaveKey = persistentSaveKey
	w.Pickups = append(w.Pickups, p)
	return p.ID
}
