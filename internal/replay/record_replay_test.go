package replay_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/replay"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
)

// fakeInput is a stub services.Input driven by a scripted timeline. One
// entry per sim tick; each entry names which actions are "down" this
// tick. It exists here (not in internal/services) because making a fake
// is a test concern and keeping it local keeps services.Input from
// accreting a "default mock" that becomes the real API.
type fakeInput struct {
	script  [][]services.Action
	tick    int
	down    map[services.Action]bool
	prev    map[services.Action]bool
	cursorX int
	cursorY int
}

func newFakeInput(script [][]services.Action) *fakeInput {
	return &fakeInput{
		script: script,
		down:   map[services.Action]bool{},
		prev:   map[services.Action]bool{},
	}
}

// advance moves to the next scripted tick. Called once per sim tick at
// the top; mirrors Ebiten's "input state is stable within a tick" model.
func (f *fakeInput) advance() {
	f.prev = f.down
	f.down = map[services.Action]bool{}
	if f.tick < len(f.script) {
		for _, a := range f.script[f.tick] {
			f.down[a] = true
		}
	}
	f.tick++
}

func (f *fakeInput) IsDown(a services.Action) bool      { return f.down[a] }
func (f *fakeInput) JustPressed(a services.Action) bool { return f.down[a] && !f.prev[a] }
func (f *fakeInput) JustReleased(a services.Action) bool {
	return !f.down[a] && f.prev[a]
}
func (f *fakeInput) Axis2D() (int, int) {
	x, y := 0, 0
	if f.down[services.ActionMoveLeft] {
		x--
	}
	if f.down[services.ActionMoveRight] {
		x++
	}
	if f.down[services.ActionMoveUp] {
		y--
	}
	if f.down[services.ActionMoveDown] {
		y++
	}
	return x, y
}
func (f *fakeInput) IsModifierDown(services.Modifier) bool       { return false }
func (f *fakeInput) MousePressed(services.MouseButton) bool      { return false }
func (f *fakeInput) MouseJustPressed(services.MouseButton) bool  { return false }
func (f *fakeInput) MouseJustReleased(services.MouseButton) bool { return false }
func (f *fakeInput) CursorPosition() (int, int)                  { return f.cursorX, f.cursorY }

var _ services.Input = (*fakeInput)(nil)

// --- Test world helper ------------------------------------------------

// newReplayWorld builds a tiny 4×4 grass world with the player at
// (16,16) and a single enemy at (24,16) (1 HP). A sword swing facing
// right will land and emit a HitEvent on the right tick.
func newReplayWorld() *world.World {
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
			Facing:    world.Facing{Dir: world.DirRight},
			Stamina:   100,
		},
		Enemies: []world.Enemy{
			world.NewEnemy(world.NoEntity, 24, 16, 1, false),
		},
		Stats: progression.DefaultStats(),
		HP:    progression.DefaultStats().MaxHP(),
	}
}

// playTick is a miniature stand-in for PlayScene.Update that only
// handles the subset of actions we want to exercise in determinism
// tests: Attack (triggers a sword swing), and the 4-direction movement
// just for the Axis2D call. It's not meant to be complete — just enough
// to drive systems.Pipeline.
func playTick(in services.Input, w *world.World, pipe *systems.Pipeline) ([]systems.Event, error) {
	if in.JustPressed(services.ActionAttack) && w.Player.SwingCD == 0 {
		w.Player.Swing = 8
		w.Player.SwingCD = 12
	}
	return pipe.Tick(w, 1.0/60.0)
}

// --- Determinism tests ------------------------------------------------

// TestRecordPlaybackIsEventForEventIdentical is the end-to-end proof of
// the architecture: record a live (fake) input + run the pipeline,
// serialize to JSON, build a fresh world, play back through the same
// tick function, and assert the event trace is identical.
func TestRecordPlaybackIsEventForEventIdentical(t *testing.T) {
	script := [][]services.Action{
		{},                         // tick 0: idle
		{services.ActionAttack},    // tick 1: press attack
		{services.ActionAttack},    // tick 2: still holding (JustPressed should not refire)
		{},                         // tick 3: release
		{},                         // tick 4: idle
		{services.ActionMoveRight}, // tick 5: walk
		{services.ActionMoveRight}, // tick 6: walk
		{services.ActionAttack,
			services.ActionMoveRight}, // tick 7: attack again
	}

	// --- Recording pass ---
	src := newFakeInput(script)
	stream := replay.NewStream(60)
	rec := replay.NewRecorder(src, stream)
	recWorld := newReplayWorld()
	recPipe := systems.Default()

	var recTrace [][]systems.Event
	for i := 0; i < len(script); i++ {
		src.advance()
		rec.BeginTick()
		evs, err := playTick(rec, recWorld, recPipe)
		if err != nil {
			t.Fatalf("record tick %d: %v", i, err)
		}
		recTrace = append(recTrace, cloneEvents(evs))
	}
	if stream.Len() != len(script) {
		t.Fatalf("recorded %d frames, want %d", stream.Len(), len(script))
	}

	// --- Round-trip stream through JSON ---
	var buf bytes.Buffer
	if err := replay.WriteJSON(&buf, stream); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	rehydrated, err := replay.ReadJSON(&buf)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	// --- Playback pass on a fresh world ---
	playWorld := newReplayWorld()
	playPipe := systems.Default()
	pb := replay.NewPlayback(rehydrated)
	var playTrace [][]systems.Event
	for pb.Step() {
		evs, err := playTick(pb, playWorld, playPipe)
		if err != nil {
			t.Fatalf("playback tick: %v", err)
		}
		playTrace = append(playTrace, cloneEvents(evs))
	}

	// --- Assertions ---
	if len(recTrace) != len(playTrace) {
		t.Fatalf("tick count mismatch: rec=%d play=%d", len(recTrace), len(playTrace))
	}
	for i := range recTrace {
		if !reflect.DeepEqual(recTrace[i], playTrace[i]) {
			t.Errorf("tick %d event trace diverged\nrec:  %+v\nplay: %+v",
				i, recTrace[i], playTrace[i])
		}
	}
	if recWorld.Player.X != playWorld.Player.X || recWorld.Player.Y != playWorld.Player.Y {
		t.Errorf("player position diverged: rec=(%.1f,%.1f) play=(%.1f,%.1f)",
			recWorld.Player.X, recWorld.Player.Y, playWorld.Player.X, playWorld.Player.Y)
	}
	if recWorld.Enemies[0].HP != playWorld.Enemies[0].HP {
		t.Errorf("enemy HP diverged: rec=%d play=%d",
			recWorld.Enemies[0].HP, playWorld.Enemies[0].HP)
	}
}

// TestReplayIsStableAcrossMultipleRuns executes the same stream against
// the same starting world twice and confirms the results are identical.
// This is the stronger determinism guarantee: any non-deterministic
// behavior in systems (map iteration, time.Now, rand) would surface here.
func TestReplayIsStableAcrossMultipleRuns(t *testing.T) {
	script := [][]services.Action{
		{}, {services.ActionAttack}, {services.ActionAttack}, {}, {},
	}
	stream := replay.NewStream(60)
	src := newFakeInput(script)
	rec := replay.NewRecorder(src, stream)
	for range script {
		src.advance()
		rec.BeginTick()
	}

	runOnce := func() ([][]systems.Event, *world.World) {
		w := newReplayWorld()
		pipe := systems.Default()
		pb := replay.NewPlayback(stream)
		var trace [][]systems.Event
		for pb.Step() {
			evs, err := playTick(pb, w, pipe)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			trace = append(trace, cloneEvents(evs))
		}
		return trace, w
	}

	traceA, worldA := runOnce()
	traceB, worldB := runOnce()

	if !reflect.DeepEqual(traceA, traceB) {
		t.Error("two replays produced different event traces (non-determinism)")
	}
	if worldA.Enemies[0].HP != worldB.Enemies[0].HP {
		t.Errorf("two replays produced different enemy HP: %d vs %d",
			worldA.Enemies[0].HP, worldB.Enemies[0].HP)
	}
}

// cloneEvents copies a slice returned by Pipeline.Tick. Pipeline
// internally reuses its EventBus buffer, so the returned slice is only
// valid until the next Tick; we clone to pin the snapshot.
func cloneEvents(src []systems.Event) []systems.Event {
	if len(src) == 0 {
		return nil
	}
	dst := make([]systems.Event, len(src))
	copy(dst, src)
	return dst
}
