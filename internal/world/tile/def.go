// Package tile defines ground tile GIDs, collision rules, and persistence keys.
package tile

import "image/color"

// Floorer is satisfied by any tile that can serve as a walkable floor.
type Floorer interface {
	IsFloor() bool
}

// Waller is satisfied by any tile that represents an impassable surface.
type Waller interface {
	IsWall() bool
}

// DamageKind identifies a source of tile damage (bomb, fire, future weapons).
type DamageKind int

const (
	DamageBomb DamageKind = iota
	DamageFire
)

// Def declares the behavior and fallback rendering of a ground tile.
type Def struct {
	GID  int
	Name string

	Solid         bool
	DamageKinds   []DamageKind
	OpenableByKey bool
	DestroyedGID  int
	SwatchColor   color.RGBA
	FloorWeight   float64
}

func (d Def) IsFloor() bool { return !d.Solid }
func (d Def) IsWall() bool  { return d.Solid }

func (d Def) AcceptsDamage(k DamageKind) bool {
	for _, dk := range d.DamageKinds {
		if dk == k {
			return true
		}
	}
	return false
}

func (d Def) Destroyable() bool { return len(d.DamageKinds) > 0 }

func (d Def) ResolvedDestroyedGID() int {
	if d.DestroyedGID == 0 {
		return GIDGrass
	}
	return d.DestroyedGID
}

var defs = map[int]Def{
	GIDEmpty:    {GID: GIDEmpty, Name: "empty", Solid: false, SwatchColor: color.RGBA{0x00, 0x00, 0x00, 0xff}},
	GIDGrass:    {GID: GIDGrass, Name: "grass", Solid: false, FloorWeight: 0.58, SwatchColor: color.RGBA{0x55, 0x88, 0x55, 0xff}},
	GIDWall:     {GID: GIDWall, Name: "wall", Solid: true, SwatchColor: color.RGBA{0x40, 0x40, 0x50, 0xff}},
	GIDCracked:  {GID: GIDCracked, Name: "cracked", DamageKinds: []DamageKind{DamageBomb}, SwatchColor: color.RGBA{0x6b, 0x4a, 0x2a, 0xff}},
	GIDDoor:     {GID: GIDDoor, Name: "door", Solid: false, SwatchColor: color.RGBA{0x3a, 0x5a, 0x3a, 0xff}},
	GIDWater:    {GID: GIDWater, Name: "water", Solid: true, FloorWeight: 0, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDLock:     {GID: GIDLock, Name: "lock", OpenableByKey: true, SwatchColor: color.RGBA{0x6a, 0x2a, 0x7a, 0xff}},
	GIDFloor2:   {GID: GIDFloor2, Name: "floor2", Solid: false, FloorWeight: 0.42, SwatchColor: color.RGBA{0x50, 0x50, 0x5c, 0xff}},
	GIDDirtPath: {GID: GIDDirtPath, Name: "dirt_path", Solid: false, FloorWeight: 0.3, SwatchColor: color.RGBA{0x8b, 0x6a, 0x3a, 0xff}},
	GIDTree:     {GID: GIDTree, Name: "tree", DamageKinds: []DamageKind{DamageFire}, SwatchColor: color.RGBA{0x2d, 0x5a, 0x2a, 0xff}},

	GIDWaterShoreTop:    {GID: GIDWaterShoreTop, Name: "water_shore_top", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreBottom: {GID: GIDWaterShoreBottom, Name: "water_shore_bottom", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreLeft:   {GID: GIDWaterShoreLeft, Name: "water_shore_left", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreRight:  {GID: GIDWaterShoreRight, Name: "water_shore_right", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreNE:     {GID: GIDWaterShoreNE, Name: "water_shore_ne", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreNW:     {GID: GIDWaterShoreNW, Name: "water_shore_nw", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSW:     {GID: GIDWaterShoreSW, Name: "water_shore_sw", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSE:     {GID: GIDWaterShoreSE, Name: "water_shore_se", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},

	GIDWaterShoreNEInner: {GID: GIDWaterShoreNEInner, Name: "water_shore_ne_inner", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreNWInner: {GID: GIDWaterShoreNWInner, Name: "water_shore_nw_inner", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSWInner: {GID: GIDWaterShoreSWInner, Name: "water_shore_sw_inner", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
	GIDWaterShoreSEInner: {GID: GIDWaterShoreSEInner, Name: "water_shore_se_inner", Solid: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}},
}

// DefOf returns the registered definition for a GID.
func DefOf(gid int) Def {
	if d, ok := defs[gid]; ok {
		return d
	}
	return Def{GID: gid, Name: "unknown", Solid: true, SwatchColor: color.RGBA{0x30, 0x30, 0x38, 0xff}}
}

// RegisteredGIDs returns all GIDs with a Def, sorted ascending.
func RegisteredGIDs() []int {
	out := make([]int, 0, len(defs))
	for gid := range defs {
		out = append(out, gid)
	}
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
