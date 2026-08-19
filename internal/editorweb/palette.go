package editorweb

import (
	"strings"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Tile palette grouping.
//
// The palette is dynamically generated from registered tile families (tile.RegisteredFamilies)
// and predefined standalone categories. Any new family registered via tile.RegisterFamily
// automatically creates its own collapsible group without needing manual rules.

// paletteGroup is one collapsible section of the palette.
type paletteGroup struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	GIDs      []int  `json:"gids"`
	Collapsed bool   `json:"collapsed,omitempty"`
}

// buildPalette groups every registered GID.
func buildPalette() []paletteGroup {
	claimed := make(map[int]bool)
	var out []paletteGroup

	// 1. Terrain group
	terrainGIDs := []int{tile.GIDGrass, tile.GIDSand, tile.GIDFloor2, tile.GIDEmpty}
	var terrain []int
	for _, gid := range terrainGIDs {
		if tile.DefOf(gid).Name != "unknown" {
			terrain = append(terrain, gid)
			claimed[gid] = true
		}
	}
	if len(terrain) > 0 {
		out = append(out, paletteGroup{ID: "terrain", Label: "Terrain", GIDs: terrain})
	}

	// 2. Dynamic Family Groups (e.g. dirt_path, water, wall, rock, etc.)
	for _, f := range tile.RegisteredFamilies() {
		var gids []int
		for _, gid := range f.GIDs {
			if tile.DefOf(gid).Name != "unknown" {
				gids = append(gids, gid)
				claimed[gid] = true
			}
		}
		if len(gids) > 0 {
			label := f.Label
			if label == f.Name {
				// Format "dirt_path" -> "Dirt Path", "water" -> "Water", etc.
				label = formatFamilyLabel(f.Name)
			}
			out = append(out, paletteGroup{
				ID:        f.Name,
				Label:     label,
				GIDs:      gids,
				Collapsed: f.Collapsed,
			})
		}
	}

	// 3. Hazards & Surfaces
	hazardGIDs := []int{tile.GIDMud, tile.GIDIce, tile.GIDLava, tile.GIDQuicksand}
	var hazards []int
	for _, gid := range hazardGIDs {
		if tile.DefOf(gid).Name != "unknown" && !claimed[gid] {
			hazards = append(hazards, gid)
			claimed[gid] = true
		}
	}
	if len(hazards) > 0 {
		out = append(out, paletteGroup{ID: "hazards", Label: "Hazards & Surfaces", GIDs: hazards})
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
	if len(structures) > 0 {
		out = append(out, paletteGroup{ID: "structures", Label: "Structures & Objects", GIDs: structures})
	}

	// 5. Catch-all for any remaining registered GIDs
	var other []int
	for _, gid := range tile.RegisteredGIDs() {
		if !claimed[gid] {
			other = append(other, gid)
			claimed[gid] = true
		}
	}
	if len(other) > 0 {
		out = append(out, paletteGroup{ID: "other", Label: "Other", GIDs: other})
	}

	return out
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
