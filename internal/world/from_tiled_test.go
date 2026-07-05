package world

import (
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/tile"
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
	if len(w.Pickups) != 1 {
		t.Fatalf("expected 1 persistent pickup to be spawned, got %d", len(w.Pickups))
	}
	if !w.Pickups[0].Opened {
		t.Errorf("expected persistent pickup to be Opened = true")
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

	if w.Tiles[5] != tile.GIDWaterShoreNW {
		t.Errorf("w.Tiles[5] = %d, want tile.GIDWaterShoreNW (%d)", w.Tiles[5], tile.GIDWaterShoreNW)
	}
	if w.Tiles[6] != tile.GIDWaterShoreNE {
		t.Errorf("w.Tiles[6] = %d, want tile.GIDWaterShoreNE (%d)", w.Tiles[6], tile.GIDWaterShoreNE)
	}
	if w.Tiles[9] != tile.GIDWaterShoreSW {
		t.Errorf("w.Tiles[9] = %d, want tile.GIDWaterShoreSW (%d)", w.Tiles[9], tile.GIDWaterShoreSW)
	}
	if w.Tiles[10] != tile.GIDWaterShoreSE {
		t.Errorf("w.Tiles[10] = %d, want tile.GIDWaterShoreSE (%d)", w.Tiles[10], tile.GIDWaterShoreSE)
	}
}

func TestAutoTileWall(t *testing.T) {
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
					2, 2, 2, 2, // Row 0
					2, 1, 1, 2, // Row 1
					2, 1, 1, 2, // Row 2
					2, 2, 2, 2, // Row 3
				},
			},
		},
	}

	w, err := BuildFromTiled(m, "testwallmap", progression.DefaultStats(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if w.Tiles[1] != tile.GIDWallBottom {
		t.Errorf("w.Tiles[1] = %d, want tile.GIDWallBottom (%d)", w.Tiles[1], tile.GIDWallBottom)
	}
	if w.Tiles[4] != tile.GIDWallRight {
		t.Errorf("w.Tiles[4] = %d, want tile.GIDWallRight (%d)", w.Tiles[4], tile.GIDWallRight)
	}
	if w.Tiles[0] != tile.GIDWallSEInner {
		t.Errorf("w.Tiles[0] = %d, want tile.GIDWallSEInner (%d)", w.Tiles[0], tile.GIDWallSEInner)
	}
}

func TestMultiLayerMap(t *testing.T) {
	m := &tiled.Map{
		Width:      2,
		Height:     2,
		TileWidth:  16,
		TileHeight: 16,
		Layers: []tiled.Layer{
			{
				Name: "base",
				Type: "tilelayer",
				Data: []int{
					tile.GIDDirtPath, tile.GIDGrass,
					tile.GIDGrass, tile.GIDDirtPath,
				},
			},
			{
				Name: "top",
				Type: "tilelayer",
				Data: []int{
					tile.GIDLock, tile.GIDEmpty,
					tile.GIDCracked, tile.GIDWall,
				},
			},
		},
	}

	w, err := BuildFromTiled(m, "multilayermap", progression.DefaultStats(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(w.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(w.Layers))
	}

	// Lock door at (0,0) over DirtPath: before destruction, gidAt returns GIDLock
	if gid := w.gidAt(0, 0); gid != tile.GIDLock {
		t.Errorf("gidAt(0,0) = %d, want GIDLock (%d)", gid, tile.GIDLock)
	}

	// Destroy lock door at (0,0): after destruction, gidAt returns lower layer GID (GIDDirtPath)
	w.DestroyedTiles[0] = true
	if gid := w.gidAt(0, 0); gid != tile.GIDDirtPath {
		t.Errorf("after destruction gidAt(0,0) = %d, want GIDDirtPath (%d)", gid, tile.GIDDirtPath)
	}
}
