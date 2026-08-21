package editorweb

import (
	"strings"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Tile palette grouping.
//
// Top-level sections (Terrain, Hazards, …) hold standalone tiles plus nested
// family rows. Each family shows its root/base tile until expanded, so the
// sidebar stays short even as new autotile families are registered.

// paletteFamily is one expandable root → variants row inside a section.
type paletteFamily struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	BaseGID int    `json:"baseGid"`
	GIDs    []int  `json:"gids"`
}

// paletteGroup is one collapsible section of the palette.
type paletteGroup struct {
	ID        string          `json:"id"`
	Label     string          `json:"label"`
	GIDs      []int           `json:"gids,omitempty"`
	Families  []paletteFamily `json:"families,omitempty"`
	Collapsed bool            `json:"collapsed,omitempty"`
}

// buildPalette groups every registered GID.
func buildPalette() []paletteGroup {
	claimed := make(map[int]bool)
	familiesBySection := map[string][]paletteFamily{}

	for _, f := range tile.RegisteredFamilies() {
		var gids []int
		for _, gid := range f.GIDs {
			if tile.DefOf(gid).Name != "unknown" {
				gids = append(gids, gid)
				claimed[gid] = true
			}
		}
		if len(gids) == 0 {
			continue
		}
		label := f.Label
		if label == f.Name {
			label = formatFamilyLabel(f.Name)
		}
		base := f.BaseGID
		if tile.DefOf(base).Name == "unknown" {
			base = gids[0]
		}
		section := familySection(base)
		familiesBySection[section] = append(familiesBySection[section], paletteFamily{
			ID:      f.Name,
			Label:   label,
			BaseGID: base,
			GIDs:    gids,
		})
	}

	var out []paletteGroup

	// 1. Terrain
	terrainGIDs := []int{tile.GIDGrass, tile.GIDSand, tile.GIDFloor2, tile.GIDEmpty}
	var terrain []int
	for _, gid := range terrainGIDs {
		if tile.DefOf(gid).Name != "unknown" {
			terrain = append(terrain, gid)
			claimed[gid] = true
		}
	}
	if len(terrain) > 0 || len(familiesBySection["terrain"]) > 0 {
		out = append(out, paletteGroup{
			ID:       "terrain",
			Label:    "Terrain",
			GIDs:     terrain,
			Families: familiesBySection["terrain"],
		})
	}

	// 2. Hazards & Surfaces
	hazardGIDs := []int{tile.GIDMud, tile.GIDIce, tile.GIDLava, tile.GIDQuicksand}
	var hazards []int
	for _, gid := range hazardGIDs {
		if tile.DefOf(gid).Name != "unknown" && !claimed[gid] {
			hazards = append(hazards, gid)
			claimed[gid] = true
		}
	}
	if len(hazards) > 0 || len(familiesBySection["hazards"]) > 0 {
		out = append(out, paletteGroup{
			ID:       "hazards",
			Label:    "Hazards & Surfaces",
			GIDs:     hazards,
			Families: familiesBySection["hazards"],
		})
	}

	// 3. Water
	if len(familiesBySection["water"]) > 0 {
		out = append(out, paletteGroup{
			ID:       "water",
			Label:    "Water",
			Families: familiesBySection["water"],
		})
	}

	// 4. Structures & Objects
	structureGIDs := []int{tile.GIDTree, tile.GIDCracked, tile.GIDDoor, tile.GIDLock, tile.GIDSign}
	var structures []int
	for _, gid := range structureGIDs {
		if tile.DefOf(gid).Name != "unknown" && !claimed[gid] {
			structures = append(structures, gid)
			claimed[gid] = true
		}
	}
	if len(structures) > 0 || len(familiesBySection["structures"]) > 0 {
		out = append(out, paletteGroup{
			ID:       "structures",
			Label:    "Structures & Objects",
			GIDs:     structures,
			Families: familiesBySection["structures"],
		})
	}

	// 5. Catch-all for any remaining registered GIDs / unsectioned families
	var other []int
	for _, gid := range tile.RegisteredGIDs() {
		if !claimed[gid] {
			other = append(other, gid)
			claimed[gid] = true
		}
	}
	if len(other) > 0 || len(familiesBySection["other"]) > 0 {
		out = append(out, paletteGroup{
			ID:       "other",
			Label:    "Other",
			GIDs:     other,
			Families: familiesBySection["other"],
		})
	}

	return out
}

// familySection places an autotile family under the palette section that matches
// its base tile's surface/tags.
func familySection(baseGID int) string {
	d := tile.DefOf(baseGID)
	switch d.Surface.Type {
	case tile.SurfaceMud, tile.SurfaceIce, tile.SurfaceLava, tile.SurfaceQuicksand:
		return "hazards"
	}
	if d.Water() {
		return "water"
	}
	if d.Wall() {
		return "structures"
	}
	return "terrain"
}

func formatFamilyLabel(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// defaultFavorites seeds the palette's 1-9,0 hotkey slots.
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
