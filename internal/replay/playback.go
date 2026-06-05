// playback.go — services.Input backed by a prerecorded Stream.
//
// Lifecycle
//
//	pb := replay.NewPlayback(stream)
//	for pb.Step() {
//	    // call scene.Update / pipeline.Tick reading pb as services.Input
//	}
//	// pb.Step() returned false -> stream exhausted; further polls
//	// return zero values.
//
// Edges (JustPressed / JustReleased) are computed from the diff of the
// current and previous frames. Before the first Step, both sides of the
// diff are empty, so JustPressed for the first frame reports "down now
// and not down before" correctly, matching the behavior of the live
// input on the first tick after boot.
//
// Cursor position and mouse state are served from the current frame
// verbatim. If a frame omits CursorX/CursorY, Playback returns (0, 0);
// callers that depend on the cursor must confirm the frames carry it.
package replay

import (
	"github.com/jaredwarren/game-test/internal/input"
	"github.com/jaredwarren/game-test/internal/services"
)

// Playback turns a Stream into a services.Input. Zero-value is invalid;
// use NewPlayback.
type Playback struct {
	stream *Stream

	// idx is the index of the NEXT frame to consume. Before the first
	// Step, idx == 0 and both prev + cur are empty (see lookup helpers).
	idx int

	// prev / cur hold precomputed membership sets for the frames on
	// either side of the current tick boundary. These are what the
	// lookup methods query — rebuilt once per Step rather than per poll.
	prev, cur frameView
}

// NewPlayback constructs a Playback over stream. The stream is not
// copied; callers should not mutate it during playback.
func NewPlayback(stream *Stream) *Playback {
	return &Playback{stream: stream}
}

// Step advances to the next recorded frame. Returns false when the
// stream is exhausted. On exhaustion both prev and cur are zeroed so
// that subsequent polls return safe defaults — IsDown is false, cursor
// is (0,0), etc. This lets "for pb.Step()" terminate cleanly without
// stale state bleeding into callers that poll post-loop.
func (p *Playback) Step() bool {
	if p == nil || p.stream == nil || p.idx >= len(p.stream.Frames) {
		if p != nil {
			p.prev = frameView{}
			p.cur = frameView{}
		}
		return false
	}
	p.prev = p.cur
	p.cur = buildFrameView(p.stream.Frames[p.idx])
	p.idx++
	return true
}

// Reset rewinds the playback to its initial state. Useful in tests that
// want to replay the same stream twice against a freshly-built world.
func (p *Playback) Reset() {
	p.idx = 0
	p.prev = frameView{}
	p.cur = frameView{}
}

// Remaining reports how many frames have not yet been consumed by Step.
func (p *Playback) Remaining() int {
	if p == nil || p.stream == nil {
		return 0
	}
	return len(p.stream.Frames) - p.idx
}

// --- services.Input implementation -----------------------------------

func (p *Playback) IsDown(a services.Action) bool { return p.cur.hasAction(a) }
func (p *Playback) JustPressed(a services.Action) bool {
	return p.cur.hasAction(a) && !p.prev.hasAction(a)
}
func (p *Playback) JustReleased(a services.Action) bool {
	return !p.cur.hasAction(a) && p.prev.hasAction(a)
}

func (p *Playback) Axis2D() (x, y int) {
	if p.IsDown(services.ActionMoveLeft) {
		x--
	}
	if p.IsDown(services.ActionMoveRight) {
		x++
	}
	if p.IsDown(services.ActionMoveUp) {
		y--
	}
	if p.IsDown(services.ActionMoveDown) {
		y++
	}
	return
}

func (p *Playback) IsModifierDown(m services.Modifier) bool { return p.cur.hasModifier(m) }

func (p *Playback) MousePressed(b services.MouseButton) bool {
	return p.cur.hasMouse(b)
}
func (p *Playback) MouseJustPressed(b services.MouseButton) bool {
	return p.cur.hasMouse(b) && !p.prev.hasMouse(b)
}
func (p *Playback) MouseJustReleased(b services.MouseButton) bool {
	return !p.cur.hasMouse(b) && p.prev.hasMouse(b)
}

func (p *Playback) CursorPosition() (int, int) { return p.cur.cursorX, p.cur.cursorY }

var _ services.Input = (*Playback)(nil)

// --- internal frameView -----------------------------------------------
//
// buildFrameView resolves the Frame's string names into fast-lookup
// structures (maps keyed by the relevant enum type). Done once per Step
// so a tick's worth of polls doesn't re-scan string slices.

type frameView struct {
	actions   map[services.Action]bool
	modifiers map[services.Modifier]bool
	mouse     map[services.MouseButton]bool
	cursorX   int
	cursorY   int
}

func buildFrameView(f Frame) frameView {
	v := frameView{
		actions:   make(map[services.Action]bool, len(f.Down)),
		modifiers: make(map[services.Modifier]bool, len(f.Modifiers)),
		mouse:     make(map[services.MouseButton]bool, len(f.Mouse)),
		cursorX:   f.CursorX,
		cursorY:   f.CursorY,
	}
	for _, name := range f.Down {
		if a, ok := input.ActionFromName(name); ok {
			v.actions[a] = true
		}
	}
	for _, name := range f.Modifiers {
		if m, ok := modifierFromName(name); ok {
			v.modifiers[m] = true
		}
	}
	for _, name := range f.Mouse {
		if b, ok := mouseFromName(name); ok {
			v.mouse[b] = true
		}
	}
	return v
}

func (v *frameView) hasAction(a services.Action) bool     { return v.actions[a] }
func (v *frameView) hasModifier(m services.Modifier) bool { return v.modifiers[m] }
func (v *frameView) hasMouse(b services.MouseButton) bool { return v.mouse[b] }
