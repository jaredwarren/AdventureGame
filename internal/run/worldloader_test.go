package run

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

func TestTimeAndItemPersistence(t *testing.T) {
	assets := &mockAssetCache{mapJSON: tinyMapJSON()}
	sess := NewSession()

	// Initial load: TimeOfDay should be 0 by default, SelectedItem should be ItemSlotBomb.
	err := EnterMap(assets, sess, "field1", MapLoadOpts{})
	if err != nil {
		t.Fatalf("EnterMap failed: %v", err)
	}
	if sess.World.TimeOfDay != 0 {
		t.Errorf("expected initial TimeOfDay to be 0, got %d", sess.World.TimeOfDay)
	}
	if sess.World.SelectedItem != world.ItemSlotBomb {
		t.Errorf("expected initial SelectedItem to be ItemSlotBomb, got %v", sess.World.SelectedItem)
	}

	// Change TimeOfDay, SelectedItem, and locomotion/stamina properties
	sess.World.TimeOfDay = 4500
	sess.World.SelectedItem = world.ItemSlotTorch
	sess.World.Player.BaseSpeed = 2.5
	sess.World.Player.SprintSpeed = 4.0
	sess.World.Player.DodgeStaminaCost = 15
	sess.World.Player.DodgeDuration = 25
	sess.World.Player.DodgeMaxImpulse = 15
	sess.World.Player.DodgeSpeed = 3.5
	sess.World.Player.StaminaRegenInterval = 3
	sess.World.Player.SwordReach = 20.0
	sess.World.Player.MaxBombs = 12
	sess.World.Player.BombFuseDuration = 70
	sess.World.Player.BombRadius = 45.0
	sess.World.Player.BombDamage = 8
	sess.World.Player.TorchBurnDuration = 100
	sess.World.Player.TorchBurnInterval = 8
	sess.World.Player.TorchBurnDamage = 3

	// 1. Transition maps via EnterMap with CarryStatsFromSession: true
	err = EnterMap(assets, sess, "field2", MapLoadOpts{CarryStatsFromSession: true})
	if err != nil {
		t.Fatalf("EnterMap carry stats failed: %v", err)
	}
	if sess.World.TimeOfDay != 4500 {
		t.Errorf("expected carried TimeOfDay to be 4500, got %d", sess.World.TimeOfDay)
	}
	if sess.World.SelectedItem != world.ItemSlotTorch {
		t.Errorf("expected carried SelectedItem to be ItemSlotTorch, got %v", sess.World.SelectedItem)
	}
	if sess.World.Player.BaseSpeed != 2.5 {
		t.Errorf("expected carried BaseSpeed to be 2.5, got %f", sess.World.Player.BaseSpeed)
	}
	if sess.World.Player.SprintSpeed != 4.0 {
		t.Errorf("expected carried SprintSpeed to be 4.0, got %f", sess.World.Player.SprintSpeed)
	}
	if sess.World.Player.DodgeStaminaCost != 15 {
		t.Errorf("expected carried DodgeStaminaCost to be 15, got %d", sess.World.Player.DodgeStaminaCost)
	}
	if sess.World.Player.DodgeDuration != 25 {
		t.Errorf("expected carried DodgeDuration to be 25, got %d", sess.World.Player.DodgeDuration)
	}
	if sess.World.Player.DodgeMaxImpulse != 15 {
		t.Errorf("expected carried DodgeMaxImpulse to be 15, got %d", sess.World.Player.DodgeMaxImpulse)
	}
	if sess.World.Player.DodgeSpeed != 3.5 {
		t.Errorf("expected carried DodgeSpeed to be 3.5, got %f", sess.World.Player.DodgeSpeed)
	}
	if sess.World.Player.StaminaRegenInterval != 3 {
		t.Errorf("expected carried StaminaRegenInterval to be 3, got %d", sess.World.Player.StaminaRegenInterval)
	}
	if sess.World.Player.SwordReach != 20.0 {
		t.Errorf("expected carried SwordReach to be 20.0, got %f", sess.World.Player.SwordReach)
	}
	if sess.World.Player.MaxBombs != 12 {
		t.Errorf("expected carried MaxBombs to be 12, got %d", sess.World.Player.MaxBombs)
	}
	if sess.World.Player.BombFuseDuration != 70 {
		t.Errorf("expected carried BombFuseDuration to be 70, got %d", sess.World.Player.BombFuseDuration)
	}
	if sess.World.Player.BombRadius != 45.0 {
		t.Errorf("expected carried BombRadius to be 45.0, got %f", sess.World.Player.BombRadius)
	}
	if sess.World.Player.BombDamage != 8 {
		t.Errorf("expected carried BombDamage to be 8, got %d", sess.World.Player.BombDamage)
	}
	if sess.World.Player.TorchBurnDuration != 100 {
		t.Errorf("expected carried TorchBurnDuration to be 100, got %d", sess.World.Player.TorchBurnDuration)
	}
	if sess.World.Player.TorchBurnInterval != 8 {
		t.Errorf("expected carried TorchBurnInterval to be 8, got %d", sess.World.Player.TorchBurnInterval)
	}
	if sess.World.Player.TorchBurnDamage != 3 {
		t.Errorf("expected carried TorchBurnDamage to be 3, got %d", sess.World.Player.TorchBurnDamage)
	}

	// 2. BuildSave and ApplySave
	cam := render.NewCamera(320, 240)
	saveObj := BuildSave(sess, cam)
	if saveObj.TimeOfDay != 4500 {
		t.Errorf("expected saved TimeOfDay to be 4500, got %d", saveObj.TimeOfDay)
	}
	if saveObj.SelectedItem != int(world.ItemSlotTorch) {
		t.Errorf("expected saved SelectedItem to be ItemSlotTorch, got %v", saveObj.SelectedItem)
	}
	if saveObj.Tuning.BaseSpeed != 2.5 {
		t.Errorf("expected saved BaseSpeed to be 2.5, got %f", saveObj.Tuning.BaseSpeed)
	}
	if saveObj.Tuning.SprintSpeed != 4.0 {
		t.Errorf("expected saved SprintSpeed to be 4.0, got %f", saveObj.Tuning.SprintSpeed)
	}
	if saveObj.Tuning.SwordReach != 20.0 {
		t.Errorf("expected saved SwordReach to be 20.0, got %f", saveObj.Tuning.SwordReach)
	}
	if saveObj.Tuning.MaxBombs != 12 {
		t.Errorf("expected saved MaxBombs to be 12, got %d", saveObj.Tuning.MaxBombs)
	}
	if saveObj.Tuning.BombFuseDuration != 70 {
		t.Errorf("expected saved BombFuseDuration to be 70, got %d", saveObj.Tuning.BombFuseDuration)
	}
	if saveObj.Tuning.BombRadius != 45.0 {
		t.Errorf("expected saved BombRadius to be 45.0, got %f", saveObj.Tuning.BombRadius)
	}
	if saveObj.Tuning.BombDamage != 8 {
		t.Errorf("expected saved BombDamage to be 8, got %d", saveObj.Tuning.BombDamage)
	}
	if saveObj.Tuning.TorchBurnDuration != 100 {
		t.Errorf("expected saved TorchBurnDuration to be 100, got %d", saveObj.Tuning.TorchBurnDuration)
	}
	if saveObj.Tuning.TorchBurnInterval != 8 {
		t.Errorf("expected saved TorchBurnInterval to be 8, got %d", saveObj.Tuning.TorchBurnInterval)
	}
	if saveObj.Tuning.TorchBurnDamage != 3 {
		t.Errorf("expected saved TorchBurnDamage to be 3, got %d", saveObj.Tuning.TorchBurnDamage)
	}

	// Reload with ApplySave
	sess.World.TimeOfDay = 100
	sess.World.SelectedItem = world.ItemSlotBomb
	sess.World.Player.BaseSpeed = 1.0
	sess.World.Player.SwordReach = 5.0
	ApplySave(sess, saveObj, cam)
	if sess.World.TimeOfDay != 4500 {
		t.Errorf("expected loaded TimeOfDay to be 4500, got %d", sess.World.TimeOfDay)
	}
	if sess.World.SelectedItem != world.ItemSlotTorch {
		t.Errorf("expected loaded SelectedItem to be ItemSlotTorch, got %v", sess.World.SelectedItem)
	}
	if sess.World.Player.BaseSpeed != 2.5 {
		t.Errorf("expected loaded BaseSpeed to be 2.5, got %f", sess.World.Player.BaseSpeed)
	}
	if sess.World.Player.SwordReach != 20.0 {
		t.Errorf("expected loaded SwordReach to be 20.0, got %f", sess.World.Player.SwordReach)
	}
	if sess.World.Player.MaxBombs != 12 {
		t.Errorf("expected loaded MaxBombs to be 12, got %d", sess.World.Player.MaxBombs)
	}
	if sess.World.Player.BombFuseDuration != 70 {
		t.Errorf("expected loaded BombFuseDuration to be 70, got %d", sess.World.Player.BombFuseDuration)
	}
	if sess.World.Player.BombRadius != 45.0 {
		t.Errorf("expected loaded BombRadius to be 45.0, got %f", sess.World.Player.BombRadius)
	}
	if sess.World.Player.BombDamage != 8 {
		t.Errorf("expected loaded BombDamage to be 8, got %d", sess.World.Player.BombDamage)
	}
	if sess.World.Player.TorchBurnDuration != 100 {
		t.Errorf("expected loaded TorchBurnDuration to be 100, got %d", sess.World.Player.TorchBurnDuration)
	}
	if sess.World.Player.TorchBurnInterval != 8 {
		t.Errorf("expected loaded TorchBurnInterval to be 8, got %d", sess.World.Player.TorchBurnInterval)
	}
	if sess.World.Player.TorchBurnDamage != 3 {
		t.Errorf("expected loaded TorchBurnDamage to be 3, got %d", sess.World.Player.TorchBurnDamage)
	}

	// 3. Test WarpDoor
	sess.World.TimeOfDay = 6789
	sess.World.SelectedItem = world.ItemSlotTorch
	sess.World.Player.BaseSpeed = 2.5
	sess.World.Player.SwordReach = 20.0
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
	if sess.World.SelectedItem != world.ItemSlotTorch {
		t.Errorf("expected SelectedItem to persist through WarpDoor, got %v", sess.World.SelectedItem)
	}
	if sess.World.Player.BaseSpeed != 2.5 {
		t.Errorf("expected BaseSpeed to persist through WarpDoor, got %f", sess.World.Player.BaseSpeed)
	}
	if sess.World.Player.SwordReach != 20.0 {
		t.Errorf("expected SwordReach to persist through WarpDoor, got %f", sess.World.Player.SwordReach)
	}
	if sess.World.Player.MaxBombs != 12 {
		t.Errorf("expected MaxBombs to persist through WarpDoor, got %d", sess.World.Player.MaxBombs)
	}
	if sess.World.Player.BombFuseDuration != 70 {
		t.Errorf("expected BombFuseDuration to persist through WarpDoor, got %d", sess.World.Player.BombFuseDuration)
	}
	if sess.World.Player.BombRadius != 45.0 {
		t.Errorf("expected BombRadius to persist through WarpDoor, got %f", sess.World.Player.BombRadius)
	}
	if sess.World.Player.BombDamage != 8 {
		t.Errorf("expected BombDamage to persist through WarpDoor, got %d", sess.World.Player.BombDamage)
	}
	if sess.World.Player.TorchBurnDuration != 100 {
		t.Errorf("expected TorchBurnDuration to persist through WarpDoor, got %d", sess.World.Player.TorchBurnDuration)
	}
	if sess.World.Player.TorchBurnInterval != 8 {
		t.Errorf("expected TorchBurnInterval to persist through WarpDoor, got %d", sess.World.Player.TorchBurnInterval)
	}
	if sess.World.Player.TorchBurnDamage != 3 {
		t.Errorf("expected TorchBurnDamage to persist through WarpDoor, got %d", sess.World.Player.TorchBurnDamage)
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
