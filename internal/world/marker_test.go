package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
)

func TestMarkerTypeNames(t *testing.T) {
	t.Parallel()
	types := MarkerTypeNames()
	if len(types) != 5 {
		t.Fatalf("expected 5 marker types, got %v", types)
	}
}

func TestMarkerHandlerFor_SpawnPickup(t *testing.T) {
	t.Parallel()
	w := &World{
		MapW: 4, MapH: 4, TileW: TileSize, TileH: TileSize,
		Tiles: make([]int, 16),
		Player: DefaultPlayerTuning(),
	}
	o := &tiled.Object{Type: "pickup", X: 16, Y: 28, Properties: []tiled.Property{
		{Name: "kind", Type: "string", Value: "bomb"},
	}}
	MarkerHandlerFor("pickup").SpawnFromTiled(w, o, "test", MarkerSpawnContext{})
	if len(w.Pickups) != 1 || w.Pickups[0].Kind != PickupBomb {
		t.Fatalf("pickup spawn: %+v", w.Pickups)
	}
}
