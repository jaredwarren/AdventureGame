package world

// Default player hitbox used by BuildFromTiled and as a safe fallback when
// resolving door spawns before a live Player exists.
const (
	DefaultPlayerHitboxW = 12.0
	DefaultPlayerHitboxH = 12.0
)

// DoorSpawnStyle describes how door properties spawn_x / spawn_y map to the
// player hitbox top-left (Transform X/Y). Matches Tiled "spawn" markers by default.
type DoorSpawnStyle uint8

const (
	// DoorSpawnFeet means (SpawnX, SpawnY) is the bottom of the player AABB
	// (feet on the ground), same convention as BuildFromTiled "spawn" objects.
	DoorSpawnFeet DoorSpawnStyle = iota
	// DoorSpawnTopLeft means (SpawnX, SpawnY) is already the player hitbox top-left.
	DoorSpawnTopLeft
)

// PlayerTopLeftFromDoorSpawn converts authored door spawn coordinates to player
// Transform (top-left of hitbox). hitboxH should be the player’s H; if <= 0,
// DefaultPlayerHitboxH is used.
func PlayerTopLeftFromDoorSpawn(spawnX, spawnY float64, style DoorSpawnStyle, hitboxH float64) (x, y float64) {
	h := hitboxH
	if h <= 0 {
		h = DefaultPlayerHitboxH
	}
	if style == DoorSpawnTopLeft {
		return spawnX, spawnY
	}
	return spawnX, spawnY - h
}
