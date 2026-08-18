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
	for _, id := range []string{"field1", "E-5", "F-4", "F-6", "G-5", "A-1", "J-10"} {
		if err := EnterMap(ac, sess, id, MapLoadOpts{}); err != nil {
			t.Fatalf("EnterMap(%q): %v", id, err)
		}
		if sess.World == nil {
			t.Fatalf("EnterMap(%q): nil world", id)
		}
	}
}

