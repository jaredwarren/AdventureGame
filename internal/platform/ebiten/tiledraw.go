// tiledraw.go — vector fallback draw functions for each tile GID.
//
// To add a new tile: register a new GID constant in world/tiles.go, add a
// TileDef entry in world/tiledef.go, then add one entry to tileDrawers below.
// No changes to renderer.go are required.
package ebitenplat

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/world"
)

// Shared palette used across tile draw functions.
var (
	tileGrassColor     = color.RGBA{0x2b, 0x4a, 0x2b, 0xff}
	tileWaterColor     = color.RGBA{0x2a, 0x4a, 0x8a, 0xff}
	tileShoreLineColor = color.RGBA{0xe0, 0xd0, 0xa0, 0xff}
)

// tileDrawers maps a tile GID to its vector-fallback draw function.
// drawVectorTile in renderer.go looks up the GID here; if absent it falls
// back to drawing TileDef.SwatchColor.
var tileDrawers = map[int]func(dst *ebiten.Image, x, y, w, h float32){
	world.GIDEmpty: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
	},

	world.GIDGrass: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileGrassColor, false)
		vector.StrokeLine(dst, x+w*0.3, y+h*0.7, x+w*0.3, y+h*0.4, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.3, y+h*0.7, x+w*0.45, y+h*0.5, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.7, y+h*0.5, x+w*0.7, y+h*0.2, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.3, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
	},

	world.GIDWall: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x40, 0x40, 0x50, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x60, 0x60, 0x70, 0xff}, false)
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.25, y+h*0.5, x+w*0.25, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.75, y+h*0.5, x+w*0.75, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
	},

	world.GIDCracked: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8b, 0x6a, 0x4a, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.2, y+h*0.2, x+w*0.5, y+h*0.4, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.5, y+h*0.4, x+w*0.4, y+h*0.7, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.4, y+h*0.7, x+w*0.8, y+h*0.8, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)
	},

	world.GIDDoor: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x3a, 0x5a, 0x3a, 0xff}, false)
		vector.FillRect(dst, x+w*0.2, y+h*0.2, w*0.6, h*0.8, color.RGBA{0x15, 0x25, 0x15, 0xff}, false)
		vector.StrokeRect(dst, x+w*0.2, y+h*0.2, w*0.6, h*0.8, 1, color.RGBA{0xe0, 0xc0, 0x30, 0xff}, false)
	},

	world.GIDWater: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		vector.StrokeLine(dst, x+w*0.2, y+h*0.3, x+w*0.4, y+h*0.3, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.5, y+h*0.7, x+w*0.8, y+h*0.7, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff}, false)
	},

	world.GIDLock: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x6a, 0x2a, 0x7a, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8a, 0x4a, 0x9a, 0xff}, false)
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.18, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
		vector.FillRect(dst, x+w*0.42, y+h*0.4, w*0.16, h*0.3, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.07, color.RGBA{0, 0, 0, 255}, true)
		vector.FillRect(dst, x+w*0.47, y+h*0.42, w*0.06, h*0.18, color.RGBA{0, 0, 0, 255}, false)
	},

	world.GIDFloor2: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x3a, 0x3a, 0x44, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x4a, 0x4a, 0x54, 0xff}, false)
		vector.StrokeRect(dst, x+w*0.25, y+h*0.25, w*0.5, h*0.5, 1, color.RGBA{0x2a, 0x2a, 0x34, 0xff}, false)
	},

	world.GIDDirtPath: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x8b, 0x6a, 0x3a, 0xff}, false)
		// Worn track lines running left-right
		vector.StrokeLine(dst, x, y+h*0.35, x+w, y+h*0.35, 1, color.RGBA{0x6b, 0x4e, 0x28, 0xff}, false)
		vector.StrokeLine(dst, x, y+h*0.65, x+w, y+h*0.65, 1, color.RGBA{0x6b, 0x4e, 0x28, 0xff}, false)
		// Scattered pebble dots
		vector.FillRect(dst, x+w*0.2, y+h*0.2, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff}, false)
		vector.FillRect(dst, x+w*0.7, y+h*0.5, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff}, false)
		vector.FillRect(dst, x+w*0.45, y+h*0.75, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff}, false)
	},

	world.GIDTree: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileGrassColor, false)
		vector.FillRect(dst, x+w*0.4, y+h*0.6, w*0.2, h*0.3, color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, false)
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.3, color.RGBA{0x2d, 0x7a, 0x2a, 0xff}, true)
		vector.StrokeCircle(dst, x+w*0.5, y+h*0.4, w*0.3, 1, color.RGBA{0x1d, 0x5a, 0x1a, 0xff}, true)
	},

	// Straight shore transitions
	world.GIDWaterShoreTop: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h*0.5, tileGrassColor, false)
		vector.FillRect(dst, x, y+h*0.5, w, h*0.5, tileWaterColor, false)
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, tileShoreLineColor, false)
	},
	world.GIDWaterShoreBottom: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h*0.5, tileWaterColor, false)
		vector.FillRect(dst, x, y+h*0.5, w, h*0.5, tileGrassColor, false)
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, tileShoreLineColor, false)
	},
	world.GIDWaterShoreLeft: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w*0.5, h, tileGrassColor, false)
		vector.FillRect(dst, x+w*0.5, y, w*0.5, h, tileWaterColor, false)
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h, 1, tileShoreLineColor, false)
	},
	world.GIDWaterShoreRight: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w*0.5, h, tileWaterColor, false)
		vector.FillRect(dst, x+w*0.5, y, w*0.5, h, tileGrassColor, false)
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h, 1, tileShoreLineColor, false)
	},

	// Convex outer corners
	world.GIDWaterShoreNW: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
		path.LineTo(x, y+h)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreNE: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h)
		path.LineTo(x+w*0.5, y+h)
		path.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y+h)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreSW: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w*0.5, y)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
		path.LineTo(x+w, y+h)
		path.LineTo(x, y+h)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreSE: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x+w*0.5, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h)
		path.LineTo(x, y+h)
		path.LineTo(x, y+h*0.5)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x, y+h*0.5)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},

	// Concave inner corners
	world.GIDWaterShoreNWInner: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w*0.5, y)
		path.QuadTo(x, y, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y)
		linePath.QuadTo(x, y, x, y+h*0.5)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreNEInner: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x+w, y)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w, y, x+w*0.5, y)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w, y, x+w*0.5, y)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreSWInner: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x, y+h)
		path.LineTo(x+w*0.5, y+h)
		path.QuadTo(x, y+h, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y+h)
		linePath.QuadTo(x, y+h, x, y+h*0.5)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
	world.GIDWaterShoreSEInner: func(dst *ebiten.Image, x, y, w, h float32) {
		vector.FillRect(dst, x, y, w, h, tileWaterColor, false)
		var path vector.Path
		path.MoveTo(x+w, y+h)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w, y+h, x+w*0.5, y+h)
		path.Close()
		drawPath(dst, &path, tileGrassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w, y+h, x+w*0.5, y+h)
		drawPath(dst, &linePath, tileShoreLineColor, false, 1)
	},
}
