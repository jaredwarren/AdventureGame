// Package world is the gameplay simulation layer: player movement, sword hits,
// pickups, and simple enemy AI. It deliberately knows nothing about Ebiten—only
// numbers and structs—so you can unit-test logic without graphics.
//
// Domain-specific tables live in subpackages (import them directly when adding
// new content):
//   - world/tile   — GIDs, collision defs, tile persist keys
//   - world/pickup — collectible kinds and rewards
//   - world/enemy  — per-enemy Tiled tuning
//
// The root world package re-exports common symbols via aliases.go so existing
// callers can keep using world.GIDGrass, world.PickupCoin, etc.
//
// Coordinate conventions:
//   - Tile grid: origin top-left, tile size [TileSize] pixels (must match Tiled export).
//   - Player/enemy positions are pixel AABBs (upper-left of hitbox).
//
// Temporary / improvement notes:
//   - Collision uses 5-point sampling on the player rect (corners + center)—fast but can miss thin walls;
//     consider continuous collision or capsule if tunnels get narrower than the probe spacing.
//   - Sword hitbox timing (Swing countdown + “active window”) is hand-tuned feel, not derived from animation.
//   - Enemy chase uses tile A* (see astar.go) toward the player with axis slide; no enemy–enemy avoidance.
//   - Contact damage uses separate cooldowns on player invuln vs enemy HurtCD—tweak together for fairness.
package world

import (
	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// ItemSlot identifies which item is currently active in the player's secondary slot.
type ItemSlot int

const (
	ItemSlotBomb ItemSlot = iota // default
	ItemSlotTorch
)

type World struct {
	MapID string
	TileW int
	TileH int
	MapW  int
	MapH  int

	// Layers holds dynamic 2D tile layers [layerIndex][tileIndex] (bottom-to-top order).
	// Tiles is kept as a convenience reference to the top active layer.
	Layers [][]int
	Tiles  []int

	// ActiveLayerFilter when >= 0 restricts rendering to only that layer index. -1 shows all layers.
	ActiveLayerFilter int

	// DestroyedTiles records tiles that have been broken/opened by damage
	// (bombs, fire, ...) keyed by tile index. Rendering and collision
	// treat these tiles as removed, revealing the layer below.
	DestroyedTiles map[int]bool
	Doors          []Door
	Shrines        []Shrine
	Signs          []Sign
	Pickups        []Pickup
	IsEditor       bool // set true when running in editor mode
	Enemies        []Enemy

	Player   Player
	Stats    progression.Stats
	HP       int
	Currency int
	// Bombs is how many bomb items the player holds; pickups increment (capped at
	// Player.MaxBombs via MaxBombsCarry), placing a bomb decrements (see TryDamageFaceTile(DamageBomb)).
	Bombs           int
	HasTorch        bool
	HasPegasusBoots bool
	ShieldLevel     int
	OwnedItems      map[string]bool
	ActiveBuffs     []Buff
	SmallKey        int
	SelectedItem    ItemSlot

	// Tick is the sim-frame counter; incremented by systems.TimersSystem so
	// other systems can key periodic effects off a world-owned clock.
	Tick int

	// TimeOfDay is the current frame in the 14,400 frame cycle (4 minutes).
	TimeOfDay int

	// HasAmbientLightOverride is true if the map specified a custom light level override
	HasAmbientLightOverride bool
	AmbientLightOverride    float64

	// Flames is the list of currently active fires burning tiles (e.g. trees)
	Flames []ActiveFlame

	// ActiveBombs is the list of placed bombs counting down fuse timers
	ActiveBombs []ActiveBomb

	// DoorCooldown counts down while >0 (systems.TimersSystem); while non-
	// zero, scenes should skip door warp checks so the player does not
	// immediately re-trigger the door they spawned on after a map
	// transition.
	DoorCooldown int

	// Balance is the active GameBalance configuration; when nil, EffectiveBalance
	// falls back to balance.Default().
	Balance *balance.GameBalance

	// nextID feeds allocID(). Unexported so external code can't mint IDs
	// out of band; use the Spawn* methods instead.
	nextID uint32
}

// EffectiveBalance returns w.Balance if non-nil, or balance.Default() otherwise.
func (w *World) EffectiveBalance() *balance.GameBalance {
	if w != nil && w.Balance != nil {
		return w.Balance
	}
	return balance.Default()
}

func (w *World) gidAt(tx, ty int) int {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return tile.GIDWall
	}
	idx := ty*w.MapW + tx
	if len(w.Layers) > 0 {
		for k := len(w.Layers) - 1; k >= 0; k-- {
			if idx < len(w.Layers[k]) {
				gid := w.Layers[k][idx]
				if gid != tile.GIDEmpty && !w.DestroyedTiles[idx] {
					return gid
				}
				if gid != tile.GIDEmpty && k == 0 {
					return gid
				}
			}
		}
	}
	if idx < len(w.Tiles) {
		return w.Tiles[idx]
	}
	return tile.GIDEmpty
}

// GIDAt returns the tile GID at tile coordinates (for debug / tools).
func (w *World) GIDAt(tx, ty int) int {
	return w.gidAt(tx, ty)
}

// SurfaceAtFeet returns the SurfaceDef for the tile at world position (px, py).
func (w *World) SurfaceAtFeet(px, py float64) tile.SurfaceDef {
	if w == nil || w.TileW <= 0 || w.TileH <= 0 {
		return tile.DefaultSurface
	}
	tx := int(px / float64(w.TileW))
	ty := int(py / float64(w.TileH))
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return tile.DefaultSurface
	}
	idx := ty*w.MapW + tx
	if len(w.Layers) > 0 {
		for k := len(w.Layers) - 1; k >= 0; k-- {
			if idx < len(w.Layers[k]) {
				gid := w.Layers[k][idx]
				if gid != tile.GIDEmpty {
					if w.DestroyedTiles[idx] {
						if def := tile.DefOf(gid); def.Destroyable() || def.OpenableByKey {
							continue
						}
					}
					return tile.SurfaceForGID(gid)
				}
			}
		}
	}
	gid := w.gidAt(tx, ty)
	return tile.SurfaceForGID(gid)
}

func (w *World) tileIndex(tx, ty int) int {
	return ty*w.MapW + tx
}

// PlayerRect returns the player's AABB.
func (w *World) PlayerRect() geom.Rect { return w.Player.Rect() }

func (w *World) MaxHP() int {
	return w.Stats.MaxHP()
}

func (w *World) MaxStamina() int {
	return w.Stats.MaxStamina()
}

func (w *World) SwordDamage() int {
	return w.Stats.SwordDamage()
}

const CycleLength = 14400

func (w *World) IsNight() bool {
	dn := w.EffectiveBalance().DayNight
	return w.TimeOfDay < dn.NightEndTick || w.TimeOfDay >= dn.NightStartTick
}

func (w *World) LightMultiplier() float64 {
	if w.HasAmbientLightOverride {
		return w.AmbientLightOverride
	}
	dn := w.EffectiveBalance().DayNight
	t := float64(w.TimeOfDay)
	if t >= 0 && t < float64(dn.NightEndTick) {
		return dn.MinAmbientLight + (t/float64(dn.NightEndTick))*(1.0-dn.MinAmbientLight)
	}
	if t >= float64(dn.NightEndTick) && t < float64(dn.DuskStartTick) {
		return 1.0
	}
	duskDuration := float64(dn.DuskEndTick - dn.DuskStartTick)
	if duskDuration > 0 && t >= float64(dn.DuskStartTick) && t < float64(dn.DuskEndTick) {
		return 1.0 - ((t-float64(dn.DuskStartTick))/duskDuration)*(1.0-dn.MinAmbientLight)
	}
	return dn.MinAmbientLight
}
