package world

import (
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
)

func tinyMapJSON(markerObjects string) string {
	const w, h = 4, 4
	var b strings.Builder
	b.WriteString(`{"width":4,"height":4,"tilewidth":16,"tileheight":16,"layers":[`)
	b.WriteString(`{"name":"ground","type":"tilelayer","data":[`)
	for i := 0; i < w*h; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("1")
	}
	b.WriteString(`]},{"name":"markers","type":"objectgroup","objects":`)
	b.WriteString(markerObjects)
	b.WriteString(`}]}`)
	return b.String()
}

func TestBuildFromTiled_SkipsPersistentPickupWhenCollected(t *testing.T) {
	t.Parallel()
	objs := `[{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0},
	{"id":7,"type":"pickup","x":48,"y":48,"width":0,"height":0,"properties":[
		{"name":"kind","type":"string","value":"coin"},
		{"name":"persistent","type":"bool","value":true}
	]}]`
	m, err := tiled.ParseMap([]byte(tinyMapJSON(objs)))
	if err != nil {
		t.Fatal(err)
	}
	key := PersistentPickupSaveKey("testmap", 7)
	collected := map[string]struct{}{key: {}}
	w, err := BuildFromTiled(m, "testmap", progression.DefaultStats(), collected)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Pickups) != 0 {
		t.Fatalf("expected persistent pickup skipped, got %d pickups", len(w.Pickups))
	}
}

func TestBuildFromTiled_SpawnsPersistentPickupWhenNotCollected(t *testing.T) {
	t.Parallel()
	objs := `[{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0},
	{"id":7,"type":"pickup","x":48,"y":48,"width":0,"height":0,"properties":[
		{"name":"kind","type":"string","value":"coin"},
		{"name":"persistent","type":"bool","value":true}
	]}]`
	m, err := tiled.ParseMap([]byte(tinyMapJSON(objs)))
	if err != nil {
		t.Fatal(err)
	}
	w, err := BuildFromTiled(m, "testmap", progression.DefaultStats(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Pickups) != 1 {
		t.Fatalf("expected 1 pickup, got %d", len(w.Pickups))
	}
	want := PersistentPickupSaveKey("testmap", 7)
	if w.Pickups[0].PersistentSaveKey != want {
		t.Errorf("PersistentSaveKey = %q want %q", w.Pickups[0].PersistentSaveKey, want)
	}
}

func TestBuildFromTiled_NonPersistentPickupIgnoresCollectedSet(t *testing.T) {
	t.Parallel()
	objs := `[{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0},
	{"id":9,"type":"pickup","x":48,"y":48,"width":0,"height":0,"properties":[
		{"name":"kind","type":"string","value":"coin"}
	]}]`
	m, err := tiled.ParseMap([]byte(tinyMapJSON(objs)))
	if err != nil {
		t.Fatal(err)
	}
	collected := map[string]struct{}{PersistentPickupSaveKey("testmap", 9): {}}
	w, err := BuildFromTiled(m, "testmap", progression.DefaultStats(), collected)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Pickups) != 1 {
		t.Fatalf("expected non-persistent pickup despite key in set, got %d", len(w.Pickups))
	}
	if w.Pickups[0].PersistentSaveKey != "" {
		t.Errorf("expected empty PersistentSaveKey for non-persistent pickup")
	}
}

func TestAutoTileWater(t *testing.T) {
	// Create a custom 4x4 map with some grass (1) and water (5)
	// Grid structure:
	// Grass Grass Grass Grass
	// Grass Water Water Grass
	// Grass Grass Grass Grass
	// Grass Grass Grass Grass
	//
	// The water tile at (1, 1) has grass on the top, left, bottom, but wait:
	// Let's design a custom grid:
	// Grass, Grass, Grass, Grass
	// Grass, Water, Water, Water
	// Grass, Water, Water, Water
	// Grass, Grass, Grass, Grass
	
	m := &tiled.Map{
		Width:      4,
		Height:     4,
		TileWidth:  16,
		TileHeight: 16,
		Layers: []tiled.Layer{
			{
				Name: "ground",
				Type: "tilelayer",
				Data: []int{
					1, 1, 1, 1, // Row 0
					1, 5, 5, 1, // Row 1
					1, 5, 5, 1, // Row 2
					1, 1, 1, 1, // Row 3
				},
			},
		},
	}

	w, err := BuildFromTiled(m, "testmap", progression.DefaultStats(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify coordinates:
	// Row 1, Col 1 (index 5): Water tile surrounded by grass on top (1,0) and left (0,1).
	// NW Corner -> GIDWaterShoreNW (14)
	if w.Tiles[5] != GIDWaterShoreNW {
		t.Errorf("w.Tiles[5] = %d, want GIDWaterShoreNW (%d)", w.Tiles[5], GIDWaterShoreNW)
	}

	// Row 1, Col 2 (index 6): Water tile surrounded by grass on top (2,0) and right (3,1).
	// NE Corner -> GIDWaterShoreNE (13)
	if w.Tiles[6] != GIDWaterShoreNE {
		t.Errorf("w.Tiles[6] = %d, want GIDWaterShoreNE (%d)", w.Tiles[6], GIDWaterShoreNE)
	}

	// Row 2, Col 1 (index 9): Water tile surrounded by grass on bottom (1,3) and left (0,2).
	// SW Corner -> GIDWaterShoreSW (15)
	if w.Tiles[9] != GIDWaterShoreSW {
		t.Errorf("w.Tiles[9] = %d, want GIDWaterShoreSW (%d)", w.Tiles[9], GIDWaterShoreSW)
	}

	// Row 2, Col 2 (index 10): Water tile surrounded by grass on bottom (2,3) and right (3,2).
	// SE Corner -> GIDWaterShoreSE (16)
	if w.Tiles[10] != GIDWaterShoreSE {
		t.Errorf("w.Tiles[10] = %d, want GIDWaterShoreSE (%d)", w.Tiles[10], GIDWaterShoreSE)
	}
}

