// Tiled → World conversion (marker conventions documented in BuildFromTiled).
//
// Pickup objects may set property persistent=true (bool) so the pickup is
// tracked by PersistentPickupSaveKey(mapID, object id) across saves; see
// Session.CollectedPersistentPickups / save.GameSave.CollectedPickupKeys.
//
// Spawn markers and doors (default) use feet-on-ground Y: see PlayerTopLeftFromDoorSpawn and
// optional door property spawn_anchor ("feet" default, "topleft" for hitbox top-left).
// TODO: support embedded/external tilesets if you stop using flat GIDs only.
package world

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jaredwarren/game-test/internal/geom"
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
	w.Player = Player{
		ID:                    w.allocID(),
		Transform:             Transform{},
		Hitbox:                Hitbox{W: DefaultPlayerHitboxW, H: DefaultPlayerHitboxH},
		Facing:                Facing{Dir: DirDown},
		SwingDuration:         8,
		MaxSwingCD:            12,
		SwingActiveStart:      2,
		SwingActiveEnd:        7,
		TorchSwingDuration:    8,
		MaxTorchSwingCD:       12,
		TorchSwingActiveStart: 2,
		TorchSwingActiveEnd:   7,
		BaseSpeed:             1.35,
		SprintSpeed:           2.15,
		DodgeStaminaCost:      20,
		DodgeDuration:         20,
		DodgeMaxImpulse:       12,
		DodgeSpeed:            2.8,
		StaminaRegenInterval:       2,
		SwordReach:                 14.0,
		SwordThickness:             10.0,
		TorchReach:                 14.0,
		TorchThickness:             10.0,
		InvulnFrames:               45,
		EnemyKnockbackForce:        20.0,
		PlayerKnockbackForce:       6.0,
		PlayerHazardKnockbackForce: 12.0,
		MaxBombs:                   8,
		BombFuseDuration:           90,
		BombRadius:                 32.0,
		BombDamage:                 4,
		TorchBurnDuration:          72,
		TorchBurnInterval:          12,
		TorchBurnDamage:            1,
	}
	w.Player.Stamina = w.MaxStamina()

	spawned := false
	objs := m.ObjectGroup("markers")
	for i := range objs {
		o := &objs[i]
		switch o.Type {
		case "spawn":
			w.Player.X, w.Player.Y = PlayerTopLeftFromDoorSpawn(o.X, o.Y, DoorSpawnFeet, w.Player.H)
			spawned = true
		case "enemy":
			w.SpawnEnemy(o.X, o.Y-defaultEnemyH, 3, false)
		case "pickup":
			k := PickupCoin
			if s, ok := tiled.ObjProp(o, "kind"); ok {
				switch s {
				case "heart":
					k = PickupHeart
				case "bomb":
					k = PickupBomb
				case "key":
					k = PickupSmallKey
				case "torch":
					k = PickupTorch
				}
			}
			persist, _ := tiled.ObjPropBool(o, "persistent")
			var saveKey string
			if persist {
				if o.ID != 0 {
					saveKey = PersistentPickupSaveKey(mapID, o.ID)
				} else if o.Name != "" {
					saveKey = fmt.Sprintf("%s:name:%s", mapID, o.Name)
				}
			}
			var opened bool
			if saveKey != "" && collectedPersistent != nil {
				if _, skip := collectedPersistent[saveKey]; skip {
					opened = true
				}
			}
			id := w.SpawnPickup(o.X, o.Y-defaultPickupHitbox, k, saveKey)
			if opened {
				for i := range w.Pickups {
					if w.Pickups[i].ID == id {
						w.Pickups[i].Opened = true
					}
				}
			}
		case "door":
			tmap, _ := tiled.ObjProp(o, "target_map")
			sx, _ := tiled.ObjProp(o, "spawn_x")
			sy, _ := tiled.ObjProp(o, "spawn_y")
			fx, _ := strconv.ParseFloat(sx, 64)
			fy, _ := strconv.ParseFloat(sy, 64)
			style := DoorSpawnFeet
			if a, ok := tiled.ObjProp(o, "spawn_anchor"); ok {
				switch strings.ToLower(strings.TrimSpace(a)) {
				case "topleft", "top_left", "top-left", "origin":
					style = DoorSpawnTopLeft
				}
			}
			// NOTE: Door rects must lie mostly on walkable tiles. Placing them flush with the outer
			// wall (e.g. x=304 on a 320px-wide map) puts the trigger inside solid collision—unreachable.
			w.Doors = append(w.Doors, Door{
				ID:         w.allocID(),
				Rect:       geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
				TargetMap:  tmap,
				SpawnX:     fx,
				SpawnY:     fy,
				SpawnStyle: style,
			})
		case "shrine":
			w.Shrines = append(w.Shrines, Shrine{
				ID:      w.allocID(),
				TiledID: o.ID,
				Rect:    geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
			})
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

