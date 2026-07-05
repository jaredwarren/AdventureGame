// names.go — the canonical Action ↔ string table.
//
// Every services.Action enum value (except ActionNone) must appear here.
// The strings are what lands in the JSON config file, so:
//
//   - Stable across enum reordering (iota renumbers would not break files).
//   - Human-readable without the "Action" prefix ("MoveUp", not
//     "ActionMoveUp").
//   - The zero value (ActionNone) is deliberately omitted: scenes never
//     look it up, and binding a key to "nothing" is meaningless.
//
// If a new Action is added to services without an entry here, the
// TestActionNamesCoversAllActions test will fail. Add the name at the
// same time as the enum value.
package input

import "github.com/jaredwarren/game-test/internal/services"

// actionNames is the forward map (enum → serialized name). Source of truth
// for marshaling. The reverse map is built lazily below.
var actionNames = map[services.Action]string{
	services.ActionMoveUp:    "MoveUp",
	services.ActionMoveDown:  "MoveDown",
	services.ActionMoveLeft:  "MoveLeft",
	services.ActionMoveRight: "MoveRight",

	services.ActionAttack:   "Attack",
	services.ActionBomb:     "Bomb",
	services.ActionTorch:    "Torch",
	services.ActionDodge:    "Dodge",
	services.ActionSprint:   "Sprint",
	services.ActionInteract: "Interact",
	services.ActionItemMenu: "ItemMenu",

	services.ActionPause:   "Pause",
	services.ActionConfirm: "Confirm",
	services.ActionCancel:  "Cancel",

	services.ActionQuickSave:     "QuickSave",
	services.ActionQuickLoad:     "QuickLoad",
	services.ActionCopyBugDigest: "CopyBugDigest",

	services.ActionDebugToggle:       "DebugToggle",
	services.ActionToggleReduceShake: "ToggleReduceShake",

	services.ActionEditorToggleMode:       "EditorToggleMode",
	services.ActionEditorToggleLayer:      "EditorToggleLayer",
	services.ActionEditorToggleVisibility: "EditorToggleVisibility",
	services.ActionEditorAdd:         "EditorAdd",
	services.ActionEditorDelete:     "EditorDelete",
	services.ActionEditorNextType:   "EditorNextType",
	services.ActionEditorPrevType:   "EditorPrevType",
	services.ActionEditorBrush1:     "EditorBrush1",
	services.ActionEditorBrush2:     "EditorBrush2",
	services.ActionEditorBrush3:     "EditorBrush3",
	services.ActionEditorBrush4:     "EditorBrush4",
	services.ActionEditorBrush5:     "EditorBrush5",
	services.ActionEditorBrush6:     "EditorBrush6",
	services.ActionEditorBrush7:     "EditorBrush7",
	services.ActionEditorBrush8:     "EditorBrush8",
	services.ActionEditorBrushClear: "EditorBrushClear",
	services.ActionEditorTileMenu:   "EditorTileMenu",
}

// actionByName is the reverse map (serialized name → enum). Initialized
// once at package load; not safe to mutate after init.
var actionByName = func() map[string]services.Action {
	m := make(map[string]services.Action, len(actionNames))
	for a, n := range actionNames {
		m[n] = a
	}
	return m
}()

// ActionName returns the canonical string for a. Returns the empty string
// for ActionNone or any unregistered enum value; callers should treat
// that as "not bindable".
func ActionName(a services.Action) string { return actionNames[a] }

// ActionFromName returns the Action for the given canonical name.
// ok is false when the name is unknown (unknown names should be logged
// as warnings, not silently coerced).
func ActionFromName(name string) (a services.Action, ok bool) {
	a, ok = actionByName[name]
	return
}

// AllActionNames returns every registered action name. Primarily for
// tests and for LoadOrInit when writing the defaults file.
func AllActionNames() []string {
	out := make([]string, 0, len(actionNames))
	for _, n := range actionNames {
		out = append(out, n)
	}
	return out
}

// AllActions returns every registered services.Action value. Callers who
// need to iterate the enum (e.g. input-replay snapshot sampling) should
// use this rather than hardcoding [ActionNone+1 .. ActionLast], which
// silently drifts when the enum grows.
//
// Order is unspecified; if callers need deterministic ordering, sort by
// ActionName themselves.
func AllActions() []services.Action {
	out := make([]services.Action, 0, len(actionNames))
	for a := range actionNames {
		out = append(out, a)
	}
	return out
}
