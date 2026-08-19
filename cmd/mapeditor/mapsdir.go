package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// moduleName identifies this repo's go.mod, so a stray go.mod in a parent
// directory cannot be mistaken for the game's root.
const moduleName = "github.com/jaredwarren/game-test"

// resolveMapsDir finds the maps directory.
//
// The in-game editor hardcodes filepath.Join("assets", "maps", ...), which only
// works when the process CWD happens to be the repo root. Running it from
// anywhere else silently fails to find the map. This walks up to the module root
// instead, so the tool works from any subdirectory or from an installed binary.
func resolveMapsDir(explicit string) (string, error) {
	if explicit != "" {
		return checkDir(explicit)
	}
	if v := os.Getenv("GAME_MAPS_DIR"); v != "" {
		return checkDir(v)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, ok := findModuleRoot(cwd); ok {
		if d, err := checkDir(filepath.Join(root, "assets", "maps")); err == nil {
			return d, nil
		}
	}
	// Covers `go build -o bin/mapeditor` run from elsewhere.
	if exe, err := os.Executable(); err == nil {
		if root, ok := findModuleRoot(filepath.Dir(exe)); ok {
			if d, err := checkDir(filepath.Join(root, "assets", "maps")); err == nil {
				return d, nil
			}
		}
	}
	return "", fmt.Errorf("cannot locate assets/maps from %s; pass -maps <dir> or set GAME_MAPS_DIR", cwd)
}

func checkDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	return abs, nil
}

// findModuleRoot walks up looking for a go.mod that declares this module.
func findModuleRoot(start string) (string, bool) {
	dir := start
	for {
		if declaresModule(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func declaresModule(goMod string) bool {
	f, err := os.Open(goMod)
	if err != nil {
		return false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if name, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(name) == moduleName
		}
	}
	return false
}
