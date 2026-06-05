// Package services defines backend-agnostic interfaces ("ports") used by scenes
// and systems. Concrete implementations live under internal/platform/<backend>.
//
// Rationale: keep gameplay code free of ebiten/inpututil imports so simulation
// is testable headlessly and input/rendering backends can be swapped.
package services

// Action is a logical player intent. The InputManager maps physical keys and
// gamepad buttons to Actions. Scenes and systems read Actions, not keys.
type Action int

const (
	ActionNone Action = iota

	// Movement axis (held).
	ActionMoveUp
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight

	// Combat / traversal (edge-triggered unless noted).
	ActionAttack
	ActionBomb
	ActionTorch
	ActionDodge
	ActionSprint // HELD, not edge-triggered.
	ActionInteract
	ActionItemMenu // opens / closes item selection overlay

	// Meta / UI.
	ActionPause
	ActionConfirm
	ActionCancel

	// Quick-save / load. Chorded behavior lives in the scene via IsModifierDown.
	ActionQuickSave
	ActionQuickLoad
	ActionCopyBugDigest

	// Debug.
	ActionDebugToggle
	ActionToggleReduceShake

	// Editor-specific.
	ActionEditorToggleMode
	ActionEditorAdd
	ActionEditorDelete
	ActionEditorNextType
	ActionEditorPrevType
	ActionEditorBrush1
	ActionEditorBrush2
	ActionEditorBrush3
	ActionEditorBrush4
	ActionEditorBrush5
	ActionEditorBrush6
	ActionEditorBrush7
	ActionEditorBrush8
	ActionEditorBrushClear
)

// Modifier is a held modifier key (Ctrl/Shift/Alt). Kept separate from Action
// so that chord detection is explicit at the call site.
type Modifier int

const (
	ModCtrl Modifier = iota
	ModShift
	ModAlt
)

// MouseButton abstracts pointer buttons for the editor.
type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseRight
	MouseMiddle
)

// Input is the only input surface scenes and systems should use.
//
// Invariants:
//   - Axis2D returns raw integer direction in {-1, 0, 1}; diagonals are not
//     normalized. Callers that need unit-length vectors normalize themselves.
//   - JustPressed/JustReleased are edge-triggered for the current tick.
//   - CursorPosition returns logical (layout) pixels, not window pixels.
type Input interface {
	IsDown(a Action) bool
	JustPressed(a Action) bool
	JustReleased(a Action) bool

	Axis2D() (x, y int)

	IsModifierDown(m Modifier) bool

	MousePressed(b MouseButton) bool
	MouseJustPressed(b MouseButton) bool
	MouseJustReleased(b MouseButton) bool
	CursorPosition() (x, y int)
}
