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
	if w.TileW != TileSize || w.TileH != TileSize {
		return nil, fmt.Errorf("expected %dx%d tiles", TileSize, TileSize)
	}
	w.Player = Player{
		ID:        w.allocID(),
		Transform: Transform{},
		Hitbox:    Hitbox{W: DefaultPlayerHitboxW, H: DefaultPlayerHitboxH},
		Facing:    Facing{Dir: DirDown},
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
			if saveKey != "" && collectedPersistent != nil {
				if _, skip := collectedPersistent[saveKey]; skip {
					break
				}
			}
			w.SpawnPickup(o.X, o.Y-defaultPickupHitbox, k, saveKey)
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
				ID:   w.allocID(),
				Rect: geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
			})
		}
	}

	if !spawned {
		w.Player.X = float64(m.TileWidth * 2)
		w.Player.Y = float64(m.TileHeight * 2)
	}

	return w, nil
}
