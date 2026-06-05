// Package input is the pure-core binding layer between named Actions
// (services.Action) and symbolic key names ("ArrowUp", "W", "Shift", ...).
//
// Layering
//
//   - This package is Ebiten-free. It speaks only strings: it knows what
//     an Action is called ("MoveUp"), what keys are bound to it by name,
//     how to read/write that table as JSON, and how to merge a user's
//     partial file over the built-in defaults.
//
//   - The actual string→key resolution lives in the platform layer
//     (internal/platform/ebiten/keynames.go). That is where a name like
//     "ArrowUp" becomes ebiten.KeyArrowUp. Separating the two means a
//     future SDL/terminal/web backend can reuse the exact same JSON file.
//
//   - The archtest (internal/archtest/imports_test.go) enforces the
//     Ebiten-free property by listing this package in pureCorePackages.
//
// File format
//
//   - JSON at the OS user-config directory (os.UserConfigDir) under
//     game-test/keybinds.json. Schema:
//
//     {
//     "version": 1,
//     "actions": {
//     "MoveUp":    ["ArrowUp", "W"],
//     "MoveDown":  ["ArrowDown", "S"],
//     ...
//     }
//     }
//
//   - Version is a forward-compatibility anchor. Readers must tolerate a
//     higher version (warn and fall back to defaults) and accept a lower
//     version by explicit migration.
//
//   - Missing actions merge from defaults; extra actions are retained as
//     warnings but not applied (so a typo doesn't silently vanish on
//     resave). Empty key lists are allowed (the action becomes unbound).
//
// Scope
//
//   - Phase 5 is read-only at startup. Editing the file requires restarting
//     the game. An in-game rebinding scene and hot reload are future work;
//     they will sit on top of this package without changing its API.
package input
