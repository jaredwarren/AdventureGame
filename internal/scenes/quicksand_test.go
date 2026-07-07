package scenes

import (
	"math"
	"testing"

	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func TestQuicksandMechanics(t *testing.T) {
	// 1. Setup scene, world and tile map
	// The quicksand GIDQuicksand is placed at tile (1,1) -> X range: [16, 32], Y range: [16, 32].
	s := &PlayScene{}
	w := &world.World{
		TileW: 16,
		TileH: 16,
		MapW:  5,
		MapH:  5,
	}
	w.Layers = [][]int{
		{
			tile.GIDGrass, tile.GIDGrass, tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
			tile.GIDGrass, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDGrass,
			tile.GIDGrass, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDGrass,
			tile.GIDGrass, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDQuicksand, tile.GIDGrass,
			tile.GIDGrass, tile.GIDGrass, tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
		},
	}
	w.Tiles = w.Layers[0]

	w.Player = world.DefaultPlayerTuning()
	w.Player.W = 12
	w.Player.H = 12
	w.Stats.Resolve = 1 // MaxStamina = 100
	w.Player.Stamina = w.MaxStamina()

	mi := &mockInput{}
	ctx := &mockGameContext{input: mi}
	staminaSys := systems.StaminaSystem{}
	var bus systems.EventBus

	stepTick := func() {
		s.handleMovement(ctx, w)
		_ = staminaSys.Update(w, &bus, 1.0/60)
		w.Tick++
	}

	// Make sure we have Pegasus Boots so sprinting is allowed
	w.HasPegasusBoots = true

	// Position 1: Start on Grass (X=9, Y=24) - center is 15 (just left of quicksand boundary at X=16).
	w.Player.X = 9
	w.Player.Y = 24
	mi.axisX = 1
	mi.axisY = 0

	// Walk one tick to cross the boundary into quicksand
	mi.sprintDown = false
	stepTick()
	baseSpeed := w.Player.EffectiveBaseSpeed()
	if w.Player.X <= 9 {
		t.Errorf("expected player to move right, got X=%f", w.Player.X)
	}

	// Now we are inside the quicksand tile (center > 16)
	if w.Player.X+6 <= 16 {
		t.Errorf("expected player center to have crossed the boundary, but X=%f", w.Player.X)
	}

	// Next step should apply the quicksand slowdown (SpeedMultiplier = 0.05)
	// We run it a few ticks to let the momentum/friction settle to target speed
	for k := 0; k < 5; k++ {
		stepTick()
	}
	lastX := w.Player.X
	stepTick()

	dxWalk := w.Player.X - lastX
	expectedWalk := baseSpeed * 0.05
	if dxWalk <= 0 || dxWalk > expectedWalk+0.01 {
		t.Errorf("expected slow quicksand speed (~%f), got moved by %f", expectedWalk, dxWalk)
	}

	// Try starting a sprint while walking on quicksand
	mi.sprintDown = true
	mi.axisX = 1
	stepTick()
	if w.Player.IsSprinting {
		t.Error("expected player NOT to be able to start sprinting from a standstill on quicksand")
	}

	// Move player back to grass (X=8, Y=24)
	w.Player.X = 8
	w.Player.Y = 24
	w.Player.WasOnQuicksand = false
	w.Player.RunningAcrossQuicksand = false
	mi.sprintDown = true
	mi.axisX = 1
	stepTick()
	if !w.Player.IsSprinting {
		t.Error("expected player to sprint on grass")
	}

	// Sprint from grass (X=9) into quicksand (boundary at X=16)
	w.Player.X = 9
	w.Player.Y = 24
	w.Player.WasOnQuicksand = false
	w.Player.RunningAcrossQuicksand = false
	w.Player.IsSprinting = true
	mi.sprintDown = true
	mi.axisX = 1
	stepTick()

	// They should cross into quicksand, and RunningAcrossQuicksand should become true
	if !w.Player.RunningAcrossQuicksand {
		t.Error("expected RunningAcrossQuicksand to be true when crossing the boundary while sprinting")
	}

	// Next step: should run at full sprint speed across quicksand
	lastX = w.Player.X
	stepTick()
	dxSprint := w.Player.X - lastX
	sprintSpeed := w.Player.EffectiveSprintSpeed()
	if dxSprint <= baseSpeed || dxSprint < sprintSpeed-0.01 {
		t.Errorf("expected sprint speed across quicksand (~%f), got %f", sprintSpeed, dxSprint)
	}

	// Release sprint key while on quicksand
	mi.sprintDown = false
	stepTick() // Tick N: moves at sprint speed, then StaminaSystem updates IsSprinting to false
	if w.Player.IsSprinting {
		t.Error("expected IsSprinting to become false after releasing sprint key")
	}

	// We run it a few ticks to let the momentum/friction settle to target walking speed
	for k := 0; k < 5; k++ {
		stepTick()
	}

	lastX = w.Player.X
	stepTick() // Tick N+1: moves at slowed speed, sets RunningAcrossQuicksand to false
	if w.Player.RunningAcrossQuicksand {
		t.Error("expected RunningAcrossQuicksand to become false after stopping sprint")
	}
	dxSlowed := w.Player.X - lastX
	if dxSlowed > expectedWalk+0.01 {
		t.Errorf("expected to slow down after stopping sprint, got moved by %f", dxSlowed)
	}
}

func TestIceFriction(t *testing.T) {
	s := &PlayScene{}
	w := &world.World{
		TileW: 16,
		TileH: 16,
		MapW:  3,
		MapH:  3,
	}
	w.Tiles = []int{
		tile.GIDIce, tile.GIDIce, tile.GIDIce,
		tile.GIDIce, tile.GIDIce, tile.GIDIce,
		tile.GIDIce, tile.GIDIce, tile.GIDIce,
	}
	w.Layers = [][]int{w.Tiles}

	w.Player = world.DefaultPlayerTuning()
	w.Player.W = 12
	w.Player.H = 12
	w.Player.X = 16
	w.Player.Y = 16

	mi := &mockInput{}
	ctx := &mockGameContext{input: mi}

	// Press move right
	mi.axisX = 1
	mi.axisY = 0

	s.handleMovement(ctx, w)
	// Because of Ice friction (0.02), player velocity VX should only build up by 2% towards target speed.
	expectedVX := w.Player.EffectiveBaseSpeed() * 1.1 * 0.02
	if math.Abs(w.Player.VX-expectedVX) > 0.01 {
		t.Errorf("expected VX to be ~%f, got %f", expectedVX, w.Player.VX)
	}

	// Let the player slide for one tick with no input
	mi.axisX = 0
	lastX := w.Player.X
	s.handleMovement(ctx, w)

	// Player VX should decay but still be non-zero (i.e. they slide!)
	if w.Player.VX <= 0 {
		t.Error("expected player to continue sliding (VX > 0) when keys are released on Ice")
	}
	if w.Player.X <= lastX {
		t.Error("expected player position to have moved forward due to sliding")
	}
}
