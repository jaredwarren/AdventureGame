package scenes

import (
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

type mockAssetCache struct {
	mapJSON []byte
}

func (m *mockAssetCache) Atlas(id services.AtlasID) (services.Atlas, error) {
	return nil, nil
}

func (m *mockAssetCache) MapData(id string) ([]byte, error) {
	return m.mapJSON, nil
}

func tinyMapJSON() []byte {
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
	b.WriteString(`]},{"name":"markers","type":"objectgroup","objects":[`)
	b.WriteString(`{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0}`)
	b.WriteString(`]}]}`)
	return []byte(b.String())
}

func TestTimeOfDayPersistence(t *testing.T) {
	assets := &mockAssetCache{mapJSON: tinyMapJSON()}
	sess := NewSession()

	// Initial load: TimeOfDay should be 0 by default.
	err := EnterMap(assets, sess, "field1", MapLoadOpts{})
	if err != nil {
		t.Fatalf("EnterMap failed: %v", err)
	}
	if sess.World.TimeOfDay != 0 {
		t.Errorf("expected initial TimeOfDay to be 0, got %d", sess.World.TimeOfDay)
	}

	// Change TimeOfDay
	sess.World.TimeOfDay = 4500

	// 1. Transition maps via EnterMap with CarryStatsFromSession: true
	err = EnterMap(assets, sess, "field2", MapLoadOpts{CarryStatsFromSession: true})
	if err != nil {
		t.Fatalf("EnterMap carry stats failed: %v", err)
	}
	if sess.World.TimeOfDay != 4500 {
		t.Errorf("expected carried TimeOfDay to be 4500, got %d", sess.World.TimeOfDay)
	}

	// 2. BuildSave and ApplySave
	cam := render.NewCamera(320, 240)
	saveObj := BuildSave(sess, cam)
	if saveObj.TimeOfDay != 4500 {
		t.Errorf("expected saved TimeOfDay to be 4500, got %d", saveObj.TimeOfDay)
	}

	// Reload with ApplySave
	sess.World.TimeOfDay = 100
	ApplySave(sess, saveObj, cam)
	if sess.World.TimeOfDay != 4500 {
		t.Errorf("expected loaded TimeOfDay to be 4500, got %d", sess.World.TimeOfDay)
	}

	// 3. Test WarpDoor
	sess.World.TimeOfDay = 6789
	// Add a target door in the world
	sess.World.Doors = []world.Door{
		{
			Rect:      geom.Rect{X: 0, Y: 0, W: 16, H: 16},
			TargetMap: "field3",
			SpawnX:    16,
			SpawnY:    16,
		},
	}
	// Call WarpDoor
	door := &sess.World.Doors[0]
	err = WarpDoor(assets, sess, cam, door)
	if err != nil {
		t.Fatalf("WarpDoor failed: %v", err)
	}
	if sess.World.TimeOfDay != 6789 {
		t.Errorf("expected TimeOfDay to persist through WarpDoor, got %d", sess.World.TimeOfDay)
	}
}

func tinyMapWithPropertiesJSON(properties string) []byte {
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
	b.WriteString(`]},{"name":"markers","type":"objectgroup","objects":[`)
	b.WriteString(`{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0}`)
	b.WriteString(`]}]`)
	if properties != "" {
		b.WriteString(`, "properties": `)
		b.WriteString(properties)
	}
	b.WriteString(`}`)
	return []byte(b.String())
}

func TestAmbientLightOverride(t *testing.T) {
	props := `[{"name": "light_level", "type": "float", "value": 0.45}]`
	assets := &mockAssetCache{mapJSON: tinyMapWithPropertiesJSON(props)}
	sess := NewSession()

	err := EnterMap(assets, sess, "dungeon", MapLoadOpts{})
	if err != nil {
		t.Fatalf("EnterMap failed: %v", err)
	}
	if !sess.World.HasAmbientLightOverride {
		t.Errorf("expected HasAmbientLightOverride to be true")
	}
	if sess.World.AmbientLightOverride != 0.45 {
		t.Errorf("expected AmbientLightOverride to be 0.45, got %f", sess.World.AmbientLightOverride)
	}
	if mult := sess.World.LightMultiplier(); mult != 0.45 {
		t.Errorf("expected LightMultiplier to be 0.45, got %f", mult)
	}
}

