package world

import (
	"github.com/jaredwarren/game-test/internal/geom"
)

// Door is an authored map transition marker. Rect is baked by Tiled so it
// doesn't use Transform/Hitbox (shape can be non-entity-sized).
//
// SpawnX/SpawnY are interpreted using SpawnStyle (default DoorSpawnFeet).
// Optional Tiled property spawn_anchor: "feet" (default) or "topleft".
// KeepSpawnX/KeepSpawnY come from authored spawn_x/spawn_y of "*" and copy
// the matching player axis at warp time.
type Door struct {
	ID   EntityID
	Rect geom.Rect

	TargetMap  string
	SpawnX     float64
	SpawnY     float64
	KeepSpawnX bool
	KeepSpawnY bool
	SpawnStyle DoorSpawnStyle
}
