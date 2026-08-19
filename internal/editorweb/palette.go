package editorweb

import (
	"strings"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Tile palette grouping.
//
// The in-game editor collapses the water, wall and rock transition variants into
// submenus (internal/scenes/editor_helpers.go) using a hand-maintained list,
// which is how GIDs like sand get added to the registry and forgotten in the
// palette. Here the grouping is derived from the registry by name prefix, so a
// new GID always lands somewhere, and TestPaletteCoversAllRegisteredGIDs fails
// if two rules ever claim the same tile.

// paletteGroup is one collapsible section of the palette.
type paletteGroup struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	GIDs  []int  `json:"gids"`
	// Collapsed hints that this group is a big block of autotile variants that
	// should start folded away.
	Collapsed bool `json:"collapsed,omitempty"`
}

// paletteRule assigns tiles to a group. Order matters: the first match wins, and
// the final catch-all rule guarantees total coverage.
type paletteRule struct {
	id        string
	label     string
	collapsed bool
	match     func(def tile.Tile) bool
}

var paletteRules = []paletteRule{
	{
		id: "water", label: "Water", collapsed: true,
		match: func(d tile.Tile) bool { return strings.HasPrefix(d.Name, "water") },
	},
	{
		id: "wall", label: "Wall", collapsed: true,
		match: func(d tile.Tile) bool { return strings.HasPrefix(d.Name, "wall") },
	},
	{
		id: "rock", label: "Rock", collapsed: true,
		match: func(d tile.Tile) bool { return strings.HasPrefix(d.Name, "rock") },
	},
	// Catch-all. Keep last.
	{id: "terrain", label: "Terrain", match: func(tile.Tile) bool { return true }},
}

// buildPalette groups every registered GID, preserving registry order within
// each group. Empty groups are dropped.
func buildPalette() []paletteGroup {
	byID := make(map[string]*paletteGroup, len(paletteRules))
	order := make([]*paletteGroup, 0, len(paletteRules))
	for _, r := range paletteRules {
		g := &paletteGroup{ID: r.id, Label: r.label, Collapsed: r.collapsed, GIDs: []int{}}
		byID[r.id] = g
		order = append(order, g)
	}

	for _, gid := range tile.RegisteredGIDs() {
		def := tile.DefOf(gid)
		for _, r := range paletteRules {
			if r.match(def) {
				byID[r.id].GIDs = append(byID[r.id].GIDs, gid)
				break
			}
		}
	}

	out := make([]paletteGroup, 0, len(order))
	for _, g := range order {
		if len(g.GIDs) > 0 {
			out = append(out, *g)
		}
	}
	return out
}

// defaultFavorites seeds the palette's 1-9,0 hotkey slots.
//
// This ports the in-game editor's brush palette (internal/scenes/editor.go) so
// existing muscle memory carries over: 1-8 are the common brushes and 0 erases.
var defaultFavorites = []int{
	tile.GIDGrass,
	tile.GIDWall,
	tile.GIDCracked,
	tile.GIDDoor,
	tile.GIDWater,
	tile.GIDLock,
	tile.GIDFloor2,
	tile.GIDTree,
	tile.GIDRock,
	tile.GIDEmpty, // slot 0 is the eraser
}
