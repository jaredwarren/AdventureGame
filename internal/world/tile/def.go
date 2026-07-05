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

type TileTag string

const (
	TagSolid        TileTag = "solid"
	TagWater        TileTag = "water"
	TagWaterShore   TileTag = "water_shore"
	TagDoor         TileTag = "door"
	TagLock         TileTag = "lock"
	TagIgnitable    TileTag = "ignitable"
	TagDestructible TileTag = "destructible"
)

// DamageKind identifies a source of tile damage (bomb, fire, future weapons).
type DamageKind int

const (
	DamageBomb DamageKind = iota
	DamageFire
)

// Tile declares the behavior, surface attributes, and fallback rendering of a ground tile.
type Tile struct {
	GID           int
	Name          string
	Tags          []TileTag
	Surface       SurfaceDef
	SolidRects    []geom.Rect
	DamageKinds   []DamageKind
	OpenableByKey bool
	DestroyedGID  int
	SwatchColor   color.RGBA
	FloorWeight   float64
	VectorDraw    func(c Canvas, x, y, w, h float32)
}

func (d Tile) HasTag(tag TileTag) bool {
	for _, tg := range d.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// Compatibility helper methods
func (d Tile) Solid() bool      { return d.HasTag(TagSolid) }
func (d Tile) Water() bool      { return d.HasTag(TagWater) }
func (d Tile) WaterShore() bool { return d.HasTag(TagWaterShore) }
func (d Tile) IsFloor() bool    { return !d.Solid() }
func (d Tile) IsWall() bool     { return d.Solid() }
func (d Tile) IsWater() bool    { return d.Water() }
func (d Tile) IsLand() bool     { return !d.Water() && !d.Solid() }

func (d Tile) DrawVector(c Canvas, x, y, w, h float32) {
	if d.VectorDraw != nil {
		d.VectorDraw(c, x, y, w, h)
		return
	}
	c.FillRect(x, y, w, h, d.SwatchColor)
}

func (d Tile) AcceptsDamage(k DamageKind) bool {
	for _, dk := range d.DamageKinds {
		if dk == k {
			return true
		}
	}
	return false
}

func (d Tile) Destroyable() bool { return len(d.DamageKinds) > 0 }

func (d Tile) ResolvedDestroyedGID() int {
	if d.DestroyedGID == 0 {
		return GIDGrass
	}
	return d.DestroyedGID
}

var waterSurface = SurfaceDef{
	Type:            SurfaceWater,
	SpeedMultiplier: 0.5,
	Friction:        0.8,
}

var defs = map[int]Tile{
	GIDEmpty:    {GID: GIDEmpty, Name: "empty", SwatchColor: color.RGBA{0x00, 0x00, 0x00, 0xff}, VectorDraw: drawEmpty},
	GIDGrass:    {GID: GIDGrass, Name: "grass", FloorWeight: 0.58, SwatchColor: color.RGBA{0x55, 0x88, 0x55, 0xff}, VectorDraw: drawGrass},
	GIDWall:     {GID: GIDWall, Name: "wall", Tags: []TileTag{TagSolid}, SwatchColor: color.RGBA{0x40, 0x40, 0x50, 0xff}, VectorDraw: drawWall},
	GIDCracked:  {GID: GIDCracked, Name: "cracked", Tags: []TileTag{TagDestructible}, DamageKinds: []DamageKind{DamageBomb}, SwatchColor: color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, VectorDraw: drawCracked},
	GIDDoor:     {GID: GIDDoor, Name: "door", Tags: []TileTag{TagDoor}, SwatchColor: color.RGBA{0x3a, 0x5a, 0x3a, 0xff}, VectorDraw: drawDoor},
	GIDWater:    {GID: GIDWater, Name: "water", Tags: []TileTag{TagSolid, TagWater}, Surface: waterSurface, FloorWeight: 0, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff}, VectorDraw: drawWater},
	GIDLock:     {GID: GIDLock, Name: "lock", Tags: []TileTag{TagLock}, OpenableByKey: true, SwatchColor: color.RGBA{0x6a, 0x2a, 0x7a, 0xff}, VectorDraw: drawLock},
	GIDFloor2:   {GID: GIDFloor2, Name: "floor2", FloorWeight: 0.42, SwatchColor: color.RGBA{0x50, 0x50, 0x5c, 0xff}, VectorDraw: drawFloor2},
	GIDDirtPath: {GID: GIDDirtPath, Name: "dirt_path", FloorWeight: 0.3, SwatchColor: color.RGBA{0x8b, 0x6a, 0x3a, 0xff}, VectorDraw: drawDirtPath},
	GIDTree:     {GID: GIDTree, Name: "tree", Tags: []TileTag{TagIgnitable}, DamageKinds: []DamageKind{DamageFire}, SwatchColor: color.RGBA{0x2d, 0x5a, 0x2a, 0xff}, VectorDraw: drawTree},

	GIDWaterShoreTop: {
		GID: GIDWaterShoreTop, Name: "water_shore_top", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 8, W: 16, H: 8}}, VectorDraw: drawShoreTop,
	},
	GIDWaterShoreBottom: {
		GID: GIDWaterShoreBottom, Name: "water_shore_bottom", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 16, H: 8}}, VectorDraw: drawShoreBottom,
	},
	GIDWaterShoreLeft: {
		GID: GIDWaterShoreLeft, Name: "water_shore_left", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}}, VectorDraw: drawShoreLeft,
	},
	GIDWaterShoreRight: {
		GID: GIDWaterShoreRight, Name: "water_shore_right", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}}, VectorDraw: drawShoreRight,
	},
	GIDWaterShoreNE: {
		GID: GIDWaterShoreNE, Name: "water_shore_ne", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNE,
	},
	GIDWaterShoreNW: {
		GID: GIDWaterShoreNW, Name: "water_shore_nw", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNW,
	},
	GIDWaterShoreSW: {
		GID: GIDWaterShoreSW, Name: "water_shore_sw", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSW,
	},
	GIDWaterShoreSE: {
		GID: GIDWaterShoreSE, Name: "water_shore_se", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSE,
	},

	GIDWaterShoreNEInner: {
		GID: GIDWaterShoreNEInner, Name: "water_shore_ne_inner", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNEInner,
	},
	GIDWaterShoreNWInner: {
		GID: GIDWaterShoreNWInner, Name: "water_shore_nw_inner", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 8, W: 8, H: 8}}, VectorDraw: drawShoreNWInner,
	},
	GIDWaterShoreSWInner: {
		GID: GIDWaterShoreSWInner, Name: "water_shore_sw_inner", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSWInner,
	},
	GIDWaterShoreSEInner: {
		GID: GIDWaterShoreSEInner, Name: "water_shore_se_inner", Tags: []TileTag{TagSolid, TagWater, TagWaterShore}, Surface: waterSurface, SwatchColor: color.RGBA{0x2a, 0x4a, 0x8a, 0xff},
		SolidRects: []geom.Rect{{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 0, W: 8, H: 8}}, VectorDraw: drawShoreSEInner,
	},
}

// DefOf returns the registered definition for a GID.
func DefOf(gid int) Tile {
	if d, ok := defs[gid]; ok {
		return d
	}
	return Tile{GID: gid, Name: "unknown", Tags: []TileTag{TagSolid}, SwatchColor: color.RGBA{0x30, 0x30, 0x38, 0xff}}
}

// RegisteredGIDs returns a slice of all registered tile GIDs.
func RegisteredGIDs() []int {
	gids := make([]int, 0, len(defs))
	for g := range defs {
		gids = append(gids, g)
	}
	return gids
}
