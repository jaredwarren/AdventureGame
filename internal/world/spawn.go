package world

import (
	"strconv"
	"strings"
)

// Default player hitbox used by BuildFromTiled and as a safe fallback when
// resolving door spawns before a live Player exists.
const (
	DefaultPlayerHitboxW = 12.0
	DefaultPlayerHitboxH = 12.0
)

// DoorSpawnKeep is the authored spawn_x / spawn_y value that copies the
// player's matching axis at warp time.
const DoorSpawnKeep = "*"

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

// ParseDoorSpawnCoord parses a Tiled spawn_x / spawn_y string.
// Trimmed "*" means keep the player's matching axis (value is unused).
// Empty or missing is 0, matching the historical ParseFloat fallback.
// ok is false for any other non-numeric string.
func ParseDoorSpawnCoord(s string) (value float64, keep bool, ok bool) {
	s = strings.TrimSpace(s)
	if s == DoorSpawnKeep {
		return 0, true, true
	}
	if s == "" {
		return 0, false, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false, false
	}
	return v, false, true
}

// ResolveDoorSpawn converts a door's authored spawn (numbers and optional keep
// axes) plus the outgoing player top-left into destination player top-left.
func ResolveDoorSpawn(d Door, playerX, playerY, hitboxH float64) (x, y float64) {
	h := hitboxH
	if h <= 0 {
		h = DefaultPlayerHitboxH
	}
	sx, sy := d.SpawnX, d.SpawnY
	if d.KeepSpawnX {
		sx = playerX
	}
	if d.KeepSpawnY {
		if d.SpawnStyle == DoorSpawnTopLeft {
			sy = playerY
		} else {
			sy = playerY + h
		}
	}
	return PlayerTopLeftFromDoorSpawn(sx, sy, d.SpawnStyle, h)
}
