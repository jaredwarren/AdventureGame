package scenes

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/world"
)

func TestMapIDFromPersistKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		key, want string
	}{
		{"field1:2:3", "field1"},
		{"my:map:4:5", "my:map"},
		{"field1:42", "field1"},
		{"dungeon:shrine:7", "dungeon"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := mapIDFromPersistKey(tc.key); got != tc.want {
			t.Errorf("mapIDFromPersistKey(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestProgressMapsFromSave_GroupsByMap(t *testing.T) {
	t.Parallel()
	s := &save.GameSave{
		CollectedPickupKeys: []string{"field1:10", "dungeon:3"},
		DestroyedTileKeys:   []string{"field1:1:2"},
		OpenedLockTileKeys:  []string{"dungeon:0:0"},
		ActivatedShrines:    []string{"field1:shrine:1"},
	}
	maps := progressMapsFromSave(s)
	if len(maps) != 2 {
		t.Fatalf("expected 2 maps, got %d", len(maps))
	}
	if _, ok := maps["field1"].CollectedPickups["field1:10"]; !ok {
		t.Error("missing field1 pickup")
	}
	if _, ok := maps["field1"].DestroyedTiles["field1:1:2"]; !ok {
		t.Error("missing field1 destroyed tile")
	}
	if _, ok := maps["dungeon"].OpenedLocks["dungeon:0:0"]; !ok {
		t.Error("missing dungeon lock")
	}
}

func TestMapProgress_PerMapIsolation(t *testing.T) {
	assets := &mockAssetCache{mapJSON: tinyMapJSON()}
	sess := NewSession()

	if err := EnterMap(assets, sess, "field1", MapLoadOpts{}); err != nil {
		t.Fatal(err)
	}
	sess.MarkDestroyedTile("field1", world.MapTilePersistKey("field1", 1, 1))

	if err := EnterMap(assets, sess, "field2", MapLoadOpts{CarryStatsFromSession: true}); err != nil {
		t.Fatal(err)
	}
	if p := sess.ProgressFor("field2"); len(p.DestroyedTiles) != 0 {
		t.Errorf("field2 should have no destroyed tiles, got %v", p.DestroyedTiles)
	}

	if err := EnterMap(assets, sess, "field1", MapLoadOpts{CarryStatsFromSession: true}); err != nil {
		t.Fatal(err)
	}
	key := world.MapTilePersistKey("field1", 1, 1)
	if _, ok := sess.ProgressFor("field1").DestroyedTiles[key]; !ok {
		t.Error("field1 progress should persist after revisiting map")
	}
}

func TestFlattenMapsToSave_RoundTrip(t *testing.T) {
	t.Parallel()
	sess := NewSession()
	sess.ProgressFor("a").MarkDestroyedTile(world.MapTilePersistKey("a", 0, 0))
	sess.ProgressFor("b").MarkCollectedPickup("b:5")

	gs := &save.GameSave{}
	flattenMapsToSave(gs, sess.Maps)

	loaded := progressMapsFromSave(gs)
	if _, ok := loaded["a"].DestroyedTiles[world.MapTilePersistKey("a", 0, 0)]; !ok {
		t.Error("missing destroyed tile after round trip")
	}
	if _, ok := loaded["b"].CollectedPickups["b:5"]; !ok {
		t.Error("missing pickup after round trip")
	}
}
