// bindings.go — the user-facing binding table and its JSON shape.
//
// Bindings are a simple {actionName → []keyName} map plus a schema
// version. This package does not resolve keyName to a backend key value;
// that's a platform concern (internal/platform/ebiten/keynames.go).
//
// # Merge semantics
//
// LoadOrInit uses Merge to combine the user's file with the built-in
// defaults. The rule is "user wins per action": if the file names an
// action, its key list replaces the default entirely (NOT unioned). An
// empty list explicitly unbinds the action. Actions absent from the
// file fall back to defaults.
//
// This matches user expectation for rebinding UIs: "I pressed Q for
// Attack, stop giving me Z as well unless I added Z back". Unioning
// would silently keep the old binding alive — surprising and bad.
package input

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jaredwarren/game-test/internal/services"
)

// SchemaVersion is the on-disk format version. Bump when a migration is
// required; readers still tolerate files with an older version and apply
// their migration path (future).
const SchemaVersion = 1

// Bindings is the in-memory shape of the keybinds JSON file. Exported
// fields carry the JSON tags for marshaling.
//
// Action keys use the canonical names from names.go ("MoveUp" etc).
// Key values are backend-neutral strings resolved by the platform layer.
type Bindings struct {
	Version int                 `json:"version"`
	Actions map[string][]string `json:"actions"`
}

// DefaultBindings returns the factory defaults — the same bindings that
// ship with the game. Callers mutate the returned value freely; it is
// not shared.
func DefaultBindings() *Bindings {
	return &Bindings{
		Version: SchemaVersion,
		Actions: map[string][]string{
			ActionName(services.ActionMoveUp):    {"ArrowUp", "W"},
			ActionName(services.ActionMoveDown):  {"ArrowDown", "S"},
			ActionName(services.ActionMoveLeft):  {"ArrowLeft", "A"},
			ActionName(services.ActionMoveRight): {"ArrowRight", "D"},

			ActionName(services.ActionAttack):   {"Z"},
			ActionName(services.ActionBomb):     {"X"},
			ActionName(services.ActionTorch):    {"T"},
			ActionName(services.ActionDodge):    {"Alt"},
			ActionName(services.ActionSprint):   {"Shift"},
			ActionName(services.ActionInteract): {"E"},
			ActionName(services.ActionItemMenu): {"Tab"},

			ActionName(services.ActionPause):   {"P"},
			ActionName(services.ActionConfirm): {"Enter"},
			ActionName(services.ActionCancel):  {"Escape"},

			ActionName(services.ActionQuickSave):     {"S"},
			ActionName(services.ActionQuickLoad):     {"L", "K"},
			ActionName(services.ActionCopyBugDigest): {"C"},

			ActionName(services.ActionDebugToggle):       {"F3"},
			ActionName(services.ActionToggleReduceShake): {"R"},

			ActionName(services.ActionEditorToggleMode):       {"E"},
			ActionName(services.ActionEditorToggleLayer):      {"L"},
			ActionName(services.ActionEditorToggleVisibility): {"V"},
			ActionName(services.ActionEditorTileMenu):    {"Tab"},
			ActionName(services.ActionEditorAdd):        {"A"},
			ActionName(services.ActionEditorDelete):     {"Delete"},
			ActionName(services.ActionEditorNextType):   {"BracketRight"},
			ActionName(services.ActionEditorPrevType):   {"BracketLeft"},
			ActionName(services.ActionEditorBrush1):     {"Digit1"},
			ActionName(services.ActionEditorBrush2):     {"Digit2"},
			ActionName(services.ActionEditorBrush3):     {"Digit3"},
			ActionName(services.ActionEditorBrush4):     {"Digit4"},
			ActionName(services.ActionEditorBrush5):     {"Digit5"},
			ActionName(services.ActionEditorBrush6):     {"Digit6"},
			ActionName(services.ActionEditorBrush7):     {"Digit7"},
			ActionName(services.ActionEditorBrush8):     {"Digit8"},
			ActionName(services.ActionEditorBrushClear): {"Digit0"},
		},
	}
}

// Clone returns a deep copy. Safe to modify independently.
func (b *Bindings) Clone() *Bindings {
	if b == nil {
		return nil
	}
	out := &Bindings{Version: b.Version, Actions: make(map[string][]string, len(b.Actions))}
	for k, v := range b.Actions {
		cpy := make([]string, len(v))
		copy(cpy, v)
		out.Actions[k] = cpy
	}
	return out
}

// Merge returns a new Bindings where every action present in other
// overrides the corresponding entry in b, and actions absent from other
// fall back to b. Neither receiver is mutated.
//
// An empty key list in other is preserved as "explicitly unbound" — the
// caller decided to zero this action out.
func (b *Bindings) Merge(other *Bindings) *Bindings {
	out := b.Clone()
	if other == nil {
		return out
	}
	if other.Version > 0 {
		out.Version = other.Version
	}
	for action, keys := range other.Actions {
		cpy := make([]string, len(keys))
		copy(cpy, keys)
		out.Actions[action] = cpy
	}
	return out
}

// Validate returns human-readable warnings for any issues that are not
// fatal but deserve attention: unknown action names, actions missing
// from the file (which will use defaults), empty key lists (unbound).
// The returned slice is deterministic (sorted).
func (b *Bindings) Validate() []string {
	var warnings []string
	if b == nil {
		return nil
	}
	for action, keys := range b.Actions {
		if _, known := ActionFromName(action); !known {
			warnings = append(warnings, fmt.Sprintf("unknown action %q (ignored)", action))
			continue
		}
		if len(keys) == 0 {
			warnings = append(warnings, fmt.Sprintf("action %q has no keys bound (unbound)", action))
		}
	}
	sort.Strings(warnings)
	return warnings
}

// ResolveAction returns the keys bound to a, or nil if the action is
// absent (callers should NOT treat nil and empty-slice the same: a
// missing action should fall back to defaults via Merge, while an empty
// list means "I explicitly unbound this"). Typical use is after Merge
// has already reconciled with defaults.
func (b *Bindings) ResolveAction(a services.Action) ([]string, bool) {
	if b == nil {
		return nil, false
	}
	name := ActionName(a)
	if name == "" {
		return nil, false
	}
	keys, ok := b.Actions[name]
	return keys, ok
}

// MarshalIndentJSON returns the file bytes with 2-space indentation.
// Wraps json.MarshalIndent so callers don't import encoding/json just to
// write the file.
func (b *Bindings) MarshalIndentJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}
