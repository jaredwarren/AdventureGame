// Tile definition registry: the single source of truth for per-GID
// collision, destructibility, lock behavior, and fallback render color.
//
// Adding a new ground tile should only require:
//  1. A new GID constant in tiles.go.
//  2. A new entry in tileDefs below (and, if the tile should be drawable,
//     a matching frame in assets/sprites/tile_atlas.json).
//
// Systems (SolidAt, TryDamageFaceTile, ApplyDestroyedTiles, renderer,
// editor palette) read from this table instead of switching on GID, so
// new tile types and new damage kinds do not fan out across the codebase.
package world

import "image/color"

// DamageKind identifies a source of tile damage (bomb, fire, future
// weapons). New weapons add a new constant here and declare the tiles
// they can destroy via TileDef.DamageKinds.
type DamageKind int

const (
	DamageBomb DamageKind = iota
	DamageFire
)

// TileDef declares the behavior and fallback rendering of a ground tile.
// The zero value intentionally describes a passable, indestructible tile
// so unknown GIDs fail closed via SolidAt's default branch.
type TileDef struct {
	GID  int
	Name string

	// Solid is the base collision for the tile when not in a special
	// state (destroyable, openable-by-key).
	Solid bool

	// DamageKinds lists the damage sources that can destroy this tile.
	// Non-empty implies the tile is solid until it is marked destroyed
	// in World.DestroyedTiles.
	DamageKinds []DamageKind

	// OpenableByKey marks the tile as a locked door: solid unless the
	// player has a small key; walking onto it consumes one key and
	// converts it to DestroyedGID (see systems.LockSystem).
	OpenableByKey bool

	// DestroyedGID is the tile GID a destroyed/opened tile becomes in
	// renderers and persistence fallbacks. Zero = GIDGrass.
	DestroyedGID int

	// SwatchColor is the flat fallback fill used when tile sprites are
	// disabled or the atlas is missing a frame.
	SwatchColor color.RGBA
}

// AcceptsDamage reports whether this tile type can be destroyed by the
// given kind.
func (d TileDef) AcceptsDamage(k DamageKind) bool {
	for _, dk := range d.DamageKinds {
		if dk == k {
			return true
		}
	}
	return false
}

// Destroyable reports whether at least one damage kind can destroy this tile.
func (d TileDef) Destroyable() bool { return len(d.DamageKinds) > 0 }

// ResolvedDestroyedGID returns DestroyedGID or GIDGrass fallback.
func (d TileDef) ResolvedDestroyedGID() int {
	if d.DestroyedGID == 0 {
		return GIDGrass
	}
	return d.DestroyedGID
}

var tileDefs = map[int]TileDef{
	GIDEmpty:   {GID: GIDEmpty, Name: "empty", Solid: false, SwatchColor: color.RGBA{0x00, 0x00, 0x00, 0xff}},
	GIDGrass:   {GID: GIDGrass, Name: "grass", Solid: false, SwatchColor: color.RGBA{0x2b, 0x4a, 0x2b, 0xff}},
	GIDWall:    {GID: GIDWall, Name: "wall", Solid: true, SwatchColor: color.RGBA{0x40, 0x40, 0x50, 0xff}},
	GIDCracked: {GID: GIDCracked, Name: "cracked", DamageKinds: []DamageKind{DamageBomb}, SwatchColor: color.RGBA{0x6b, 0x4a, 0x2a, 0xff}},
	GIDDoor:    {GID: GIDDoor, Name: "door", Solid: false, SwatchColor: color.RGBA{0x3a, 0x5a, 0x3a, 0xff}},
	// Blocking water. A future “deep water” vs shallow variant can split into
	// two GIDs if you want both barriers and walkable puddles.
	GIDWater:  {GID: GIDWater, Name: "water", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDLock:   {GID: GIDLock, Name: "lock", OpenableByKey: true, SwatchColor: color.RGBA{0x6a, 0x2a, 0x7a, 0xff}},
	GIDFloor2: {GID: GIDFloor2, Name: "floor2", Solid: false, SwatchColor: color.RGBA{0x3a, 0x3a, 0x44, 0xff}},
	GIDTree:   {GID: GIDTree, Name: "tree", DamageKinds: []DamageKind{DamageFire}, SwatchColor: color.RGBA{0x2d, 0x5a, 0x2a, 0xff}},

	// Water shore transition tiles (solid water block behavior)
	GIDWaterShoreTop:    {GID: GIDWaterShoreTop, Name: "water_shore_top", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreBottom: {GID: GIDWaterShoreBottom, Name: "water_shore_bottom", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreLeft:   {GID: GIDWaterShoreLeft, Name: "water_shore_left", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreRight:  {GID: GIDWaterShoreRight, Name: "water_shore_right", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreNE:     {GID: GIDWaterShoreNE, Name: "water_shore_ne", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreNW:     {GID: GIDWaterShoreNW, Name: "water_shore_nw", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSW:     {GID: GIDWaterShoreSW, Name: "water_shore_sw", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSE:     {GID: GIDWaterShoreSE, Name: "water_shore_se", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
}

// TileDefOf returns the registered definition for a GID. When the GID is
// not in the registry, the returned def has Solid=true so unknown tiles
// fail closed (matching the old SolidAt default branch), and a visible
// fallback color.
func TileDefOf(gid int) TileDef {
	if d, ok := tileDefs[gid]; ok {
		return d
	}
	return TileDef{GID: gid, Name: "unknown", Solid: true, SwatchColor: color.RGBA{0x30, 0x30, 0x38, 0xff}}
}

// RegisteredTileGIDs returns all GIDs with a TileDef, sorted ascending.
// Used by the editor brush palette and tests.
func RegisteredTileGIDs() []int {
	out := make([]int, 0, len(tileDefs))
	for gid := range tileDefs {
		out = append(out, gid)
	}
	// Tiny set (<20); simple selection sort keeps this dep-free.
	for i := 0; i < len(out); i++ {
		min := i
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[min] {
				min = j
			}
		}
		out[i], out[min] = out[min], out[i]
	}
	return out
}
