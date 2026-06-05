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
