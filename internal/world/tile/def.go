// Package tile defines ground tile GIDs, collision rules, and persistence keys.
package tile

import (
	"image/color"
	"slices"

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
	TagWall         TileTag = "wall"
	TagWater        TileTag = "water"
	TagWaterShore   TileTag = "water_shore"
	TagFloor        TileTag = "floor"
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
func (d Tile) Wall() bool       { return d.HasTag(TagWall) }
func (d Tile) Water() bool      { return d.HasTag(TagWater) }
func (d Tile) WaterShore() bool { return d.HasTag(TagWaterShore) }
func (d Tile) IsFloor() bool {
	return d.HasTag(TagFloor) || (!d.Solid() && !d.Water() && d.GID != GIDEmpty)
}
func (d Tile) IsWall() bool  { return d.Solid() }
func (d Tile) IsWater() bool { return d.Water() }
func (d Tile) IsLand() bool  { return !d.Water() && !d.Solid() }

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
	SpeedMultiplier: 1, // was 0.5
	Friction:        1, // was 0.8
}

var wallColor = color.RGBA{0x40, 0x40, 0x50, 0xff}
var rockColor = color.RGBA{0x8b, 0x4d, 0x3a, 0xff}
var dirtColor = color.RGBA{0x8b, 0x6a, 0x3a, 0xff}

var defs = make(map[int]Tile)

func init() {
	// Register standalone tiles
	RegisterSingleTile(Tile{GID: GIDEmpty, Name: "empty", SwatchColor: color.RGBA{0x00, 0x00, 0x00, 0xff}, VectorDraw: drawEmpty})
	RegisterSingleTile(Tile{GID: GIDGrass, Name: "grass", Tags: []TileTag{TagFloor}, FloorWeight: 0.58, SwatchColor: color.RGBA{0x55, 0x88, 0x55, 0xff}, VectorDraw: drawGrass})
	RegisterSingleTile(Tile{GID: GIDCracked, Name: "cracked", Tags: []TileTag{TagDestructible, TagWall}, DamageKinds: []DamageKind{DamageBomb}, SwatchColor: color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, VectorDraw: drawCracked})
	RegisterSingleTile(Tile{GID: GIDDoor, Name: "door", Tags: []TileTag{TagDoor}, SwatchColor: color.RGBA{0x3a, 0x5a, 0x3a, 0xff}, VectorDraw: drawDoor})
	RegisterSingleTile(Tile{GID: GIDLock, Name: "lock", Tags: []TileTag{TagLock}, OpenableByKey: true, SwatchColor: color.RGBA{0x6a, 0x2a, 0x7a, 0xff}, VectorDraw: drawLock})
	RegisterSingleTile(Tile{GID: GIDFloor2, Name: "floor2", Tags: []TileTag{TagFloor}, FloorWeight: 0.42, SwatchColor: color.RGBA{0x50, 0x50, 0x5c, 0xff}, VectorDraw: drawFloor2})
	RegisterSingleTile(Tile{GID: GIDTree, Name: "tree", Tags: []TileTag{TagIgnitable}, DamageKinds: []DamageKind{DamageFire}, SwatchColor: color.RGBA{0x2d, 0x5a, 0x2a, 0xff}, VectorDraw: drawTree})
	RegisterSingleTile(Tile{GID: GIDQuicksand, Name: "quicksand", Tags: []TileTag{TagFloor}, Surface: QuicksandSurface, SwatchColor: color.RGBA{0xdf, 0xc7, 0x94, 0xff}, VectorDraw: drawQuicksand})
	RegisterSingleTile(Tile{GID: GIDMud, Name: "mud", Tags: []TileTag{TagFloor}, Surface: MudSurface, SwatchColor: color.RGBA{0x6b, 0x4c, 0x35, 0xff}, VectorDraw: drawMud})
	RegisterSingleTile(Tile{GID: GIDIce, Name: "ice", Tags: []TileTag{TagFloor}, Surface: IceSurface, SwatchColor: color.RGBA{0xa0, 0xd8, 0xef, 0xff}, VectorDraw: drawIce})
	RegisterSingleTile(Tile{GID: GIDLava, Name: "lava", Tags: []TileTag{TagFloor}, Surface: LavaSurface, SwatchColor: color.RGBA{0xd3, 0x54, 0x00, 0xff}, VectorDraw: drawLava})
	RegisterSingleTile(Tile{GID: GIDSign, Name: "sign", Tags: []TileTag{TagSolid, TagWall}, SwatchColor: color.RGBA{0x8b, 0x5a, 0x2b, 0xff}, VectorDraw: drawSign})
	RegisterSingleTile(Tile{GID: GIDSand, Name: "sand", Tags: []TileTag{TagFloor}, Surface: DefaultSurface, SwatchColor: color.RGBA{0xe5, 0xd3, 0xa3, 0xff}, VectorDraw: drawSand})

	// Register 13-tile families
	RegisterFamily(FamilyConfig{
		Name:     "wall",
		Category: "wall",
		BaseGID:  GIDWall,
		ExplicitGIDs: [13]int{
			GIDWall,
			GIDWallTop, GIDWallBottom, GIDWallLeft, GIDWallRight,
			GIDWallNE, GIDWallNW, GIDWallSW, GIDWallSE,
			GIDWallNEInner, GIDWallNWInner, GIDWallSWInner, GIDWallSEInner,
		},
		Kind:      FamilyWall,
		Style:     TileStyle{FillColor: wallColor, EdgeColor: tileWallEdgeColor, LineWidth: 1.0, BaseDrawer: drawWall},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:     "water",
		Category: "water",
		BaseGID:  GIDWater,
		ExplicitGIDs: [13]int{
			GIDWater,
			GIDWaterShoreTop, GIDWaterShoreBottom, GIDWaterShoreLeft, GIDWaterShoreRight,
			GIDWaterShoreNE, GIDWaterShoreNW, GIDWaterShoreSW, GIDWaterShoreSE,
			GIDWaterShoreNEInner, GIDWaterShoreNWInner, GIDWaterShoreSWInner, GIDWaterShoreSEInner,
		},
		ExplicitNames: [13]string{
			"water",
			"water_shore_top", "water_shore_bottom", "water_shore_left", "water_shore_right",
			"water_shore_ne", "water_shore_nw", "water_shore_sw", "water_shore_se",
			"water_shore_ne_inner", "water_shore_nw_inner", "water_shore_sw_inner", "water_shore_se_inner",
		},
		Kind:      FamilyWater,
		Surface:   waterSurface,
		Style:     TileStyle{FillColor: tileWaterColor, EdgeColor: tileShoreLineColor, LineWidth: tileShoreLineWidth, BaseDrawer: drawWater},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:     "rock",
		Category: "rock",
		BaseGID:  GIDRock,
		ExplicitGIDs: [13]int{
			GIDRock,
			GIDRockTop, GIDRockBottom, GIDRockLeft, GIDRockRight,
			GIDRockNE, GIDRockNW, GIDRockSW, GIDRockSE,
			GIDRockNEInner, GIDRockNWInner, GIDRockSWInner, GIDRockSEInner,
		},
		Kind:      FamilyRock,
		Style:     TileStyle{FillColor: rockColor, EdgeColor: tileRockEdgeColor, LineWidth: 1.0, HasDetail: true, DetailColor: tileRockEdgeColor, BaseDrawer: drawRock},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:     "dirt_path",
		Category: "dirt_path",
		BaseGID:  GIDDirtPath,
		ExplicitGIDs: [13]int{
			GIDDirtPath,
			GIDDirtPathTop, GIDDirtPathBottom, GIDDirtPathLeft, GIDDirtPathRight,
			GIDDirtPathNE, GIDDirtPathNW, GIDDirtPathSW, GIDDirtPathSE,
			GIDDirtPathNEInner, GIDDirtPathNWInner, GIDDirtPathSWInner, GIDDirtPathSEInner,
		},
		Kind:           FamilyFloor,
		FloorWeight:    0.3,
		Style:          TileStyle{FillColor: dirtColor, EdgeColor: tileDirtPathEdgeColor, LineWidth: 1.0, HasDetail: true, DetailColor: tileDirtPebbleColor, BaseDrawer: drawDirtPath},
		VariantDrawers: dirtPathVariantDrawers,
		Collapsed:      false,
	})

	RegisterFamily(FamilyConfig{
		Name:           "cobble_path",
		Label:          "Cobblestone Path",
		Category:       "cobble_path",
		BaseGID:        GIDCobblePath,
		Kind:           FamilyFloor,
		FloorWeight:    0.3,
		Style:          TileStyle{FillColor: tileCobblePathColor, EdgeColor: tileCobblePathEdgeColor, LineWidth: 1.0, HasDetail: true, DetailColor: tileCobblePebbleColor, BaseDrawer: drawCobblePath},
		VariantDrawers: cobblePathVariantDrawers,
		Collapsed:      false,
	})

	RegisterFamily(FamilyConfig{
		Name:        "sand_path",
		Label:       "Sand & Shore",
		Category:    "sand_path",
		BaseGID:     GIDSandPath,
		Kind:        FamilyFloor,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0xe5, 0xd3, 0xa3, 0xff},
			EdgeColor:   color.RGBA{0xc8, 0xb6, 0x86, 0xff},
			DetailColor: color.RGBA{0xc8, 0xb6, 0x86, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawSand,
		},
		Collapsed: false,
	})

	RegisterFamily(FamilyConfig{
		Name:        "snow",
		Label:       "Snow & Alpine",
		Category:    "snow",
		BaseGID:     GIDSnow,
		Kind:        FamilyFloor,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0xf0, 0xf8, 0xff, 0xff},
			EdgeColor:   color.RGBA{0xb8, 0xd0, 0xe8, 0xff},
			DetailColor: color.RGBA{0xd4, 0xe4, 0xf4, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawSnow,
		},
		Collapsed: false,
	})

	RegisterFamily(FamilyConfig{
		Name:        "mud_path",
		Label:       "Mud & Swamp",
		Category:    "mud_path",
		BaseGID:     GIDMudPath,
		Kind:        FamilyFloor,
		Surface:     MudSurface,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0x6b, 0x4c, 0x35, 0xff},
			EdgeColor:   color.RGBA{0x4a, 0x32, 0x22, 0xff},
			DetailColor: color.RGBA{0x4a, 0x32, 0x22, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawMud,
		},
		Collapsed: false,
	})

	RegisterFamily(FamilyConfig{
		Name:        "dark_grass",
		Label:       "Dark Forest Earth",
		Category:    "dark_grass",
		BaseGID:     GIDDarkGrass,
		Kind:        FamilyFloor,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0x35, 0x5a, 0x35, 0xff},
			EdgeColor:   color.RGBA{0x22, 0x3d, 0x22, 0xff},
			DetailColor: color.RGBA{0x45, 0x6e, 0x45, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawDarkGrass,
		},
		Collapsed: false,
	})

	RegisterFamily(FamilyConfig{
		Name:      "deep_water",
		Label:     "Deep Ocean Water",
		Category:  "deep_water",
		BaseGID:   GIDDeepWater,
		Kind:      FamilyWater,
		Surface:   waterSurface,
		Style:     TileStyle{FillColor: color.RGBA{0x18, 0x2a, 0x5a, 0xff}, EdgeColor: color.RGBA{0x3a, 0x6a, 0xba, 0xff}, LineWidth: tileShoreLineWidth, BaseDrawer: drawDeepWater},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:      "lava_shore",
		Label:     "Lava & Magma",
		Category:  "lava",
		BaseGID:   GIDLavaShore,
		Kind:      FamilyWater,
		Surface:   LavaSurface,
		Style:     TileStyle{FillColor: color.RGBA{0xd3, 0x54, 0x00, 0xff}, EdgeColor: color.RGBA{0xf3, 0x9c, 0x12, 0xff}, LineWidth: tileShoreLineWidth, BaseDrawer: drawLava},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:      "swamp_water",
		Label:     "Murky Swamp Water",
		Category:  "swamp_water",
		BaseGID:   GIDSwampWater,
		Kind:      FamilyWater,
		Surface:   waterSurface,
		Style:     TileStyle{FillColor: color.RGBA{0x32, 0x44, 0x2e, 0xff}, EdgeColor: color.RGBA{0x5c, 0x7a, 0x44, 0xff}, LineWidth: tileShoreLineWidth, BaseDrawer: drawSwampWater},
		Collapsed: true,
	})

	RegisterFamily(FamilyConfig{
		Name:        "ice",
		Label:       "Ice & Frozen Lake",
		Category:    "ice",
		BaseGID:     GIDIceFamily,
		Kind:        FamilyFloor,
		Surface:     IceSurface,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0xa0, 0xd8, 0xef, 0xff},
			EdgeColor:   color.RGBA{0x7a, 0xb8, 0xd4, 0xff},
			DetailColor: color.RGBA{0xff, 0xff, 0xff, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawIce,
		},
		Collapsed: false,
	})

	RegisterFamily(FamilyConfig{
		Name:        "quicksand",
		Label:       "Quicksand",
		Category:    "quicksand",
		BaseGID:     GIDQuicksandFamily,
		Kind:        FamilyFloor,
		Surface:     QuicksandSurface,
		FloorWeight: 0.3,
		Style: TileStyle{
			FillColor:   color.RGBA{0xdf, 0xc7, 0x94, 0xff},
			EdgeColor:   color.RGBA{0xb8, 0x9e, 0x6a, 0xff},
			DetailColor: color.RGBA{0xb8, 0x9e, 0x6a, 0xff},
			LineWidth:   1.0,
			HasDetail:   true,
			BaseDrawer:  drawQuicksand,
		},
		Collapsed: false,
	})
}

// DefOf returns the registered definition for a GID.
func DefOf(gid int) Tile {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if d, ok := defs[gid]; ok {
		return d
	}
	return Tile{GID: gid, Name: "unknown", Tags: []TileTag{TagSolid}, SwatchColor: color.RGBA{0x30, 0x30, 0x38, 0xff}}
}

// RegisteredGIDs returns a slice of all registered tile GIDs in sorted order.
func RegisteredGIDs() []int {
	registryMu.RLock()
	defer registryMu.RUnlock()
	gids := make([]int, 0, len(defs))
	for g := range defs {
		gids = append(gids, g)
	}
	slices.Sort(gids)
	return gids
}
