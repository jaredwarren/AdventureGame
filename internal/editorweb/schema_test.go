package editorweb

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func testSchema(t *testing.T) *Schema {
	t.Helper()
	s, err := BuildSchema(animOpts{})
	if err != nil {
		t.Fatalf("BuildSchema: %v", err)
	}
	return s
}

// TestSchemaCoversAllRegisteredGIDs is the drift guard between the Go tile
// registry and whatever the browser draws.
func TestSchemaCoversAllRegisteredGIDs(t *testing.T) {
	s := testSchema(t)

	seen := map[int]bool{}
	for _, art := range s.Tiles {
		if seen[art.GID] {
			t.Errorf("gid %d appears twice in the schema", art.GID)
		}
		seen[art.GID] = true
	}
	for _, gid := range tile.RegisteredGIDs() {
		if !seen[gid] {
			t.Errorf("gid %d (%s) is registered but missing from the schema", gid, tile.DefOf(gid).Name)
		}
		delete(seen, gid)
	}
	for gid := range seen {
		t.Errorf("gid %d is in the schema but not registered", gid)
	}
}

// TestPaletteCoversAllRegisteredGIDs catches the "added GIDSand, forgot the
// palette" failure that the in-game editor's hand-maintained brush list has
// today: a tile you cannot select is a tile you cannot place.
func TestPaletteCoversAllRegisteredGIDs(t *testing.T) {
	s := testSchema(t)

	group := map[int]string{}
	for _, g := range s.Palette {
		for _, gid := range g.GIDs {
			if prev, dup := group[gid]; dup {
				t.Errorf("gid %d is in both %q and %q; palette rules must be mutually exclusive", gid, prev, g.ID)
			}
			group[gid] = g.ID
			if tile.DefOf(gid).Name == "unknown" {
				t.Errorf("palette group %q lists unregistered gid %d", g.ID, gid)
			}
		}
	}
	for _, gid := range tile.RegisteredGIDs() {
		if _, ok := group[gid]; !ok {
			t.Errorf("gid %d (%s) is registered but reachable from no palette group", gid, tile.DefOf(gid).Name)
		}
	}
}

// TestPaletteGroupsAreStable documents the current grouping so an accidental
// rule change is visible in review.
func TestPaletteGroupsAreStable(t *testing.T) {
	s := testSchema(t)
	want := map[string]int{
		"terrain":     4,
		"dirt_path":   13,
		"cobble_path": 13,
		"hazards":     4,
		"structures":  5,
		"water":       13,
		"wall":        13,
		"rock":        13,
	}
	got := map[string]int{}
	for _, g := range s.Palette {
		got[g.ID] = len(g.GIDs)
	}
	for id, n := range want {
		if got[id] != n {
			t.Errorf("palette group %q has %d tiles, want %d", id, got[id], n)
		}
	}
	if len(got) != len(want) {
		t.Errorf("palette has %d groups, want %d: %v", len(got), len(want), got)
	}
}

// TestFavoritesAreRegistered keeps the 1-9,0 hotkey slots pointing at real tiles.
func TestFavoritesAreRegistered(t *testing.T) {
	s := testSchema(t)
	if len(s.Favorites) != 10 {
		t.Errorf("%d favorite slots, want 10 (keys 1-9 and 0)", len(s.Favorites))
	}
	for i, gid := range s.Favorites {
		if tile.DefOf(gid).Name == "unknown" {
			t.Errorf("favorite slot %d points at unregistered gid %d", i, gid)
		}
	}
	if last := s.Favorites[len(s.Favorites)-1]; last != tile.GIDEmpty {
		t.Errorf("slot 0 is gid %d, want GIDEmpty so it erases like the in-game editor", last)
	}
}

// TestSchemaMarkersMatchRegistry keeps the marker list and its order in step
// with world.MarkerTypeNames.
func TestSchemaMarkersMatchRegistry(t *testing.T) {
	s := testSchema(t)
	want := world.MarkerTypeNames()
	if len(s.Markers) != len(want) {
		t.Fatalf("%d marker schemas, registry has %d", len(s.Markers), len(want))
	}
	for i, ms := range s.Markers {
		if ms.Type != want[i] {
			t.Errorf("marker %d is %q, registry has %q", i, ms.Type, want[i])
		}
	}
}

// TestMapPropertiesMatchLoader pins the map-level property list against what
// world.BuildFromTiled actually reads. These cannot be probed because nothing
// writes them by default.
func TestMapPropertiesMatchLoader(t *testing.T) {
	s := testSchema(t)
	// Read by BuildFromTiled: light_level is preferred, ambient_light is the
	// legacy fallback used only when light_level is absent.
	want := []string{"light_level", "ambient_light"}

	var got []string
	for _, f := range s.MapProperties {
		got = append(got, f.Name)
		if f.TiledType != "float" {
			t.Errorf("map property %q has tiledType %q, want float", f.Name, f.TiledType)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("map properties %v, want %v", got, want)
	}
}

// TestSchemaConstantsMatchWorld makes sure the hitbox sizes the client draws
// marker boxes with come from the game, not from a stale copy.
func TestSchemaConstantsMatchWorld(t *testing.T) {
	s := testSchema(t)
	c := s.Constants

	// Cross-check against the real hit rects rather than the constants directly,
	// so this fails if either the constant or the rect formula moves.
	spawnRect := world.MarkerObjectHitRect(tiled.Object{Type: "spawn"})
	if c.PlayerHitbox.W != spawnRect.W || c.PlayerHitbox.H != spawnRect.H {
		t.Errorf("playerHitbox %+v != spawn marker hit rect %vx%v", c.PlayerHitbox, spawnRect.W, spawnRect.H)
	}
	enemyRect := world.MarkerObjectHitRect(tiled.Object{Type: "enemy"})
	if c.EnemyHitbox.W != enemyRect.W || c.EnemyHitbox.H != enemyRect.H {
		t.Errorf("enemyHitbox %+v != enemy marker hit rect %vx%v", c.EnemyHitbox, enemyRect.W, enemyRect.H)
	}
	pickupRect := world.MarkerObjectHitRect(tiled.Object{Type: "pickup"})
	if c.PickupHitbox.W != pickupRect.W || c.PickupHitbox.H != pickupRect.H {
		t.Errorf("pickupHitbox %+v != pickup marker hit rect %vx%v", c.PickupHitbox, pickupRect.W, pickupRect.H)
	}
	if s.TileSize != tile.Size {
		t.Errorf("tileSize %d != tile.Size %d", s.TileSize, tile.Size)
	}
	if s.Layers.DefaultBaseGID != tile.GIDGrass {
		t.Errorf("defaultBaseGid %d != GIDGrass %d", s.Layers.DefaultBaseGID, tile.GIDGrass)
	}
}

// TestSchemaJSONIsStable guards ETag caching: the payload must be byte-identical
// across builds, so no map[...] may leak into it.
func TestSchemaJSONIsStable(t *testing.T) {
	a, err := newCachedSchema(animOpts{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := newCachedSchema(animOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if a.etag != b.etag {
		t.Error("schema etag differs between builds; something nondeterministic is in the payload")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(a.body, &probe); err != nil {
		t.Fatalf("schema is not a JSON object: %v", err)
	}
	for _, required := range []string{"schemaVersion", "tileSize", "colors", "tiles", "palette", "markers", "pickups"} {
		if _, ok := probe[required]; !ok {
			t.Errorf("schema is missing required key %q", required)
		}
	}
}
