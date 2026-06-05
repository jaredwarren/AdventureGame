package input_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/input"
	"github.com/jaredwarren/game-test/internal/services"
)

// TestActionNamesCoversEveryAction is the architectural invariant: every
// services.Action value (except ActionNone) must have a registered name
// in names.go. If this test fails you added an enum value without a
// name — add one in the same commit.
func TestActionNamesCoversEveryAction(t *testing.T) {
	for a := services.ActionNone + 1; a <= services.ActionEditorBrushClear; a++ {
		if name := input.ActionName(a); name == "" {
			t.Errorf("services.Action(%d) has no registered name in input.actionNames", int(a))
		}
	}
}

// TestDefaultBindingsCoversEveryAction confirms factory defaults bind
// every (named) Action to at least one key. A shipping default with no
// binding is almost always a mistake.
func TestDefaultBindingsCoversEveryAction(t *testing.T) {
	b := input.DefaultBindings()
	for a := services.ActionNone + 1; a <= services.ActionEditorBrushClear; a++ {
		name := input.ActionName(a)
		if name == "" {
			continue
		}
		keys, ok := b.Actions[name]
		if !ok || len(keys) == 0 {
			t.Errorf("default binding missing for action %q", name)
		}
	}
}

// TestMergePreservesUserValuesAndFillsMissing exercises the merge rule:
// user-specified actions replace defaults wholesale; missing actions
// keep defaults.
func TestMergePreservesUserValuesAndFillsMissing(t *testing.T) {
	defaults := input.DefaultBindings()
	user := &input.Bindings{
		Version: 1,
		Actions: map[string][]string{
			"Attack": {"Q"}, // user rebinds to Q only
			"Bomb":   {},    // explicit unbind
			// MoveUp absent: should inherit default
		},
	}
	merged := defaults.Merge(user)

	if got := merged.Actions["Attack"]; !reflect.DeepEqual(got, []string{"Q"}) {
		t.Errorf("Attack=%v, want [Q]", got)
	}
	if got := merged.Actions["Bomb"]; len(got) != 0 {
		t.Errorf("Bomb=%v, want empty (explicit unbind)", got)
	}
	if got := merged.Actions["MoveUp"]; len(got) == 0 {
		t.Errorf("MoveUp unexpectedly empty after merge; want defaults")
	}
}

// TestMergeDoesNotMutateReceivers confirms we return a fresh value.
// If merge mutated in place, repeated calls or concurrent reads would
// produce surprising behavior.
func TestMergeDoesNotMutateReceivers(t *testing.T) {
	defaults := input.DefaultBindings()
	snapshot := defaults.Clone()
	user := &input.Bindings{Actions: map[string][]string{"Attack": {"Q"}}}

	_ = defaults.Merge(user)

	if !reflect.DeepEqual(defaults, snapshot) {
		t.Errorf("Merge mutated receiver: defaults changed")
	}
}

// TestValidateDetectsUnknownActionsAndEmptyLists runs validation over a
// crafted table with both problem cases.
func TestValidateDetectsUnknownActionsAndEmptyLists(t *testing.T) {
	b := &input.Bindings{
		Version: 1,
		Actions: map[string][]string{
			"Attack":        {"Q"},
			"WhatIsThis":    {"A"},
			"EmptyExplicit": {},
		},
	}
	ws := b.Validate()
	if len(ws) < 2 {
		t.Fatalf("want at least 2 warnings (unknown + empty), got %v", ws)
	}
	joined := strings.Join(ws, "|")
	if !strings.Contains(joined, "WhatIsThis") {
		t.Errorf("missing unknown-action warning: %v", ws)
	}
	if !strings.Contains(joined, "EmptyExplicit") {
		t.Errorf("missing empty-list warning: %v", ws)
	}
}

// TestBindingsRoundtripJSON ensures MarshalIndent+Unmarshal is lossless.
func TestBindingsRoundtripJSON(t *testing.T) {
	src := input.DefaultBindings()
	raw, err := src.MarshalIndentJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got input.Bindings
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Version != src.Version {
		t.Errorf("Version changed: got %d want %d", got.Version, src.Version)
	}
	if len(got.Actions) != len(src.Actions) {
		t.Errorf("Actions len changed: got %d want %d", len(got.Actions), len(src.Actions))
	}
	for k, v := range src.Actions {
		if !reflect.DeepEqual(v, got.Actions[k]) {
			t.Errorf("Actions[%q] roundtrip mismatch: got %v want %v", k, got.Actions[k], v)
		}
	}
}

// TestLoadOrInitWritesDefaultsOnFirstRun covers the happy first-run
// path: no file present, we produce one with defaults and return them.
func TestLoadOrInitWritesDefaultsOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybinds.json")

	b, warnings, err := input.LoadOrInit(path)
	if err != nil {
		t.Fatalf("LoadOrInit: %v", err)
	}
	if b == nil {
		t.Fatal("nil bindings")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file written to %s, stat: %v", path, err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "wrote default") {
		t.Errorf("want first-run warning mentioning 'wrote default', got %v", warnings)
	}
	// Parse the on-disk file and confirm it equals defaults.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	var onDisk input.Bindings
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("on-disk file not valid JSON: %v", err)
	}
	if onDisk.Version != input.SchemaVersion {
		t.Errorf("on-disk version = %d, want %d", onDisk.Version, input.SchemaVersion)
	}
}

// TestLoadOrInitMergesUserFileOverDefaults simulates a user who rebound
// one action: we keep their choice, fill the rest from defaults.
func TestLoadOrInitMergesUserFileOverDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybinds.json")
	mustWriteJSON(t, path, input.Bindings{
		Version: 1,
		Actions: map[string][]string{"Attack": {"Q"}},
	})

	b, _, err := input.LoadOrInit(path)
	if err != nil {
		t.Fatalf("LoadOrInit: %v", err)
	}
	if got := b.Actions["Attack"]; !reflect.DeepEqual(got, []string{"Q"}) {
		t.Errorf("Attack=%v, want [Q]", got)
	}
	if got := b.Actions["MoveUp"]; len(got) == 0 {
		t.Errorf("MoveUp missing after merge (expected default fill)")
	}
}

// TestLoadOrInitMalformedFileFallsBackToDefaults confirms the game is
// still launchable if the user corrupts the config.
func TestLoadOrInitMalformedFileFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybinds.json")
	if err := os.WriteFile(path, []byte("{not json at all"), 0o644); err != nil {
		t.Fatalf("seed malformed: %v", err)
	}
	b, warnings, err := input.LoadOrInit(path)
	if err != nil {
		t.Fatalf("LoadOrInit returned error (should warn, not fail): %v", err)
	}
	if b == nil {
		t.Fatal("nil bindings after malformed file")
	}
	// Defaults intact.
	if len(b.Actions) == 0 {
		t.Error("expected defaults, got empty bindings")
	}
	// At least one warning mentioning JSON / invalid.
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "JSON") || strings.Contains(w, "json") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about invalid JSON, got %v", warnings)
	}
}

// TestResolveAction returns the key list for a known action and reports
// ok=false for ActionNone and unregistered values.
func TestResolveAction(t *testing.T) {
	b := input.DefaultBindings()
	keys, ok := b.ResolveAction(services.ActionAttack)
	if !ok || len(keys) == 0 {
		t.Errorf("Attack: ok=%v keys=%v", ok, keys)
	}
	if _, ok := b.ResolveAction(services.ActionNone); ok {
		t.Errorf("ActionNone should not resolve")
	}
}

func mustWriteJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestAllActionNamesIsStable returns registered names in any order but
// without duplicates or empties.
func TestAllActionNamesIsStable(t *testing.T) {
	names := input.AllActionNames()
	sort.Strings(names)
	seen := map[string]bool{}
	for _, n := range names {
		if n == "" {
			t.Error("empty name in AllActionNames")
		}
		if seen[n] {
			t.Errorf("duplicate name %q", n)
		}
		seen[n] = true
	}
}
