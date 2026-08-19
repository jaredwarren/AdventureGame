package ebitenplat

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/services"
)

func TestTouchControlsEdgeDetection(t *testing.T) {
	tc := NewTouchControls(true)
	tc.prevDown[services.ActionAttack] = false
	tc.curDown[services.ActionAttack] = true

	if !tc.JustPressed(services.ActionAttack) {
		t.Fatal("expected JustPressed on rising edge")
	}
	if tc.JustReleased(services.ActionAttack) {
		t.Fatal("JustReleased should be false while held")
	}

	tc.prevDown[services.ActionAttack] = true
	if tc.JustPressed(services.ActionAttack) {
		t.Fatal("JustPressed should not repeat while held")
	}

	tc.prevDown[services.ActionAttack] = true
	tc.curDown[services.ActionAttack] = false
	if !tc.JustReleased(services.ActionAttack) {
		t.Fatal("expected JustReleased on falling edge")
	}
}

func TestPointInRect(t *testing.T) {
	if !pointInRect(10, 10, 0, 0, 20, 20) {
		t.Fatal("point inside rect")
	}
	if pointInRect(20, 10, 0, 0, 20, 20) {
		t.Fatal("right edge is exclusive")
	}
	if pointInRect(-1, 10, 0, 0, 20, 20) {
		t.Fatal("point outside left")
	}
}

func TestTouchControlsDisabled(t *testing.T) {
	tc := NewTouchControls(false)
	tc.curDown[services.ActionAttack] = true
	if tc.IsDown(services.ActionAttack) {
		t.Fatal("disabled controls should not report down")
	}
}

func TestStickDirFromUnit(t *testing.T) {
	cases := []struct {
		name                  string
		nx, ny                float32
		up, down, left, right bool
	}{
		{"right", 1, 0, false, false, false, true},
		{"left", -1, 0, false, false, true, false},
		{"up", 0, -1, true, false, false, false},
		{"down", 0, 1, false, true, false, false},
		{"down-right", 0.707, 0.707, false, true, false, true},
		{"up-left", -0.707, -0.707, true, false, true, false},
		{"dead-ish", 0.2, 0.2, false, false, false, false},
	}
	for _, tc := range cases {
		up, down, left, right := stickDirFromUnit(tc.nx, tc.ny)
		if up != tc.up || down != tc.down || left != tc.left || right != tc.right {
			t.Errorf("%s: got up=%v down=%v left=%v right=%v",
				tc.name, up, down, left, right)
		}
	}
}

func TestApplyStickDeltaDeadzoneAndCapture(t *testing.T) {
	tc := NewTouchControls(true)
	tc.applyStickDelta(0, 0)
	if tc.curDown[services.ActionMoveUp] || tc.curDown[services.ActionMoveRight] {
		t.Fatal("center should be deadzone")
	}

	tc.applyStickDelta(20, 0)
	if !tc.curDown[services.ActionMoveRight] {
		t.Fatal("right drag should move right")
	}
	if tc.stickKnobDX <= 0 {
		t.Fatal("knob should follow right")
	}

	tc.applyStickDelta(100, 0) // beyond pad radius
	if tc.stickKnobDX > stickRadius+0.01 {
		t.Fatalf("knob clamped to radius, got %v", tc.stickKnobDX)
	}
}

func TestZoneHitCircleAndRect(t *testing.T) {
	c := circleBtn(50, 50, 10, services.ActionAttack, "ATK")
	if !zoneHit(50, 50, c) {
		t.Fatal("center should hit circle")
	}
	if zoneHit(50, 61, c) {
		t.Fatal("outside radius should miss")
	}
	r := rectBtn(0, 0, 20, 20, services.ActionConfirm, "START")
	if !zoneHit(10, 10, r) {
		t.Fatal("inside rect should hit")
	}
}
