// Package archtest enforces architectural boundaries by parsing Go imports.
//
// The core simulation packages (world, tiled, progression, save, geom, render,
// services, scenes, systems, input, replay) must never import Ebiten or
// anything under internal/platform. This keeps the simulation unit-testable in
// headless mode and lets us swap rendering/input backends without touching
// game logic.
//
// If this test fails, either (a) your change violates the architecture and
// should route through internal/services instead, or (b) you're intentionally
// changing the rules — in which case update bannedImports/allowedInternalImports
// below and say so in the commit message.
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
//
// Note: internal/core was removed as it was a dormant placeholder that does not exist on disk.
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

// allowedInternalImports defines the architecture contract layer table (archtest v2).
// Maps each internal package path (relative to repo root) to its allowed internal imports.
// Every internal package on disk MUST be classified here or in exemptPackages.
var allowedInternalImports = map[string][]string{
	"internal/geom":        {},
	"internal/tiled":       {},
	"internal/progression": {},
	"internal/render":      {},
	"internal/save": {
		// TODO(phase-4): remove internal/world when save codec is fully decoupled
		"internal/world",
	},
	"internal/input": {
		"internal/services",
	},
	"internal/replay": {
		"internal/services",
		"internal/input",
	},
	"internal/services": {
		"internal/world",
		"internal/render",
	},
	"internal/world": {
		"internal/geom",
		"internal/tiled",
		"internal/progression",
		"internal/world/enemy",
		"internal/world/pickup",
		"internal/world/tile",
	},
	"internal/world/tile": {
		"internal/geom",
		"internal/tiled",
		"internal/progression",
	},
	"internal/world/pickup": {
		"internal/geom",
		"internal/tiled",
		"internal/progression",
	},
	"internal/world/enemy": {
		"internal/geom",
		"internal/tiled",
		"internal/progression",
	},
	"internal/systems": {
		"internal/world",
		"internal/geom",
	},
	"internal/dungeon": {
		"internal/tiled",
		"internal/geom",
	},
	"internal/scenes": {
		"internal/systems",
		"internal/world",
		"internal/services",
		"internal/render",
		"internal/progression",
		"internal/geom",
		"internal/save",
		"internal/tiled",
		"internal/dungeon",
	},
	"internal/platform/ebiten": {
		"internal/services",
		"internal/world",
		"internal/render",
		"internal/input",
	},
	"internal/game": {
		"internal/platform/ebiten",
		"internal/scenes",
		"internal/services",
		"internal/render",
		"internal/save",
	},
}

// exemptPackages lists internal packages that are exempt from layer import rules (e.g. test-only packages).
var exemptPackages = map[string]bool{
	"internal/archtest": true,
}

func TestCorePackagesAreEbitenFree(t *testing.T) {
	root := repoRoot(t)
	for _, pkg := range pureCorePackages {
		dir := filepath.Join(root, pkg)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Package may not exist yet; skip.
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

func TestLayeredImports(t *testing.T) {
	root := repoRoot(t)
	internalDir := filepath.Join(root, "internal")

	// 1. Walk internal/ to ensure every directory containing .go files is classified.
	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		hasGoFiles := false
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				hasGoFiles = true
				break
			}
		}
		if hasGoFiles {
			if !exemptPackages[rel] {
				if _, ok := allowedInternalImports[rel]; !ok {
					t.Errorf("unclassified internal package directory %q: add to allowedInternalImports in internal/archtest/imports_test.go", rel)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk internal dir: %v", err)
	}

	// 2. Verify allowed internal imports for each classified package.
	const moduleInternalPrefix = "github.com/jaredwarren/game-test/internal/"

	for pkgPath, allowedList := range allowedInternalImports {
		pkgDir := filepath.Join(root, filepath.FromSlash(pkgPath))
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			continue
		}

		allowedSet := make(map[string]bool)
		for _, allowed := range allowedList {
			allowedSet[allowed] = true
		}

		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			t.Fatalf("read %s: %v", pkgDir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			filePath := filepath.Join(pkgDir, e.Name())
			imports := readImports(t, filePath)
			relFile, _ := filepath.Rel(root, filePath)
			relFile = filepath.ToSlash(relFile)

			for _, imp := range imports {
				if strings.HasPrefix(imp, moduleInternalPrefix) {
					targetPkg := strings.TrimPrefix(imp, "github.com/jaredwarren/game-test/")
					if targetPkg == pkgPath {
						continue
					}
					if !allowedSet[targetPkg] {
						t.Errorf("%s imports %q (%q is not in allowed internal imports for %q)", relFile, imp, targetPkg, pkgPath)
					}
				}
			}
		}
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
