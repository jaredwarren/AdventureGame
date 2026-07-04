// Package tile defines ground tile GIDs, collision rules, and persistence keys.
package tile

import (
	"image/color"

	"github.com/jaredwarren/game-test/internal/geom"
)

// Floorer is satisfied by any tile that can serve as a walkable floor.
type Floorer interface {
	IsFloor() bool
}

// Waller is satisfied by any tile that represents an impassable surface.
type Waller interface {
	IsWall() bool
}

// Waterer is satisfied by any tile that is water or a water shore transition.
type Waterer interface {
	IsWater() bool
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
	Water         bool
	WaterShore    bool
	SolidRects    []geom.Rect
	DamageKinds   []DamageKind
	OpenableByKey bool
	DestroyedGID  int
	SwatchColor   color.RGBA
	FloorWeight   float64
	VectorDraw    func(c Canvas, x, y, w, h float32)
}

func (d Def) IsFloor() bool { return !d.Solid }
func (d Def) IsWall() bool  { return d.Solid }
func (d Def) IsWater() bool { return d.Water }
func (d Def) IsLand() bool  { return !d.Water }

func (d Def) DrawVector(c Canvas, x, y, w, h float32) {
	if d.VectorDraw != nil {
		d.VectorDraw(c, x, y, w, h)
		return
	}
	c.FillRect(x, y, w, h, d.SwatchColor)
}

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
	GIDEmpty:    {GID: GIDEmpty, Name: "empty", Solid: false, SwatchColor: color.RGBA{0x00, 0x00, 0x00, 0xff}, VectorDraw: drawEmpty},
	GIDGrass:    {GID: GIDGrass, Name: "grass", Solid: false, FloorWeight: 0.58, SwatchColor: color.RGBA{0x55, 0x88, 0x55, 0xff}, VectorDraw: drawGrass},
	GIDWall:     {GID: GIDWall, Name: "wall", Solid: true, SwatchColor: color.RGBA{0x40, 0x40, 0x50, 0xff}, VectorDraw: drawWall},
	GIDCracked:  {GID: GIDCracked, Name: "cracked", DamageKinds: []DamageKind{DamageBomb}, SwatchColor: color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, VectorDraw: drawCracked},
	GIDDoor:     {GID: GIDDoor, Name: "door", Solid: false, SwatchColor: color.RGBA{0x3a, 0x5a, 0x3a, 0xff}, VectorDraw: drawDoor},
	GIDWater:    {GID: GIDWater, Name: "water", Solid: true, Water: true, FloorWeight: 0, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}, VectorDraw: drawWater},
	GIDLock:     {GID: GIDLock, Name: "lock", OpenableByKey: true, SwatchColor: color.RGBA{0x6a, 0x2a, 0x7a, 0xff}, VectorDraw: drawLock},
	GIDFloor2:   {GID: GIDFloor2, Name: "floor2", Solid: false, FloorWeight: 0.42, SwatchColor: color.RGBA{0x50, 0x50, 0x5c, 0xff}, VectorDraw: drawFloor2},
	GIDDirtPath: {GID: GIDDirtPath, Name: "dirt_path", Solid: false, FloorWeight: 0.3, SwatchColor: color.RGBA{0x8b, 0x6a, 0x3a, 0xff}, VectorDraw: drawDirtPath},
	GIDTree:     {GID: GIDTree, Name: "tree", DamageKinds: []DamageKind{DamageFire}, SwatchColor: color.RGBA{0x2d, 0x5a, 0x2a, 0xff}, VectorDraw: drawTree},

	GIDWaterShoreTop: {
		GID: GIDWaterShoreTop, Name: "water_shore_top", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 8, W: 16, H: 8}}, VectorDraw: drawShoreTop,
	},
	GIDWaterShoreBottom: {
		GID: GIDWaterShoreBottom, Name: "water_shore_bottom", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 16, H: 8}}, VectorDraw: drawShoreBottom,
	},
	GIDWaterShoreLeft: {
		GID: GIDWaterShoreLeft, Name: "water_shore_left", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}}, VectorDraw: drawShoreLeft,
	},
	GIDWaterShoreRight: {
		GID: GIDWaterShoreRight, Name: "water_shore_right", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}}, VectorDraw: drawShoreRight,
	},
	GIDWaterShoreNE: {
		GID: GIDWaterShoreNE, Name: "water_shore_ne", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNE,
	},
	GIDWaterShoreNW: {
		GID: GIDWaterShoreNW, Name: "water_shore_nw", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNW,
	},
	GIDWaterShoreSW: {
		GID: GIDWaterShoreSW, Name: "water_shore_sw", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSW,
	},
	GIDWaterShoreSE: {
		GID: GIDWaterShoreSE, Name: "water_shore_se", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSE,
	},

	GIDWaterShoreNEInner: {
		GID: GIDWaterShoreNEInner, Name: "water_shore_ne_inner", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNEInner,
	},
	GIDWaterShoreNWInner: {
		GID: GIDWaterShoreNWInner, Name: "water_shore_nw_inner", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNWInner,
	},
	GIDWaterShoreSWInner: {
		GID: GIDWaterShoreSWInner, Name: "water_shore_sw_inner", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSWInner,
	},
	GIDWaterShoreSEInner: {
		GID: GIDWaterShoreSEInner, Name: "water_shore_se_inner", Solid: true, Water: true, WaterShore: true, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSEInner,
	},
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
