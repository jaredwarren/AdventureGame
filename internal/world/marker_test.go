package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func TestMarkerTypeNames(t *testing.T) {
	names := MarkerTypeNames()
	if len(names) < 3 {
		t.Fatalf("expected >=3 marker types, got %v", names)
	}
}

func TestMarkerHandlerFor_SpawnPickup(t *testing.T) {
	t.Parallel()
	w := &World{
		MapW: 4, MapH: 4, TileW: tile.Size, TileH: tile.Size,
		Tiles:  make([]int, 16),
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
