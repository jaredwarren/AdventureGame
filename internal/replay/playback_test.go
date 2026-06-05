package replay_test

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/replay"
	"github.com/jaredwarren/game-test/internal/services"
)

// TestPlaybackInitialStateIsZero confirms that before the first Step,
// every polling method returns a safe zero: no key down, no cursor.
// Important because scenes poll immediately on construction.
func TestPlaybackInitialStateIsZero(t *testing.T) {
	pb := replay.NewPlayback(replay.NewStream(60))
	if pb.IsDown(services.ActionAttack) {
		t.Error("IsDown on fresh Playback should be false")
	}
	if pb.JustPressed(services.ActionAttack) {
		t.Error("JustPressed on fresh Playback should be false")
	}
	x, y := pb.CursorPosition()
	if x != 0 || y != 0 {
		t.Errorf("CursorPosition on fresh Playback = (%d,%d), want (0,0)", x, y)
	}
}

// TestPlaybackStepExhaustion validates the iteration contract used by
// callers: "for pb.Step() { ... }" terminates cleanly and further polls
// continue to return zero.
func TestPlaybackStepExhaustion(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{Down: []string{"Attack"}})
	pb := replay.NewPlayback(s)

	if !pb.Step() {
		t.Fatal("Step 1 returned false, want true")
	}
	if !pb.IsDown(services.ActionAttack) {
		t.Error("expected Attack down after Step 1")
	}
	if pb.Step() {
		t.Fatal("Step 2 returned true, want false (stream has one frame)")
	}
	// After exhaustion, polls should return false, not panic.
	if pb.IsDown(services.ActionAttack) {
		t.Error("IsDown after exhaustion should be false")
	}
	if pb.Remaining() != 0 {
		t.Errorf("Remaining=%d, want 0", pb.Remaining())
	}
}

// TestPlaybackEdgeDetection is the core-correctness test: JustPressed
// and JustReleased must be derivable from the delta between consecutive
// frames, matching inpututil's semantics.
func TestPlaybackEdgeDetection(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{})                         // frame 1: nothing down
	s.Append(replay.Frame{Down: []string{"Attack"}}) // frame 2: attack pressed
	s.Append(replay.Frame{Down: []string{"Attack"}}) // frame 3: still held
	s.Append(replay.Frame{})                         // frame 4: released
	pb := replay.NewPlayback(s)

	// Frame 1: nothing.
	pb.Step()
	if pb.IsDown(services.ActionAttack) || pb.JustPressed(services.ActionAttack) {
		t.Errorf("frame 1: Attack unexpectedly hot")
	}

	// Frame 2: edge-triggered press.
	pb.Step()
	if !pb.IsDown(services.ActionAttack) {
		t.Error("frame 2: IsDown(Attack)=false, want true")
	}
	if !pb.JustPressed(services.ActionAttack) {
		t.Error("frame 2: JustPressed(Attack)=false, want true")
	}
	if pb.JustReleased(services.ActionAttack) {
		t.Error("frame 2: JustReleased unexpectedly true")
	}

	// Frame 3: still held — JustPressed must be false now.
	pb.Step()
	if !pb.IsDown(services.ActionAttack) {
		t.Error("frame 3: IsDown(Attack)=false, want true")
	}
	if pb.JustPressed(services.ActionAttack) {
		t.Error("frame 3: JustPressed should have cleared")
	}

	// Frame 4: released.
	pb.Step()
	if pb.IsDown(services.ActionAttack) {
		t.Error("frame 4: IsDown(Attack) should be false")
	}
	if !pb.JustReleased(services.ActionAttack) {
		t.Error("frame 4: JustReleased(Attack)=false, want true")
	}
}

// TestPlaybackReset returns the cursor to the start of the stream so
// tests can run the same stream twice.
func TestPlaybackReset(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{Down: []string{"Attack"}})
	pb := replay.NewPlayback(s)

	pb.Step()
	if !pb.IsDown(services.ActionAttack) {
		t.Fatal("setup: first Step didn't load Attack frame")
	}
	pb.Reset()
	if pb.IsDown(services.ActionAttack) {
		t.Error("after Reset, IsDown should be false until next Step")
	}
	if !pb.Step() {
		t.Error("Step after Reset should succeed")
	}
	if !pb.IsDown(services.ActionAttack) {
		t.Error("post-Reset first Step should load frame 1 again")
	}
}

// TestPlaybackAxis2D is a tiny smoke test for the derived axis helper.
// The helper is identical on Recorder/Playback because Axis2D is a
// function of IsDown only.
func TestPlaybackAxis2D(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{Down: []string{"MoveRight", "MoveDown"}})
	pb := replay.NewPlayback(s)
	pb.Step()
	x, y := pb.Axis2D()
	if x != 1 || y != 1 {
		t.Errorf("Axis2D=(%d,%d), want (1,1) for Right+Down", x, y)
	}
}

// TestPlaybackModifiersAndMouse exercises the non-action channels.
func TestPlaybackModifiersAndMouse(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{
		Modifiers: []string{"Ctrl"},
		Mouse:     []string{"Left"},
		CursorX:   42,
		CursorY:   7,
	})
	pb := replay.NewPlayback(s)
	pb.Step()
	if !pb.IsModifierDown(services.ModCtrl) {
		t.Error("Ctrl not reported down")
	}
	if pb.IsModifierDown(services.ModShift) {
		t.Error("Shift spuriously reported down")
	}
	if !pb.MousePressed(services.MouseLeft) {
		t.Error("MouseLeft not reported pressed")
	}
	if !pb.MouseJustPressed(services.MouseLeft) {
		t.Error("MouseLeft not reported just-pressed on edge")
	}
	if x, y := pb.CursorPosition(); x != 42 || y != 7 {
		t.Errorf("CursorPosition=(%d,%d), want (42,7)", x, y)
	}
}

// TestPlaybackIgnoresUnknownNames — a stream written by a newer schema
// might contain action/modifier names this build does not understand.
// Those names should be silently ignored, not panic, not match real
// actions. This is what makes forward-compat upgrades safe.
func TestPlaybackIgnoresUnknownNames(t *testing.T) {
	s := replay.NewStream(60)
	s.Append(replay.Frame{
		Down:      []string{"Attack", "FutureActionFromV2"},
		Modifiers: []string{"Hyper"},
		Mouse:     []string{"Back"},
	})
	pb := replay.NewPlayback(s)
	pb.Step()
	if !pb.IsDown(services.ActionAttack) {
		t.Error("known action dropped alongside unknown")
	}
}
