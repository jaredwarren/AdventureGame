package world

import (
	"github.com/jaredwarren/game-test/internal/geom"
)

// Door is an authored map transition marker. Rect is baked by Tiled so it
// doesn't use Transform/Hitbox (shape can be non-entity-sized).
//
// SpawnX/SpawnY are interpreted using SpawnStyle (default DoorSpawnFeet).
// Optional Tiled property spawn_anchor: "feet" (default) or "topleft".
type Door struct {
	ID   EntityID
	Rect geom.Rect

	TargetMap  string
	SpawnX     float64
	SpawnY     float64
	SpawnStyle DoorSpawnStyle
}
