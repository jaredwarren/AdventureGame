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
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// BuildFromTiled clones tile GIDs into memory and instantiates entities from the markers layer.
// collectedPersistent lists PersistentPickupSaveKey values already taken this run (from save);
// when non-nil, pickups with matching keys are not spawned. Pass nil if nothing was collected yet.
func BuildFromTiled(m *tiled.Map, mapID string, stats progression.Stats, collectedPersistent map[string]struct{}) (*World, error) {
	var layers [][]int
	for i := range m.Layers {
		if m.Layers[i].Type == "tilelayer" && len(m.Layers[i].Data) == m.Width*m.Height {
			layers = append(layers, append([]int(nil), m.Layers[i].Data...))
		}
	}
	if len(layers) == 0 {
		tl := m.TileLayer("ground")
		if tl == nil || len(tl.Data) != m.Width*m.Height {
			return nil, fmt.Errorf("map %s: missing ground layer or bad data len", mapID)
		}
		layers = append(layers, append([]int(nil), tl.Data...))
	}
	if len(layers) == 1 {
		baseLayer := make([]int, m.Width*m.Height)
		for i := range baseLayer {
			baseLayer[i] = tile.GIDGrass
		}
		layers = [][]int{baseLayer, layers[0]}
	}

	topLayer := layers[len(layers)-1]

	w := &World{
		MapID:             mapID,
		TileW:             m.TileWidth,
		TileH:             m.TileHeight,
		MapW:              m.Width,
		MapH:              m.Height,
		Layers:            layers,
		Tiles:             topLayer,
		ActiveLayerFilter: -1,
		DestroyedTiles:    make(map[int]bool),
		Stats:             stats,
		HP:                stats.MaxHP(),
	}
	if val, ok := tiled.MapPropFloat(m, "light_level"); ok {
		w.HasAmbientLightOverride = true
		w.AmbientLightOverride = val
	} else if val, ok := tiled.MapPropFloat(m, "ambient_light"); ok {
		w.HasAmbientLightOverride = true
		w.AmbientLightOverride = val
	}
	if w.TileW != tile.Size || w.TileH != tile.Size {
		return nil, fmt.Errorf("expected %dx%d tiles", tile.Size, tile.Size)
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
