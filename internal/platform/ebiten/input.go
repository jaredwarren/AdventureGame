// Package ebitenplat provides Ebiten-backed implementations of internal/services.
//
// This is the ONLY package (besides cmd/game and internal/game draw code) that
// imports github.com/hajimehoshi/ebiten/v2. Scenes and systems depend on the
// interfaces in internal/services instead.
package ebitenplat

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/jaredwarren/game-test/internal/input"
	"github.com/jaredwarren/game-test/internal/services"
)

// Input implements services.Input on top of ebiten + inpututil. Edge detection
// is delegated to inpututil; no internal state is required.
//
// The down map is built at construction from a resolved input.Bindings
// table (see NewInput). Modifiers and mouse buttons stay hardcoded because
// they're not user-rebindable today — Ctrl is always Ctrl.
type Input struct {
	down        map[services.Action][]ebiten.Key
	modifiers   map[services.Modifier][]ebiten.Key
	mouseButton map[services.MouseButton]ebiten.MouseButton
	touch       *TouchControls
}

// InputOptions configures optional touch overlay behavior.
type InputOptions struct {
	TouchControls bool
}

// NewInput resolves the symbolic key names in b against keynames.go and
// builds an Input ready for per-frame polling. Key names that don't
// resolve produce warnings; the Action keeps whatever keys did resolve.
// An action with no resolvable keys still exists in the map but its
// polling methods always return false.
//
// The warnings slice is intended for main() to log once; callers free to
// ignore. NewInput itself never returns an error — a bad config always
// degrades, never blocks launch.
func NewInput(b *input.Bindings) (*Input, []string) {
	return NewInputWithOptions(b, InputOptions{})
}

// NewInputWithOptions is like NewInput but can enable the on-screen touch
// overlay used for WASM and future mobile builds.
func NewInputWithOptions(b *input.Bindings, opts InputOptions) (*Input, []string) {
	if b == nil {
		b = input.DefaultBindings()
	}
	var warnings []string

	// Every named Action gets an entry (possibly empty) so IsDown is
	// total over the Action enum. Actions missing from b end up empty.
	down := make(map[services.Action][]ebiten.Key, len(b.Actions))
	for name, keyNames := range b.Actions {
		action, ok := input.ActionFromName(name)
		if !ok {
			// Already reported by input.Bindings.Validate(); don't dup.
			continue
		}
		keys := make([]ebiten.Key, 0, len(keyNames))
		for _, kn := range keyNames {
			k, ok := keyByName(kn)
			if !ok {
				warnings = append(warnings, fmt.Sprintf(
					"action %q bound to unknown key %q (ignored)", name, kn))
				continue
			}
			keys = append(keys, k)
		}
		down[action] = keys
	}

	return &Input{
		down:      down,
		modifiers: defaultModifierBindings(),
		mouseButton: map[services.MouseButton]ebiten.MouseButton{
			services.MouseLeft:   ebiten.MouseButtonLeft,
			services.MouseRight:  ebiten.MouseButtonRight,
			services.MouseMiddle: ebiten.MouseButtonMiddle,
		},
		touch: NewTouchControls(opts.TouchControls),
	}, warnings
}

// BeginFrame samples touch overlay state for the current tick. App calls
// this at the top of Update before scenes read input.
func (i *Input) BeginFrame() {
	if i.touch != nil {
		i.touch.BeginFrame()
	}
}

// TouchEnabled reports whether the on-screen overlay is active.
func (i *Input) TouchEnabled() bool {
	return i.touch != nil && i.touch.Enabled()
}

// DrawTouchControls renders the virtual button overlay. Call after scene
// Draw while the renderer frame is still open.
func (i *Input) DrawTouchControls(r *Renderer) {
	if i.touch != nil {
		i.touch.Draw(r)
	}
}

// defaultModifierBindings returns the immutable mapping from logical
// Modifier to the physical keys that trigger it. Not user-rebindable:
// Ctrl / Shift / Alt are OS-level concepts and the game would misbehave
// if a user remapped "Shift" to the F1 key.
func defaultModifierBindings() map[services.Modifier][]ebiten.Key {
	return map[services.Modifier][]ebiten.Key{
		services.ModCtrl:  {ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight},
		services.ModShift: {ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
		services.ModAlt:   {ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight},
	}
}

func (i *Input) IsDown(a services.Action) bool {
	if i.touch != nil && i.touch.IsDown(a) {
		return true
	}
	for _, k := range i.down[a] {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func (i *Input) JustPressed(a services.Action) bool {
	if i.touch != nil && i.touch.JustPressed(a) {
		return true
	}
	for _, k := range i.down[a] {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func (i *Input) JustReleased(a services.Action) bool {
	if i.touch != nil && i.touch.JustReleased(a) {
		return true
	}
	for _, k := range i.down[a] {
		if inpututil.IsKeyJustReleased(k) {
			return true
		}
	}
	return false
}

// Axis2D returns raw direction in {-1,0,1}; diagonals are not normalized here.
// Callers that need unit-length vectors do the 1/√2 scale themselves.
func (i *Input) Axis2D() (x, y int) {
	if i.IsDown(services.ActionMoveLeft) {
		x--
	}
	if i.IsDown(services.ActionMoveRight) {
		x++
	}
	if i.IsDown(services.ActionMoveUp) {
		y--
	}
	if i.IsDown(services.ActionMoveDown) {
		y++
	}
	return
}

func (i *Input) IsModifierDown(m services.Modifier) bool {
	for _, k := range i.modifiers[m] {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func (i *Input) MousePressed(b services.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(i.mouseButton[b])
}

func (i *Input) MouseJustPressed(b services.MouseButton) bool {
	return inpututil.IsMouseButtonJustPressed(i.mouseButton[b])
}

func (i *Input) MouseJustReleased(b services.MouseButton) bool {
	return inpututil.IsMouseButtonJustReleased(i.mouseButton[b])
}

func (i *Input) CursorPosition() (int, int) {
	return ebiten.CursorPosition()
}

// Compile-time interface check.
var _ services.Input = (*Input)(nil)
