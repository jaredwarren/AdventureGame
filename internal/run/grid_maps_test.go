package run

import (
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/assets"
	"github.com/jaredwarren/game-test/internal/services"
)

type fsAssetCache struct{}

func (f *fsAssetCache) Atlas(id services.AtlasID) (services.Atlas, error) { return nil, nil }
func (f *fsAssetCache) MapData(id string) ([]byte, error) {
	name := "maps/" + id
	if !strings.HasSuffix(name, ".tmj") {
		name += ".tmj"
	}
	return assets.MapFS.ReadFile(name)
}

func TestLoadGridMaps(t *testing.T) {
	ac := &fsAssetCache{}
	sess := NewSession()
	for _, id := range []string{"F-5", "E-5", "F-4", "F-6", "G-5", "A-1", "J-10"} {
		if err := EnterMap(ac, sess, id, MapLoadOpts{}); err != nil {
			t.Fatalf("EnterMap(%q): %v", id, err)
		}
		if sess.World == nil {
			t.Fatalf("EnterMap(%q): nil world", id)
		}
	}
}

func TestAuthoredGridPlacement(t *testing.T) {
	ac := &fsAssetCache{}
	sess := NewSession()
	if err := EnterMap(ac, sess, "F-5", MapLoadOpts{}); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"F-4": false, "F-6": false, "E-5": false, "G-5": false, "dungeon": false}
	for _, d := range sess.World.Doors {
		if _, ok := want[d.TargetMap]; ok {
			want[d.TargetMap] = true
		}
	}
	for target, found := range want {
		if !found {
			t.Errorf("F-5 missing door to %s", target)
		}
	}

	if err := EnterMap(ac, sess, "F-6", MapLoadOpts{}); err != nil {
		t.Fatal(err)
	}
	if len(sess.World.Shrines) == 0 {
		t.Error("F-6 should keep field2 shrine")
	}

	if err := EnterMap(ac, sess, "G-5", MapLoadOpts{}); err != nil {
		t.Fatal(err)
	}
	hasBoss := false
	hasStart := false
	for _, d := range sess.World.Doors {
		if d.TargetMap == "knight_boss" {
			hasBoss = true
		}
		if d.TargetMap == "F-5" {
			hasStart = true
		}
	}
	if !hasBoss {
		t.Error("G-5 should keep maze2 door to knight_boss")
	}
	if !hasStart {
		t.Error("G-5 should have north door back to F-5")
	}
}
