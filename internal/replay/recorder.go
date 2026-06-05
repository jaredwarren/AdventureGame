// recorder.go — live-input sampler that appends Frames to a Stream.
//
// Typical usage in a hypothetical record scene:
//
//	rec := replay.NewRecorder(ctx.Input, stream)
//	ctx.Input = rec         // every read now goes through the recorder
//	for each tick {
//	    rec.BeginTick()     // snapshot; adds one frame to the stream
//	    ...scene.Update(ctx)
//	}
//
// BeginTick samples every registered action and modifier by polling the
// wrapped Input once per bit, so sampling cost is O(|Actions|) per tick
// (~30 polls on the current enum) — negligible at 60 Hz.
//
// All services.Input methods pass through to the wrapped implementation:
// the recorder is a passive observer, never substituting or modifying
// behavior mid-record. That property is what makes record-then-replay
// produce bit-identical event traces.
package replay

import (
	"sort"

	"github.com/jaredwarren/game-test/internal/input"
	"github.com/jaredwarren/game-test/internal/services"
)

// Recorder wraps a live services.Input, implements services.Input by
// pass-through, and accumulates a frame per BeginTick.
type Recorder struct {
	src    services.Input
	stream *Stream
}

// NewRecorder returns a Recorder that writes frames into stream. The
// stream's frame buffer is appended to in place; callers may continue
// to hold the *Stream and inspect / marshal it.
func NewRecorder(src services.Input, stream *Stream) *Recorder {
	return &Recorder{src: src, stream: stream}
}

// BeginTick samples the wrapped input and appends one Frame to the
// stream. Call exactly once per simulation tick, at the very top of the
// tick before any scene reads input.
func (r *Recorder) BeginTick() {
	f := Frame{}
	for _, a := range input.AllActions() {
		if r.src.IsDown(a) {
			f.Down = append(f.Down, input.ActionName(a))
		}
	}
	for _, m := range allModifiers {
		if r.src.IsModifierDown(m) {
			f.Modifiers = append(f.Modifiers, modifierName(m))
		}
	}
	for _, b := range allMouseButtons {
		if r.src.MousePressed(b) {
			f.Mouse = append(f.Mouse, mouseName(b))
		}
	}
	f.CursorX, f.CursorY = r.src.CursorPosition()

	// Deterministic ordering inside a frame — AllActions() ordering is
	// map-iteration, which Go does not guarantee between runs. Without
	// sorting, the same input produces different-looking JSON each run,
	// which wrecks golden-file diffs.
	sort.Strings(f.Down)
	sort.Strings(f.Modifiers)
	sort.Strings(f.Mouse)

	r.stream.Frames = append(r.stream.Frames, f)
}

// Stream returns the accumulating stream. Safe to call at any time;
// appending to the returned pointer is what BeginTick does.
func (r *Recorder) Stream() *Stream { return r.stream }

// --- services.Input pass-through -------------------------------------

func (r *Recorder) IsDown(a services.Action) bool       { return r.src.IsDown(a) }
func (r *Recorder) JustPressed(a services.Action) bool  { return r.src.JustPressed(a) }
func (r *Recorder) JustReleased(a services.Action) bool { return r.src.JustReleased(a) }
func (r *Recorder) Axis2D() (int, int)                  { return r.src.Axis2D() }
func (r *Recorder) IsModifierDown(m services.Modifier) bool {
	return r.src.IsModifierDown(m)
}
func (r *Recorder) MousePressed(b services.MouseButton) bool {
	return r.src.MousePressed(b)
}
func (r *Recorder) MouseJustPressed(b services.MouseButton) bool {
	return r.src.MouseJustPressed(b)
}
func (r *Recorder) MouseJustReleased(b services.MouseButton) bool {
	return r.src.MouseJustReleased(b)
}
func (r *Recorder) CursorPosition() (int, int) { return r.src.CursorPosition() }

var _ services.Input = (*Recorder)(nil)

// --- Modifier / Mouse name tables -------------------------------------
//
// Unlike Actions (where input.ActionName is the canonical map),
// Modifier and MouseButton are small enums with a fixed population.
// Keeping the table local to the replay package means we don't need to
// widen the internal/input API surface just for replay.

var allModifiers = []services.Modifier{
	services.ModCtrl,
	services.ModShift,
	services.ModAlt,
}

func modifierName(m services.Modifier) string {
	switch m {
	case services.ModCtrl:
		return "Ctrl"
	case services.ModShift:
		return "Shift"
	case services.ModAlt:
		return "Alt"
	}
	return ""
}

func modifierFromName(n string) (services.Modifier, bool) {
	switch n {
	case "Ctrl":
		return services.ModCtrl, true
	case "Shift":
		return services.ModShift, true
	case "Alt":
		return services.ModAlt, true
	}
	return 0, false
}

var allMouseButtons = []services.MouseButton{
	services.MouseLeft,
	services.MouseMiddle,
	services.MouseRight,
}

func mouseName(b services.MouseButton) string {
	switch b {
	case services.MouseLeft:
		return "Left"
	case services.MouseMiddle:
		return "Middle"
	case services.MouseRight:
		return "Right"
	}
	return ""
}

func mouseFromName(n string) (services.MouseButton, bool) {
	switch n {
	case "Left":
		return services.MouseLeft, true
	case "Middle":
		return services.MouseMiddle, true
	case "Right":
		return services.MouseRight, true
	}
	return 0, false
}
