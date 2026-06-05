// disk.go — load/save the binding table to the OS config directory.
//
// File location is resolved by the caller (cmd/game uses os.UserConfigDir
// + "game-test/keybinds.json"). We take the full path so tests can point
// at a t.TempDir() without mocking the OS helper.
//
// Startup contract
//
//   - On first run (file missing): write the defaults and return them.
//     Gives users a discoverable template they can edit.
//   - On read error or JSON parse error: return defaults plus a warning;
//     do NOT crash. A broken keybinds file must never make the game
//     unlaunchable.
//   - On successful read: merge user values over defaults; validation
//     warnings (unknown actions, empty lists) are appended to the
//     returned warning list so main can log them once.
//
// This package does no logging of its own; the caller owns log output so
// tests stay quiet and embedders (CI harness, replay tooling) can
// suppress or redirect the warnings.
package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultFileName is the filename used inside the config subdirectory.
const DefaultFileName = "keybinds.json"

// LoadOrInit reads path (creating it with defaults if it does not
// exist), merges any user overrides on top of defaults, and returns
// the effective Bindings plus a slice of non-fatal warning messages.
//
// The returned Bindings is always non-nil — even in error paths the
// caller can rely on safe defaults.
//
// The returned error is only non-nil for write failures during first-run
// initialization (disk full, readonly dir). A missing file is not an
// error; a malformed file is not an error (it degrades to defaults).
func LoadOrInit(path string) (*Bindings, []string, error) {
	defaults := DefaultBindings()

	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			warnings := []string{fmt.Sprintf("read %s: %v (using defaults)", path, err)}
			return defaults, warnings, nil
		}
		if mkErr := writeFile(path, defaults); mkErr != nil {
			return defaults, nil, fmt.Errorf("create defaults at %s: %w", path, mkErr)
		}
		return defaults, []string{fmt.Sprintf("wrote default keybinds to %s", path)}, nil
	}

	var user Bindings
	if err := json.Unmarshal(raw, &user); err != nil {
		warnings := []string{
			fmt.Sprintf("%s is not valid JSON (%v); falling back to defaults", path, err),
		}
		return defaults, warnings, nil
	}

	merged := defaults.Merge(&user)
	warnings := user.Validate()
	if user.Version > SchemaVersion {
		warnings = append(warnings, fmt.Sprintf(
			"keybinds file claims version %d, this build only understands version %d; using best-effort parse",
			user.Version, SchemaVersion,
		))
	}
	return merged, warnings, nil
}

// Save writes b to path with canonical formatting (MarshalIndent). The
// parent directory is created as needed. Typically called by a future
// rebinding UI; Phase 5 only invokes it via LoadOrInit's first-run path.
func Save(path string, b *Bindings) error { return writeFile(path, b) }

func writeFile(path string, b *Bindings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := b.MarshalIndentJSON()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
