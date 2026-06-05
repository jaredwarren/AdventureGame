// components.go — reusable building blocks for entities.
//
// Phase 4 introduces composition-first entities: instead of flat structs with
// mixed concerns (position + health + AI knobs + cooldown timers jumbled
// together), every entity is a composition of small, named components.
//
// Conventions:
//
//   - Components are plain struct types with exported fields. No methods
//     other than trivial helpers; behavior lives in systems.
//   - Components are embedded into entity types (Enemy, Player, Pickup,
//     ...) to promote fields. Accessing e.X reads Enemy.Transform.X.
//   - Components have no pointers to each other. An entity groups them by
//     value; cross-entity relationships go through EntityID (see entity.go).
//
// This deliberately stops short of a full ECS. We keep []Enemy / []Pickup
// slices and typed iteration; only the field layout changes. When (if) we
// need heterogeneous entity types at scale, we can adopt Donburi or Arche
// and this refactor will make that migration mechanical.
package world

// Transform is a 2D position in pixel-space. By convention (X, Y) is the
// top-left of the entity's AABB, not its center.
type Transform struct {
	X, Y float64
}

// Hitbox is the width+height of an entity's AABB in pixels. Paired with
// Transform to form a geom.Rect via the entity's Rect() method.
type Hitbox struct {
	W, H float64
}

// Facing stores a cardinal direction. Used by the player (sword arc, bomb
// targeting) and reserved for future enemies that aim.
type Facing struct {
	Dir Dir
}

// Health is an HP pool with an upper bound. Embedded on enemies so
// e.HP / e.MaxHP read naturally.
type Health struct {
	HP, MaxHP int
}

// AIHomer parameterizes the "seek the player at constant speed" behavior
// (see systems.EnemyAISystem). Speed is in pixels per tick.
// AggroRadius is center-to-center distance in pixels: pathing/chase runs only
// when the player is within this radius (contact damage is unchanged).
type AIHomer struct {
	Speed       float64
	AggroRadius float64
}

// ContactHurt describes "this entity damages the player on contact" plus
// its per-entity cooldown state. Damage is HP per hit; HurtCD counts down
// each tick and must be zero for a contact hit to land.
type ContactHurt struct {
	Damage int
	HurtCD int
}
