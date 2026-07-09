// systems_test.go — the proof that systems are testable without Ebiten,
// without a scene, and without any I/O.
//
// Each test constructs a minimal *world.World by hand, runs one or more
// systems, and asserts on the post-state + emitted events. No renderer,
// no audio, no main loop.
package systems_test

import (
	"math"
	"testing"

	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// newTestWorld returns a tiny 4x4 all-grass world with the player at (16,16).
// Callers can append Enemies / Pickups as needed. Keeps tests independent
// of tiled/BuildFromTiled so we don't need map files on disk.
func newTestWorld() *world.World {
	const w, h = 4, 4
	tiles := make([]int, w*h)
	for i := range tiles {
		tiles[i] = tile.GIDGrass
	}
	return &world.World{
		MapID:          "test",
		TileW:          tile.Size,
		TileH:          tile.Size,
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
	w.Bombs = w.MaxBombsCarry()
	w.Pickups = []world.Pickup{
		world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupBomb),
	}
	var bus systems.EventBus
	if err := (systems.PickupSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("PickupSystem: %v", err)
	}
	cap := w.MaxBombsCarry()
	if w.Bombs != cap {
		t.Errorf("Bombs = %d, want cap %d", w.Bombs, cap)
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
	tx := int(cx/tile.Size) + 1
	ty := int(cy / tile.Size)
	idx := ty*w.MapW + tx
	w.Tiles[idx] = tile.GIDTree

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

	// Tick HazardSystem for 90 frames to finish burning
	hazard := systems.HazardSystem{}
	for i := 0; i < 90; i++ {
		if err := hazard.Update(w, &bus, 1.0/60); err != nil {
			t.Fatalf("HazardSystem failed on tick %d: %v", i, err)
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
	burnInterval := world.DefaultPlayerTuning().TorchBurnInterval
	for i := 0; i < burnInterval; i++ {
		if err := (systems.BurnSystem{}).Update(w, &bus, 0); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		last = bus.Drain()
	}
	if w.Enemies[0].HP >= 10 {
		t.Fatalf("expected burn DoT after %d ticks, HP=%d", burnInterval, w.Enemies[0].HP)
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
	w.Tiles[1*w.MapW+1] = tile.GIDLock
	w.Tiles[1*w.MapW+2] = tile.GIDLock
	w.SmallKey = 1

	var bus systems.EventBus
	if err := (systems.LockSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("LockSystem returned %v", err)
	}
	if w.GIDAt(1, 1) != tile.GIDGrass || w.GIDAt(2, 1) != tile.GIDGrass {
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

func TestHazardSystem_FlamePlayerContactDamage(t *testing.T) {
	w := newTestWorld()
	w.Player.X = 16
	w.Player.Y = 16
	w.Flames = []world.ActiveFlame{
		{X: 24, Y: 24, TX: 1, TY: 1, Timer: 60},
	}
	w.HP = 5
	w.Player.Invuln = 0

	var bus systems.EventBus
	hazard := systems.HazardSystem{}
	if err := hazard.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("HazardSystem failed: %v", err)
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

func TestPickupSystem_ChestsAreIgnored(t *testing.T) {
	w := newTestWorld()
	p := world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupSmallKey)
	p.PersistentSaveKey = "testmap:1"
	p.IsChest = true
	w.Pickups = []world.Pickup{p}

	var bus systems.EventBus
	if err := (systems.PickupSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("PickupSystem returned %v", err)
	}

	if w.SmallKey != 0 {
		t.Errorf("SmallKey = %d, want 0 (chest should not be collected automatically)", w.SmallKey)
	}
	if w.Pickups[0].Gone || w.Pickups[0].Opened {
		t.Errorf("chest pickup was modified: %+v", w.Pickups[0])
	}
	if bus.Len() != 0 {
		t.Errorf("expected 0 events, got %d", bus.Len())
	}
}

func TestPickupSystem_PersistentNonChestsAreCollected(t *testing.T) {
	w := newTestWorld()
	p := world.NewPickup(world.NoEntity, w.Player.X+2, w.Player.Y+2, world.PickupSmallKey)
	p.PersistentSaveKey = "testmap:2"
	p.IsChest = false
	w.Pickups = []world.Pickup{p}

	var bus systems.EventBus
	if err := (systems.PickupSystem{}).Update(w, &bus, 0); err != nil {
		t.Fatalf("PickupSystem returned %v", err)
	}

	if w.SmallKey != 1 {
		t.Errorf("SmallKey = %d, want 1 (flat persistent pickup should be collected automatically)", w.SmallKey)
	}
	if !w.Pickups[0].Gone {
		t.Errorf("expected pickup to be marked Gone, got %+v", w.Pickups[0])
	}
	if bus.Len() != 1 {
		t.Errorf("expected 1 event, got %d", bus.Len())
	}
}

func TestBombSystem_BossKill(t *testing.T) {
	w := newTestWorld()
	w.Bombs = 1
	boss := world.NewEnemy(world.NoEntity, 16, 16, 2, true)
	w.Enemies = []world.Enemy{boss}

	w.ActiveBombs = []world.ActiveBomb{
		{X: 16, Y: 16, TX: 1, TY: 1, Timer: 1},
	}

	p := systems.Default()
	events, err := p.Tick(w, 1.0/60)
	if err != nil {
		t.Fatalf("pipeline tick error: %v", err)
	}

	var sawBossKill bool
	var sawExplosion bool
	for _, ev := range events {
		if h, ok := ev.(systems.HitEvent); ok {
			if h.Killed && h.IsBoss {
				sawBossKill = true
			}
		}
		if _, ok := ev.(systems.ExplosionEvent); ok {
			sawExplosion = true
		}
	}

	if !sawBossKill {
		t.Errorf("expected HitEvent with Killed=true, IsBoss=true")
	}
	if !sawExplosion {
		t.Errorf("expected ExplosionEvent")
	}
}

func TestBombSystem_CrackedWallDestruction(t *testing.T) {
	w := newTestWorld()
	idx := 1*w.MapW + 1
	w.Tiles[idx] = tile.GIDCracked

	w.ActiveBombs = []world.ActiveBomb{
		{X: 24, Y: 24, TX: 1, TY: 1, Timer: 1},
	}

	p := systems.Default()
	events, err := p.Tick(w, 1.0/60)
	if err != nil {
		t.Fatalf("pipeline tick error: %v", err)
	}

	if !w.DestroyedTiles[idx] {
		t.Errorf("expected cracked tile at (1,1) to be marked destroyed")
	}

	var sawTileDestroyed bool
	for _, ev := range events {
		if _, ok := ev.(systems.TileDestroyedEvent); ok {
			sawTileDestroyed = true
		}
	}
	if !sawTileDestroyed {
		t.Errorf("expected TileDestroyedEvent")
	}
}

func TestStaminaSystem_DrainAndRegen(t *testing.T) {
	w := newTestWorld()
	maxStam := w.MaxStamina()
	w.Player.Stamina = maxStam

	p := systems.Default()

	w.Player.SprintHeld = true
	w.Player.IsMoving = true

	_, err := p.Tick(w, 1.0/60)
	if err != nil {
		t.Fatalf("pipeline tick error: %v", err)
	}

	if w.Player.Stamina >= maxStam {
		t.Errorf("expected stamina to drain while sprinting, got %d", w.Player.Stamina)
	}

	w.Player.SprintHeld = false
	w.Player.IsMoving = false

	drainedStamina := w.Player.Stamina
	regenInterval := w.Player.EffectiveStaminaRegenInterval()
	for i := 0; i < regenInterval*2; i++ {
		_, _ = p.Tick(w, 1.0/60)
	}

	if w.Player.Stamina <= drainedStamina {
		t.Errorf("expected stamina to regen when not sprinting, got %d (was %d)", w.Player.Stamina, drainedStamina)
	}
}

func TestEnemyAISystem_CustomNightBuffsMultiplier(t *testing.T) {
	w := newTestWorld()
	w.TimeOfDay = 0 // Night!
	w.Balance = balance.Default()
	w.Balance.NightBuffs = balance.NightBuffs{
		AggroMultiplier: 2.0, // 25 * 2.0 = 50 aggro radius
		SpeedMultiplier: 3.0, // 1.0 * 3.0 = 3.0 speed
	}

	// Player is at (16, 16), center (22, 22).
	// Place enemy at (50, 16), center (56, 22), distance 34 px from player (out of 25 aggro, but within 50 aggro).
	enemy := world.NewEnemy(world.NoEntity, 50, 16, 10, false)
	enemy.AI.AggroRadius = 25
	enemy.AI.Speed = 1.0
	w.Enemies = []world.Enemy{enemy}

	var bus systems.EventBus
	ai := systems.EnemyAISystem{}
	if err := ai.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("EnemyAISystem failed: %v", err)
	}

	// Enemy should have moved left by speed ~3.0 (X decreased from 50 to ~47)
	if math.Abs(w.Enemies[0].X-47.0) > 0.1 {
		t.Errorf("expected enemy X to be ~47.0 with 3.0x speed multiplier, got %f", w.Enemies[0].X)
	}
}

func TestEnemyAISystem_QuicksandSlowdown(t *testing.T) {
	w := newTestWorld()
	w.TimeOfDay = 5000 // Day! Prevent night buff multiplication
	// Set all tiles to GIDQuicksand
	for i := range w.Tiles {
		w.Tiles[i] = tile.GIDQuicksand
	}
	w.Layers = [][]int{w.Tiles}

	// Player is at (16, 16).
	// Place enemy at (30, 16), center (37, 22), distance 14 px from player (within aggro).
	enemy := world.NewEnemy(world.NoEntity, 30, 16, 10, false)
	enemy.AI.AggroRadius = 25
	enemy.AI.Speed = 2.0
	w.Enemies = []world.Enemy{enemy}

	var bus systems.EventBus
	ai := systems.EnemyAISystem{}
	if err := ai.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("EnemyAISystem failed: %v", err)
	}

	// Quicksand SpeedMultiplier is 0.05.
	// Path target is tile center (24, 24) from enemy center (37, 22):
	// dx = -13, dy = 2, dlen = sqrt(173) = 13.1529.
	// delta X = -13 / 13.1529 * (2.0 * 0.05) = -0.098837.
	// So enemy X should decrease from 30 to ~29.90116.
	expectedX := 29.90116
	if math.Abs(w.Enemies[0].X-expectedX) > 0.001 {
		t.Errorf("expected enemy X to be %f (0.05x speed multiplier), got %f", expectedX, w.Enemies[0].X)
	}
}

func TestHazardSystem_LavaDamageAndKnockback(t *testing.T) {
	w := newTestWorld()
	// Place player on a Lava tile (GIDLava)
	w.Tiles[0] = tile.GIDLava
	w.Layers = [][]int{w.Tiles}
	w.Player.X = 8
	w.Player.Y = 8
	w.Player.Invuln = 0

	var bus systems.EventBus
	hazardSys := systems.HazardSystem{}

	// Run hazard system
	if err := hazardSys.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("HazardSystem failed: %v", err)
	}

	// Player should have taken 1 damage (HP decreased from MaxHP to MaxHP - 1)
	expectedHP := w.Stats.MaxHP() - 1
	if w.HP != expectedHP {
		t.Errorf("expected player HP to be %d, got %d", expectedHP, w.HP)
	}

	// Player should have received Invuln frames equal to Lava's HazardInterval (30)
	if w.Player.Invuln != 30 {
		t.Errorf("expected player Invuln to be 30, got %d", w.Player.Invuln)
	}

	// Check that a PlayerHurtEvent was emitted
	events := bus.Drain()
	foundHurt := false
	for _, e := range events {
		if _, ok := e.(systems.PlayerHurtEvent); ok {
			foundHurt = true
			break
		}
	}
	if !foundHurt {
		t.Error("expected PlayerHurtEvent to be emitted")
	}
}

func TestShieldSystem_DirectionalBlock(t *testing.T) {
	// 1. Initialize a test world
	w := newTestWorld()
	// Let's grant the shield and set default shield percentages
	w.GrantItem("shield")
	w.ShieldLevel = 1
	w.Player.PlayerTuning.ShieldL1BlockPercent = 0.50
	w.Player.PlayerTuning.ShieldL2BlockPercent = 0.75

	// Player is at X: 16, Y: 16, W: 12, H: 12. Center: (22, 22). Facing: DirDown.
	w.Player.Dir = world.DirDown

	// 2. Spawn enemy directly in front (below player, e.g. at X: 16, Y: 24).
	// Enemy size: 12x12 (center 22, 30).
	// Vector from player to enemy is (0, 8), unit vector (0, 1).
	// Facing vector for DirDown is (0, 1). Dot product = 1 >= 0.5.
	// This is a block!
	enemyFront := world.NewEnemy(world.EntityID(10), 16, 24, 5, false)
	enemyFront.Damage = 4
	w.Enemies = []world.Enemy{enemyFront}

	bus := systems.EventBus{}
	enemyAISys := systems.EnemyAISystem{}

	// Player HP starts at MaxHP (which is 8 by default for Vitality 1: 6 + 2*1)
	initialHP := w.HP

	// Trigger collision (EnemyAISystem updates enemy and contact damage)
	if err := enemyAISys.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("EnemyAISystem failed: %v", err)
	}

	// Damage should be reduced by 50%: 4 -> 2. HP should be initialHP - 2
	expectedHP := initialHP - 2
	if w.HP != expectedHP {
		t.Errorf("expected player HP after block to be %d, got %d", expectedHP, w.HP)
	}

	// Verify PlayerHurtEvent has Blocked = true
	events := bus.Drain()
	var hurtEvent systems.PlayerHurtEvent
	found := false
	for _, ev := range events {
		if e, ok := ev.(systems.PlayerHurtEvent); ok {
			hurtEvent = e
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PlayerHurtEvent to be emitted")
	}
	if !hurtEvent.Blocked {
		t.Errorf("expected Blocked to be true, got %t", hurtEvent.Blocked)
	}
	if hurtEvent.Damage != 2 {
		t.Errorf("expected event Damage to be 2, got %d", hurtEvent.Damage)
	}

	// 3. Reset player HP, invuln, CD, position, change facing to DirUp (opposite of enemy).
	w.HP = initialHP
	w.Player.Invuln = 0
	w.Player.X, w.Player.Y = 16, 16
	w.Enemies[0].HurtCD = 0

	w.Player.Dir = world.DirUp // Face away from enemy below
	// Facing vector: (0, -1). Dot product with enemy direction: (0, 1) * (0, -1) = -1 < 0.5.
	// This should NOT block!

	if err := enemyAISys.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("EnemyAISystem failed: %v", err)
	}

	// Damage should NOT be reduced: full 4 damage. HP should be initialHP - 4
	expectedHP = initialHP - 4
	if w.HP != expectedHP {
		t.Errorf("expected player HP after back attack to be %d, got %d", expectedHP, w.HP)
	}

	events = bus.Drain()
	found = false
	for _, ev := range events {
		if e, ok := ev.(systems.PlayerHurtEvent); ok {
			hurtEvent = e
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected PlayerHurtEvent to be emitted")
	}
	if hurtEvent.Blocked {
		t.Errorf("expected Blocked to be false, got %t", hurtEvent.Blocked)
	}
	if hurtEvent.Damage != 4 {
		t.Errorf("expected event Damage to be 4, got %d", hurtEvent.Damage)
	}

	// 4. Upgrade shield to level 2 (Mirror Shield), reset player, hit from front.
	w.ShieldLevel = 2
	w.HP = initialHP
	w.Player.Invuln = 0
	w.Player.X, w.Player.Y = 16, 16
	w.Player.Dir = world.DirDown
	w.Enemies[0].HurtCD = 0

	if err := enemyAISys.Update(w, &bus, 1.0/60); err != nil {
		t.Fatalf("EnemyAISystem failed: %v", err)
	}

	// Level 2 reduces by 75%: 4 -> 1. HP should be initialHP - 1
	expectedHP = initialHP - 1
	if w.HP != expectedHP {
		t.Errorf("expected player HP after level 2 block to be %d, got %d", expectedHP, w.HP)
	}
}

