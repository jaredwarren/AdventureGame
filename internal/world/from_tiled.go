// Tiled → World conversion (marker conventions documented in BuildFromTiled).
//
// Pickup objects may set property persistent=true (bool) so the pickup is
// tracked by PersistentPickupSaveKey(mapID, object id) across saves; see
// scenes.MapProgress per mapID / save.GameSave.CollectedPickupKeys.
//
// Spawn markers and doors (default) use feet-on-ground Y: see PlayerTopLeftFromDoorSpawn and
// optional door property spawn_anchor ("feet" default, "topleft" for hitbox top-left).
// TODO: support embedded/external tilesets if you stop using flat GIDs only.
package world

import (
	"fmt"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
)

// BuildFromTiled clones tile GIDs into memory and instantiates entities from the markers layer.
// collectedPersistent lists PersistentPickupSaveKey values already taken this run (from save);
// when non-nil, pickups with matching keys are not spawned. Pass nil if nothing was collected yet.
func BuildFromTiled(m *tiled.Map, mapID string, stats progression.Stats, collectedPersistent map[string]struct{}) (*World, error) {
	tl := m.TileLayer("ground")
	if tl == nil || len(tl.Data) != m.Width*m.Height {
		return nil, fmt.Errorf("map %s: missing ground layer or bad data len", mapID)
	}
	w := &World{
		MapID:          mapID,
		TileW:          m.TileWidth,
		TileH:          m.TileHeight,
		MapW:           m.Width,
		MapH:           m.Height,
		Tiles:          append([]int(nil), tl.Data...),
		DestroyedTiles: make(map[int]bool),
		Stats:          stats,
		HP:             stats.MaxHP(),
	}
	autoTileWater(w)
	if val, ok := tiled.MapPropFloat(m, "light_level"); ok {
		w.HasAmbientLightOverride = true
		w.AmbientLightOverride = val
	} else if val, ok := tiled.MapPropFloat(m, "ambient_light"); ok {
		w.HasAmbientLightOverride = true
		w.AmbientLightOverride = val
	}
	if w.TileW != TileSize || w.TileH != TileSize {
		return nil, fmt.Errorf("expected %dx%d tiles", TileSize, TileSize)
	}
	w.Player = DefaultPlayerTuning()
	w.Player.ID = w.allocID()
	w.Player.Hitbox = Hitbox{W: DefaultPlayerHitboxW, H: DefaultPlayerHitboxH}
	w.Player.Facing = Facing{Dir: DirDown}
	w.Player.Stamina = w.MaxStamina()

	spawned := false
	spawnCtx := MarkerSpawnContext{
		CollectedPersistent: collectedPersistent,
		Spawned:             &spawned,
	}
	for _, o := range m.ObjectGroup("markers") {
		if h := MarkerHandlerFor(o.Type); h != nil {
			h.SpawnFromTiled(w, &o, mapID, spawnCtx)
		}
	}

	if !spawned {
		w.Player.X = float64(m.TileWidth * 2)
		w.Player.Y = float64(m.TileHeight * 2)
	}

	return w, nil
}

func autoTileWater(w *World) {
	originalTiles := append([]int(nil), w.Tiles...)

	isLand := func(tx, ty int) bool {
		if tx < 0 || tx >= w.MapW || ty < 0 || ty >= w.MapH {
			return true // out of bounds is treated as land/grass
		}
		gid := originalTiles[ty*w.MapW+tx]
		return gid != GIDWater && (gid < GIDWaterShoreTop || gid > GIDWaterShoreSEInner)
	}

	for ty := 0; ty < w.MapH; ty++ {
		for tx := 0; tx < w.MapW; tx++ {
			idx := ty*w.MapW + tx
			if originalTiles[idx] != GIDWater {
				continue
			}

			top := isLand(tx, ty-1)
			bottom := isLand(tx, ty+1)
			left := isLand(tx-1, ty)
			right := isLand(tx+1, ty)

			tl := isLand(tx-1, ty-1)
			tr := isLand(tx+1, ty-1)
			bl := isLand(tx-1, ty+1)
			br := isLand(tx+1, ty+1)

			// 1. Convex (outer) corners: Grass on two adjacent sides
			if top && left {
				w.Tiles[idx] = GIDWaterShoreNW
			} else if top && right {
				w.Tiles[idx] = GIDWaterShoreNE
			} else if bottom && left {
				w.Tiles[idx] = GIDWaterShoreSW
			} else if bottom && right {
				w.Tiles[idx] = GIDWaterShoreSE
				// 2. Straight shores: Grass on one side
			} else if top {
				w.Tiles[idx] = GIDWaterShoreTop
			} else if bottom {
				w.Tiles[idx] = GIDWaterShoreBottom
			} else if left {
				w.Tiles[idx] = GIDWaterShoreLeft
			} else if right {
				w.Tiles[idx] = GIDWaterShoreRight
				// 3. Concave (inner) corners: Grass diagonally only
			} else if tl {
				w.Tiles[idx] = GIDWaterShoreNWInner
			} else if tr {
				w.Tiles[idx] = GIDWaterShoreNEInner
			} else if bl {
				w.Tiles[idx] = GIDWaterShoreSWInner
			} else if br {
				w.Tiles[idx] = GIDWaterShoreSEInner
			}
		}
	}
}
