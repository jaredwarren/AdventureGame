package scenes

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
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
}

func (c *mockGameContext) Input() services.Input {
	return c.input
}

func TestSprintStaminaLogic(t *testing.T) {
	// Create scene and world
	s := &PlayScene{}
	w := &world.World{}
	w.Stats.Resolve = 1 // MaxStamina = 110
	w.Player.Stamina = w.MaxStamina()

	mi := &mockInput{}
	ctx := &mockGameContext{input: mi}

	// 1. Stand still, hold Sprint. Stamina should NOT drain, speed should be base.
	mi.sprintDown = true
	mi.axisX = 0
	mi.axisY = 0

	s.handleMovement(ctx, w)
	if w.Player.Stamina != w.MaxStamina() {
		t.Errorf("expected stamina to remain at max when standing still, got %d", w.Player.Stamina)
	}

	// 2. Move while holding Sprint. Stamina should drain, player should sprint.
	mi.axisX = 1
	w.Player.Stamina = 10

	// Move one tick
	s.handleMovement(ctx, w)
	if w.Player.Stamina != 9 {
		t.Errorf("expected stamina to drain by 1, got %d", w.Player.Stamina)
	}
	if w.Player.SprintExhausted {
		t.Error("expected player not to be sprint exhausted yet")
	}

	// 3. Keep moving and holding Sprint until stamina is depleted.
	for w.Player.Stamina > 0 {
		s.handleMovement(ctx, w)
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
	s.handleMovement(ctx, w)
	if w.Player.Stamina != 1 {
		t.Errorf("expected stamina to regenerate to 1 when exhausted even if sprint is held, got %d", w.Player.Stamina)
	}
	if !w.Player.SprintExhausted {
		t.Error("expected player to remain sprint exhausted while holding Sprint key")
	}

	// 5. Release Sprint key. Exhaustion flag should be cleared.
	mi.sprintDown = false
	s.handleMovement(ctx, w)
	if w.Player.SprintExhausted {
		t.Error("expected sprint exhaustion to clear once Sprint key is released")
	}

	// 6. Press Sprint key again. Player should be able to sprint again.
	mi.sprintDown = true
	// Drains from 2 to 1
	s.handleMovement(ctx, w)
	if w.Player.Stamina != 1 {
		t.Errorf("expected stamina to drain to 1, got %d", w.Player.Stamina)
	}
	if w.Player.SprintExhausted {
		t.Error("expected player not to be sprint exhausted yet")
	}

	// Drains from 1 to 0
	s.handleMovement(ctx, w)
	if w.Player.Stamina != 0 {
		t.Errorf("expected stamina to drain to 0, got %d", w.Player.Stamina)
	}
	if !w.Player.SprintExhausted {
		t.Error("expected player to be sprint exhausted again after depleting stamina")
	}
}
