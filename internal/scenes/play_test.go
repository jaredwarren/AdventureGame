package scenes

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

type mockInput struct {
	services.Input
	sprintDown bool
	axisX      int
	axisY      int
}

func (m *mockInput) IsDown(a services.Action) bool {
	if a == services.ActionSprint {
		return m.sprintDown
	}
	return false
}

func (m *mockInput) Axis2D() (x, y int) {
	return m.axisX, m.axisY
}

type mockGameContext struct {
	GameContext
	input services.Input
	sess  *run.Session
}

func (c *mockGameContext) Input() services.Input {
	return c.input
}

func (c *mockGameContext) Session() *run.Session {
	return c.sess
}

func TestSprintStaminaLogic(t *testing.T) {
	// Create scene and world
	s := &PlayScene{}
	w := &world.World{}
	w.Stats.Resolve = 1 // MaxStamina = 100
	w.Player.Stamina = w.MaxStamina()

	mi := &mockInput{}
	ctx := &mockGameContext{input: mi}
	staminaSys := systems.StaminaSystem{}
	var bus systems.EventBus

	stepTick := func() {
		s.handleMovement(ctx, w)
		_ = staminaSys.Update(w, &bus, 1.0/60)
	}

	// 0. Without Pegasus Boots, holding Sprint should NOT trigger SprintHeld.
	mi.sprintDown = true
	mi.axisX = 1
	stepTick()
	if w.Player.SprintHeld {
		t.Error("expected SprintHeld to be false when player does not have Pegasus Boots")
	}
	if w.Player.Stamina != w.MaxStamina() {
		t.Errorf("expected stamina not to drain without Pegasus Boots, got %d", w.Player.Stamina)
	}

	// Grant Pegasus Boots for sprint logic testing
	w.HasPegasusBoots = true

	// 1. Stand still, hold Sprint. Stamina should NOT drain, speed should be base.
	mi.sprintDown = true
	mi.axisX = 0
	mi.axisY = 0

	stepTick()
	if w.Player.Stamina != w.MaxStamina() {
		t.Errorf("expected stamina to remain at max when standing still, got %d", w.Player.Stamina)
	}

	// 2. Move while holding Sprint. Stamina should drain, player should sprint.
	mi.axisX = 1
	w.Player.Stamina = 10

	// Move one tick
	stepTick()
	if w.Player.Stamina != 9 {
		t.Errorf("expected stamina to drain by 1, got %d", w.Player.Stamina)
	}
	if w.Player.SprintExhausted {
		t.Error("expected player not to be sprint exhausted yet")
	}

	// 3. Keep moving and holding Sprint until stamina is depleted.
	for w.Player.Stamina > 0 {
		stepTick()
	}

	if w.Player.Stamina != 0 {
		t.Errorf("expected stamina to hit 0, got %d", w.Player.Stamina)
	}
	if !w.Player.SprintExhausted {
		t.Error("expected player to be sprint exhausted when stamina hits 0")
	}

	// 4. Continue holding Sprint while moving. Player should not sprint, and stamina should regenerate.
	// (Stamina regens every even tick when not sprinting)
	w.Tick = 2
	stepTick()
	if w.Player.Stamina != 1 {
		t.Errorf("expected stamina to regenerate to 1 when exhausted even if sprint is held, got %d", w.Player.Stamina)
	}
	if !w.Player.SprintExhausted {
		t.Error("expected player to remain sprint exhausted while holding Sprint key")
	}

	// 5. Release Sprint key. Exhaustion flag should be cleared.
	mi.sprintDown = false
	stepTick()
	if w.Player.SprintExhausted {
		t.Error("expected sprint exhaustion to clear once Sprint key is released")
	}

	// 6. Press Sprint key again. Player should be able to sprint again.
	mi.sprintDown = true
	// Drains from 2 to 1
	stepTick()
	if w.Player.Stamina != 1 {
		t.Errorf("expected stamina to drain to 1, got %d", w.Player.Stamina)
	}
	if w.Player.SprintExhausted {
		t.Error("expected player not to be sprint exhausted yet")
	}

	// Drains from 1 to 0
	stepTick()
	if w.Player.Stamina != 0 {
		t.Errorf("expected stamina to drain to 0, got %d", w.Player.Stamina)
	}
	if !w.Player.SprintExhausted {
		t.Error("expected player to be sprint exhausted again after depleting stamina")
	}
}

func TestEditorTileMenuGIDs(t *testing.T) {
	s := &EditorScene{}
	s.activeLayerIndex = 0
	s.showOnlyActiveLayer = true
	gids0 := s.filteredTileGIDs()
	t.Logf("Layer 0 (filtered) GIDs:")
	for _, g := range gids0 {
		t.Logf("  GID %d: %s", g, world.TileDefOf(g).Name)
	}

	s.activeLayerIndex = 1
	gids1 := s.filteredTileGIDs()
	t.Logf("Layer 1 (filtered) GIDs:")
	for _, g := range gids1 {
		t.Logf("  GID %d: %s", g, world.TileDefOf(g).Name)
	}
}

type clickMockInput struct {
	cx, cy int
	click  bool
}

func (m *clickMockInput) IsDown(a services.Action) bool             { return false }
func (m *clickMockInput) JustPressed(a services.Action) bool        { return false }
func (m *clickMockInput) JustReleased(a services.Action) bool       { return false }
func (m *clickMockInput) Axis2D() (x, y int)                        { return 0, 0 }
func (m *clickMockInput) IsModifierDown(mod services.Modifier) bool { return false }
func (m *clickMockInput) MousePressed(b services.MouseButton) bool {
	if b == services.MouseLeft {
		return m.click
	}
	return false
}
func (m *clickMockInput) MouseJustPressed(b services.MouseButton) bool {
	if b == services.MouseLeft {
		return m.click
	}
	return false
}
func (m *clickMockInput) MouseJustReleased(b services.MouseButton) bool { return false }
func (m *clickMockInput) CursorPosition() (int, int)                    { return m.cx, m.cy }

func TestEditorTileMenuClickSelection(t *testing.T) {
	s := &EditorScene{}
	s.activeLayerIndex = 1
	s.showOnlyActiveLayer = true
	s.showTileMenu = true
	s.tileMenuCategory = "water" // start in water category
	s.tileMenuScroll = 0

	s.tm = &tiled.Map{Width: 10, Height: 10}
	w := &world.World{}
	sess := &run.Session{World: w}

	items := s.currentMenuItems()

	// Simulate clicking on row 1 (which is GIDWater base)
	row := 1
	mi := &clickMockInput{
		cx:    100,                   // inside panelX (70..250)
		cy:    30 + 22 + row*20 + 10, // inside row bounds
		click: true,
	}
	ctx := &mockGameContext{input: mi, sess: sess}
	err := s.Update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedGID := items[row].gid
	if s.brushGID != expectedGID {
		t.Errorf("expected GID %d, got %d", expectedGID, s.brushGID)
	}
}

func TestEditorTileMenuHierarchy(t *testing.T) {
	s := &EditorScene{}
	s.activeLayerIndex = 1
	s.showOnlyActiveLayer = true
	s.tileMenuCategory = ""

	// 1. Root level menu items check
	items := s.currentMenuItems()
	hasWallCategory := false
	hasWaterCategory := false
	hasRockCategory := false
	for _, item := range items {
		if item.isCategory {
			if item.category == "wall" {
				hasWallCategory = true
			} else if item.category == "water" {
				hasWaterCategory = true
			} else if item.category == "rock" {
				hasRockCategory = true
			}
		}
	}
	if !hasWallCategory {
		t.Error("expected root menu on Layer 1 to have 'wall >' category")
	}
	if !hasWaterCategory {
		t.Error("expected root menu on Layer 1 to have 'water >' category")
	}
	if !hasRockCategory {
		t.Error("expected root menu on Layer 1 to have 'rock >' category")
	}

	// 2. Navigate into water category
	s.tileMenuCategory = "water"
	waterItems := s.currentMenuItems()
	if len(waterItems) == 0 {
		t.Fatal("expected water category menu to have items")
	}
	if !waterItems[0].isBack {
		t.Errorf("expected first item in sub-menu to be back, got %v", waterItems[0])
	}
	if waterItems[1].gid != tile.GIDWater {
		t.Errorf("expected second item in sub-menu to be GIDWater (5), got %d", waterItems[1].gid)
	}

	// 3. Navigate into rock category
	s.tileMenuCategory = "rock"
	rockItems := s.currentMenuItems()
	if len(rockItems) == 0 {
		t.Fatal("expected rock category menu to have items")
	}
	if !rockItems[0].isBack {
		t.Errorf("expected first item in sub-menu to be back, got %v", rockItems[0])
	}
	if rockItems[1].gid != tile.GIDRock {
		t.Errorf("expected second item in sub-menu to be GIDRock (34), got %d", rockItems[1].gid)
	}

	// 4. Simulate clicking back
	w := &world.World{}
	sess := &run.Session{World: w}
	s.tm = &tiled.Map{Width: 10, Height: 10}

	mi := &clickMockInput{
		cx:    100,
		cy:    30 + 22 + 0*20 + 10, // Row 0 is Back
		click: true,
	}
	s.showTileMenu = true
	ctx := &mockGameContext{input: mi, sess: sess}
	err := s.Update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.tileMenuCategory != "" {
		t.Errorf("expected s.tileMenuCategory to be empty after clicking back, got %q", s.tileMenuCategory)
	}
}
