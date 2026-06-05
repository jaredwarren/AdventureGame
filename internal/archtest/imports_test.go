// Package archtest enforces architectural boundaries by parsing Go imports.
//
// The core simulation packages (world, tiled, progression, save, geom, render,
// services, scenes, systems, input, replay, core) must never import Ebiten or
// anything under internal/platform. This keeps the simulation unit-testable in
// headless mode and lets us swap rendering/input backends without touching
// game logic.
//
// If this test fails, either (a) your change violates the architecture and
// should route through internal/services instead, or (b) you're intentionally
// changing the rules — in which case update bannedImports below and say so in
// the commit message.
package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// pureCorePackages must remain free of any rendering/input/platform imports.
// Paths are relative to the repo root. internal/scenes is here because it is
// the scene-layer core: it orchestrates world + services + renderer port but
// must never reach across the port to grab an Ebiten type.
var pureCorePackages = []string{
	"internal/world",
	"internal/tiled",
	"internal/progression",
	"internal/save",
	"internal/geom",
	"internal/render",
	"internal/services",
	"internal/scenes",
	"internal/systems",
	"internal/input",
	"internal/replay",
	"internal/core",
}

// bannedImports lists import paths (prefix match) that pureCorePackages must not use.
// Prefix match means github.com/hajimehoshi/ebiten/v2/audio is also banned.
var bannedImports = []string{
	"github.com/hajimehoshi/ebiten",
	"github.com/jaredwarren/game-test/internal/platform",
	"github.com/jaredwarren/game-test/internal/game",
	"github.com/jaredwarren/game-test/assets/sprites",
	// Third-party platform crud that must route through services/* ports:
	"github.com/atotto/clipboard",
}

func TestCorePackagesAreEbitenFree(t *testing.T) {
	root := repoRoot(t)
	for _, pkg := range pureCorePackages {
		dir := filepath.Join(root, pkg)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Package may not exist yet (e.g. internal/core during migration); skip.
			continue
		}
		walkGoFiles(t, dir, func(path string) {
			imports := readImports(t, path)
			for _, imp := range imports {
				for _, banned := range bannedImports {
					if strings.HasPrefix(imp, banned) {
						rel, _ := filepath.Rel(root, path)
						t.Errorf("%s imports %q (banned for %q): route through internal/services instead", rel, imp, pkg)
					}
				}
			}
		})
	}
}

func walkGoFiles(t *testing.T, dir string, fn func(path string)) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			walkGoFiles(t, full, fn)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		fn(full)
	}
}

func readImports(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		v, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// This test lives at internal/archtest/imports_test.go; walk up to the module root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	t.Fatalf("no go.mod found from %s", wd)
	return ""
}
