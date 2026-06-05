// systems_test.go — the proof that systems are testable without Ebiten,
// without a scene, and without any I/O.
//
// Each test constructs a minimal *world.World by hand, runs one or more
// systems, and asserts on the post-state + emitted events. No renderer,
// no audio, no main loop.
package systems_test

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
)

// newTestWorld returns a tiny 4x4 all-grass world with the player at (16,16).
// Callers can append Enemies / Pickups as needed. Keeps tests independent
// of tiled/BuildFromTiled so we don't need map files on disk.
func newTestWorld() *world.World {
	const w, h = 4, 4
	tiles := make([]int, w*h)
	for i := range tiles {
		tiles[i] = world.GIDGrass
	}
	return &world.World{
		MapID:          "test",
		TileW:          world.TileSize,
		TileH:          world.TileSize,
		MapW:           w,
		MapH:           h,
		Tiles:          tiles,
		DestroyedTiles: map[int]bool{},
		Player: world.Player{
			Transform: world.Transform{X: 16, Y: 16},
			Hitbox:    world.Hitbox{W: 12, H: 12},
			Facing:    world.Facing{Dir: world.DirDown},
			Stamina:   100,
		},
		Stats: progression.DefaultStats(),
		HP:    progression.DefaultStats().MaxHP(),
	}
}

// TestTimersSystem_DecrementsAllCounters proves every tick counter walks
// toward zero and that Tick monotonically increments.
func TestTimersSystem_DecrementsAllCounters(t *testing.T) {
	w := newTestWorld()
	w.DoorCooldown = 3
	w.Player.Swing = 5
	w.Player.SwingCD = 10
	w.Player.Invuln = 4
	w.Player.DodgeTimer = 2
	enemy := world.NewEnemy(world.NoEntity, 0, 0, 1, false)
	enemy.HurtCD = 5
	w.Enemies = []world.Enemy{enemy}

	var bus systems.EventBus
	if err := (systems.TimersSystem{}).Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("TimersSystem returned %v", err)
	}
	if w.Tick != 1 {
		t.Errorf("Tick = %d, want 1", w.Tick)
	}
	if w.DoorCooldown != 2 {
		t.Errorf("DoorCooldown = %d, want 2", w.DoorCooldown)
	}
	if w.Player.Swing != 4 || w.Player.SwingCD != 9 || w.Player.Invuln != 3 || w.Player.DodgeTimer != 1 {
		t.Errorf("player timers not decremented: %+v", w.Player)
	}
	if w.Enemies[0].HurtCD != 4 {
		t.Errorf("enemy HurtCD = %d, want 4", w.Enemies[0].HurtCD)
	}
	if bus.Len() != 0 {
		t.Errorf("TimersSystem should not emit events, got %d", bus.Len())
	}
}

// TestPickupSystem_EmitsEventAndMutatesWorld covers the coin and heart cases
// together so we know both the event and the state change happen.
func TestPickupSystem_EmitsEventAndMutatesWorld(t *testing.T) {
	w := newTestWorld()
	w.HP = 1
	w.Pickups = []world.Pickup{
		world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupCoin),
		world.NewPickup(world.NoEntity, w.Player.X+3, w.Player.Y+3, world.PickupHeart),
		world.NewPickup(world.NoEntity, 9999, 9999, world.PickupBomb), // out of reach
	}

	var bus systems.EventBus
	if err := (systems.PickupSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("PickupSystem returned %v", err)
	}
	if w.Currency != 1 {
		t.Errorf("Currency = %d, want 1", w.Currency)
	}
	if w.HP != 2 {
		t.Errorf("HP = %d, want 2 (heart healed)", w.HP)
	}
	if w.Bombs != 0 {
		t.Errorf("bomb picked up despite being far away")
	}
	if !w.Pickups[0].Gone || !w.Pickups[1].Gone {
		t.Errorf("reachable pickups not marked Gone: %+v", w.Pickups)
	}

	evs := bus.Drain()
	if len(evs) != 2 {
		t.Fatalf("expected 2 events, got %d: %+v", len(evs), evs)
	}
	for _, ev := range evs {
		pe, ok := ev.(systems.PickupEvent)
		if !ok {
			t.Errorf("unexpected event type %T", ev)
			continue
		}
		if pe.Kind != world.PickupCoin && pe.Kind != world.PickupHeart {
			t.Errorf("unexpected kind %v", pe.Kind)
		}
	}
}

func TestPickupSystem_BombRespectsMaxCarry(t *testing.T) {
	w := newTestWorld()
	w.Bombs = world.MaxBombsCarry
	w.Pickups = []world.Pickup{
		world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupBomb),
	}
	var bus systems.EventBus
	if err := (systems.PickupSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("PickupSystem: %v", err)
	}
	if w.Bombs != world.MaxBombsCarry {
		t.Errorf("Bombs = %d, want cap %d", w.Bombs, world.MaxBombsCarry)
	}
	if !w.Pickups[0].Gone {
		t.Error("pickup should still be consumed when at cap")
	}
}

// TestCombatSystem_EmitsHitAndKillFlags verifies the Killed + IsBoss flags
// on HitEvent are correct.
func TestCombatSystem_EmitsHitAndKillFlags(t *testing.T) {
	w := newTestWorld()
	w.Player.Swing = 7 // active window is [5..10]
	w.Player.Dir = world.DirRight
	w.Enemies = []world.Enemy{
		world.NewEnemy(world.NoEntity, w.Player.X+w.Player.W*0.5+4, w.Player.Y+w.Player.H*0.5-6, 1, true),
	}

	var bus systems.EventBus
	if err := (systems.CombatSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("CombatSystem returned %v", err)
	}
	if w.Enemies[0].HP != 0 {
		t.Errorf("enemy HP = %d, want 0", w.Enemies[0].HP)
	}
	evs := bus.Drain()
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(evs), evs)
	}
	he, ok := evs[0].(systems.HitEvent)
	if !ok {
		t.Fatalf("expected HitEvent, got %T", evs[0])
	}
	if !he.Killed || !he.IsBoss {
		t.Errorf("flags wrong: %+v (want Killed=true IsBoss=true)", he)
	}
}

// TestCombatSystem_TorchBurnsFaceTile verifies fire damage in front of the
// player during an active torch swing.
// TestCombatSystem_TorchBurnsFaceTile verifies that swinging a torch ignites
// the facing tree, which burns over time and finally breaks via TimersSystem.
func TestCombatSystem_TorchBurnsFaceTile(t *testing.T) {
	w := newTestWorld()
	w.Player.Dir = world.DirRight
	w.Player.TorchSwing = 7
	cx := w.Player.X + w.Player.W*0.5
	cy := w.Player.Y + w.Player.H*0.5
	tx := int(cx/world.TileSize) + 1
	ty := int(cy / world.TileSize)
	idx := ty*w.MapW + tx
	w.Tiles[idx] = world.GIDTree

	var bus systems.EventBus
	if err := (systems.CombatSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("CombatSystem: %v", err)
	}

	// The tree should be ignited (not immediately destroyed)
	if w.DestroyedTiles[idx] {
		t.Fatal("tree should not be destroyed immediately upon torch hit")
	}
	if len(w.Flames) != 1 || w.Flames[0].TX != tx || w.Flames[0].TY != ty {
		t.Fatalf("expected 1 active flame at (%d, %d), got %+v", tx, ty, w.Flames)
	}

	// Tick TimersSystem for 90 frames to finish burning
	timers := systems.TimersSystem{}
	for i := 0; i < 90; i++ {
		if err := timers.Update(w, &bus, 1.0/60); err != nil {
			t.Fatalf("TimersSystem failed on tick %d: %v", i, err)
		}
	}

	// Now the tree must be destroyed
	if !w.DestroyedTiles[idx] {
		t.Fatal("expected tree tile destroyed after 90 ticks of burning")
	}
	if len(w.Flames) != 0 {
		t.Fatalf("expected active flames cleared, got %+v", w.Flames)
	}

	evs := bus.Drain()
	var sawTile bool
	for _, ev := range evs {
		if _, ok := ev.(systems.TileDestroyedEvent); ok {
			sawTile = true
		}
	}
	if !sawTile {
		t.Fatalf("expected TileDestroyedEvent, got %+v", evs)
	}
}

// TestBurnSystem_DoT damages an ignited enemy on tick intervals.
func TestBurnSystem_DoT(t *testing.T) {
	w := newTestWorld()
	w.Enemies = []world.Enemy{
		world.NewEnemy(world.NoEntity, 200, 200, 10, false),
	}
	w.IgniteEnemy(0)
	if w.Enemies[0].BurnTimer <= 0 {
		t.Fatal("IgniteEnemy should set BurnTimer")
	}
	var bus systems.EventBus
	var last []systems.Event
	for i := 0; i < world.TorchBurnTickInterval; i++ {
		if err := (systems.BurnSystem{}).Update(w, &bus, 0); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		last = bus.Drain()
	}
	if w.Enemies[0].HP >= 10 {
		t.Fatalf("expected burn DoT after %d ticks, HP=%d", world.TorchBurnTickInterval, w.Enemies[0].HP)
	}
	var sawDoT bool
	for _, ev := range last {
		he, ok := ev.(systems.HitEvent)
		if ok && he.FromBurnDoT {
			sawDoT = true
		}
	}
	if !sawDoT {
		t.Fatalf("expected HitEvent with FromBurnDoT, last events: %+v", last)
	}
}

// TestLockSystem_ConvertsTileAndConsumesKey exercises the whole lifecycle
// including the "one key per frame even if multiple tiles" invariant.
func TestLockSystem_ConvertsTileAndConsumesKey(t *testing.T) {
	w := newTestWorld()
	w.Player.X = 16
	w.Player.Y = 16
	w.Player.W = 18 // straddle tile (1,1) and (2,1)
	w.Player.H = 12
	w.Tiles[1*w.MapW+1] = world.GIDLock
	w.Tiles[1*w.MapW+2] = world.GIDLock
	w.SmallKey = 1

	var bus systems.EventBus
	if err := (systems.LockSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("LockSystem returned %v", err)
	}
	if w.GIDAt(1, 1) != world.GIDGrass || w.GIDAt(2, 1) != world.GIDGrass {
		t.Errorf("lock tiles not converted: %d %d", w.GIDAt(1, 1), w.GIDAt(2, 1))
	}
	if w.SmallKey != 0 {
		t.Errorf("SmallKey = %d, want 0 (exactly one key consumed)", w.SmallKey)
	}
	if bus.Len() != 2 {
		t.Errorf("expected 2 LockOpenEvents (one per tile), got %d", bus.Len())
	}
}

func TestEnemyAISystem_NoChaseOutsideAggro(t *testing.T) {
	w := newTestWorld()
	w.Enemies = []world.Enemy{
		world.NewEnemy(world.NoEntity, 400, 400, 3, false),
	}
	x0, y0 := w.Enemies[0].X, w.Enemies[0].Y
	var bus systems.EventBus
	if err := (systems.EnemyAISystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("EnemyAISystem: %v", err)
	}
	if w.Enemies[0].X != x0 || w.Enemies[0].Y != y0 {
		t.Errorf("enemy moved while player out of aggro: (%.1f,%.1f) -> (%.1f,%.1f)",
			x0, y0, w.Enemies[0].X, w.Enemies[0].Y)
	}
}

// TestPipeline_DefaultOrderMatchesLegacy runs a minimal scenario end-to-end
// through the default pipeline and confirms we see expected events in
// expected order.
func TestPipeline_DefaultOrderMatchesLegacy(t *testing.T) {
	w := newTestWorld()
	w.Player.Swing = 7
	w.Player.Dir = world.DirRight
	w.Enemies = []world.Enemy{
		world.NewEnemy(world.NoEntity, w.Player.X+w.Player.W*0.5+4, w.Player.Y+w.Player.H*0.5-6, 3, false),
	}
	w.Pickups = []world.Pickup{
		world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupCoin),
	}

	pipe := systems.Default()
	evs, err := pipe.Tick(w, 1.0/60)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	var sawHit, sawPickup bool
	for _, ev := range evs {
		switch ev.(type) {
		case systems.HitEvent:
			sawHit = true
		case systems.PickupEvent:
			sawPickup = true
		}
	}
	if !sawHit {
		t.Errorf("missing HitEvent")
	}
	if !sawPickup {
		t.Errorf("missing PickupEvent")
	}
	if w.Tick != 1 {
		t.Errorf("Tick not incremented by TimersSystem in pipeline: %d", w.Tick)
	}
	// After Timers runs, swing decremented from 7 -> 6 (still in active window),
	// so combat hitbox should have been tested in this same frame.
	if w.Player.Swing != 6 {
		t.Errorf("Swing = %d, want 6 (pipeline ordering broken?)", w.Player.Swing)
	}
}

func TestTimersSystem_FlamePlayerContactDamage(t *testing.T) {
	w := newTestWorld()
	// Set player position to overlap with tile (1, 1)
	w.Player.X = 16
	w.Player.Y = 16
	// Add an active flame at (1, 1)
	w.Flames = []world.ActiveFlame{
		{
			X:     24,
			Y:     24,
			TX:    1,
			TY:    1,
			Timer: 50,
		},
	}
	w.HP = 5
	w.Player.Invuln = 0

	var bus systems.EventBus
	timers := systems.TimersSystem{}
	if err := timers.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("TimersSystem failed: %v", err)
	}

	if w.HP != 4 {
		t.Errorf("expected HP to be 4 after flame contact, got %d", w.HP)
	}
	if w.Player.Invuln != 45 {
		t.Errorf("expected Player.Invuln to be 45, got %d", w.Player.Invuln)
	}
	evs := bus.Drain()
	var sawHurt bool
	for _, ev := range evs {
		if _, ok := ev.(systems.PlayerHurtEvent); ok {
			sawHurt = true
		}
	}
	if !sawHurt {
		t.Errorf("expected PlayerHurtEvent, got %+v", evs)
	}
}

