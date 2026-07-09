package tile

import (
	"image/color"
	"math"
)

var (
	tileGrassColor     = color.RGBA{0x55, 0x88, 0x55, 0xff}
	tileWaterColor     = color.RGBA{0x2a, 0x4a, 0x8a, 0xff}
	tileShoreLineColor = color.RGBA{0xe0, 0xd0, 0xa0, 0xff}
	tileWallColor      = color.RGBA{0x40, 0x40, 0x50, 0xff}
	tileWallEdgeColor  = color.RGBA{0x20, 0x20, 0x28, 0xff}
)

const tileShoreLineWidth float32 = 2.0

func drawEmpty(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x00, 0x00, 0x00, 0xff})
}

func drawGrass(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileGrassColor)
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.3, y+h*0.4, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff})
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.45, y+h*0.5, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff})
	c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.7, y+h*0.2, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff})
	c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.3, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff})
}

func drawWall(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileWallColor)
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x20, 0x20, 0x28, 0xff})
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
	c.StrokeLine(x+w*0.25, y+h*0.5, x+w*0.25, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
	c.StrokeLine(x+w*0.75, y+h*0.5, x+w*0.75, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
}

func drawWallTop(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w, h*0.5, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.25, y+h*0.5, x+w*0.25, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
	c.StrokeLine(x+w*0.75, y+h*0.5, x+w*0.75, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
}

func drawWallBottom(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h*0.5, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
}

func drawWallLeft(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWallColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
}

func drawWallRight(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWallColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, 1, tileWallEdgeColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.5, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff})
}

func drawWallNW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.5, y+h, 1, tileWallEdgeColor)
}

func drawWallNE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.5, y+h, 1, tileWallEdgeColor)
}

func drawWallSW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
}

func drawWallSE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
}

func drawWallNWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWallColor)
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
}

func drawWallNEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWallColor)
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
}

func drawWallSWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWallColor)
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWallColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.5, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.5, y+h, 1, tileWallEdgeColor)
}

func drawWallSEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWallColor)
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWallColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w, y+h*0.5, 1, tileWallEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.5, y+h, 1, tileWallEdgeColor)
}

func drawCracked(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x6b, 0x4a, 0x2a, 0xff})
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8b, 0x6a, 0x4a, 0xff})
	c.StrokeLine(x+w*0.2, y+h*0.2, x+w*0.5, y+h*0.4, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff})
	c.StrokeLine(x+w*0.5, y+h*0.4, x+w*0.4, y+h*0.7, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff})
	c.StrokeLine(x+w*0.4, y+h*0.7, x+w*0.8, y+h*0.8, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff})
}

func drawDoor(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x3a, 0x5a, 0x3a, 0xff})
	c.FillRect(x+w*0.2, y+h*0.2, w*0.6, h*0.8, color.RGBA{0x15, 0x25, 0x15, 0xff})
	c.StrokeRect(x+w*0.2, y+h*0.2, w*0.6, h*0.8, 1, color.RGBA{0xe0, 0xc0, 0x30, 0xff})
}

func drawWater(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileWaterColor)

	tick := c.Tick()
	tx, ty := c.GridPos()

	// Subtle wave animation (horizontal sway)
	angle1 := float64(tick+tx*17+ty*37) * 2 * math.Pi / 180.0
	offset1 := float32(math.Sin(angle1)) * w * 0.08

	angle2 := float64(tick+tx*53+ty*29) * 2 * math.Pi / 240.0
	offset2 := float32(math.Sin(angle2)) * w * 0.08

	c.StrokeLine(x+w*0.2+offset1, y+h*0.3, x+w*0.4+offset1, y+h*0.3, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff})
	c.StrokeLine(x+w*0.5+offset2, y+h*0.7, x+w*0.8+offset2, y+h*0.7, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff})
}

func drawLock(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x6a, 0x2a, 0x7a, 0xff})
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8a, 0x4a, 0x9a, 0xff})
	c.FillCircle(x+w*0.5, y+h*0.4, w*0.18, color.RGBA{0xff, 0xd7, 0x00, 0xff})
	c.FillRect(x+w*0.42, y+h*0.4, w*0.16, h*0.3, color.RGBA{0xff, 0xd7, 0x00, 0xff})
	c.FillCircle(x+w*0.5, y+h*0.4, w*0.07, color.RGBA{0, 0, 0, 255})
	c.FillRect(x+w*0.47, y+h*0.42, w*0.06, h*0.18, color.RGBA{0, 0, 0, 255})
}

func drawFloor2(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x50, 0x50, 0x5c, 0xff})
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x65, 0x65, 0x75, 0xff})
	c.StrokeRect(x+w*0.25, y+h*0.25, w*0.5, h*0.5, 1, color.RGBA{0x3c, 0x3c, 0x46, 0xff})
}

func drawDirtPath(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x8b, 0x6a, 0x3a, 0xff})
	c.StrokeLine(x, y+h*0.35, x+w, y+h*0.35, 1, color.RGBA{0x6b, 0x4e, 0x28, 0xff})
	c.StrokeLine(x, y+h*0.65, x+w, y+h*0.65, 1, color.RGBA{0x6b, 0x4e, 0x28, 0xff})
	c.FillRect(x+w*0.2, y+h*0.2, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff})
	c.FillRect(x+w*0.7, y+h*0.5, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff})
	c.FillRect(x+w*0.45, y+h*0.75, 2, 2, color.RGBA{0xa8, 0x8a, 0x58, 0xff})
}

func drawTree(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.4, y+h*0.6, w*0.2, h*0.3, color.RGBA{0x6b, 0x4a, 0x2a, 0xff})
	c.FillCircle(x+w*0.5, y+h*0.4, w*0.3, color.RGBA{0x2d, 0x7a, 0x2a, 0xff})
	c.StrokeCircle(x+w*0.5, y+h*0.4, w*0.3, 1, color.RGBA{0x1d, 0x5a, 0x1a, 0xff})
}

func drawShoreTop(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w, h*0.5, tileWaterColor)
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, tileShoreLineWidth, tileShoreLineColor)
}

func drawShoreBottom(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h*0.5, tileWaterColor)
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, tileShoreLineWidth, tileShoreLineColor)
}

func drawShoreLeft(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, tileShoreLineWidth, tileShoreLineColor)
}

func drawShoreRight(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, tileShoreLineWidth, tileShoreLineColor)
}

func drawShoreNW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreNE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y+h)
	lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreSW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreSE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreNWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y)
	lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreNEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreSWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWaterColor)
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y+h)
	lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

func drawShoreSEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
	c.DrawPath(lp, tileShoreLineColor, false, tileShoreLineWidth)
}

var (
	tileRockColor     = color.RGBA{0x8b, 0x4d, 0x3a, 0xff}
	tileRockEdgeColor = color.RGBA{0x5a, 0x2a, 0x1d, 0xff}
)

func drawRock(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileRockColor)
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, tileRockEdgeColor)
	// Draw craggy/jagged cracks inside
	c.StrokeLine(x+w*0.2, y+h*0.3, x+w*0.4, y+h*0.4, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.4, y+h*0.4, x+w*0.3, y+h*0.7, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.7, y+h*0.2, x+w*0.6, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.6, y+h*0.5, x+w*0.8, y+h*0.8, 1, tileRockEdgeColor)
}

func drawRockTop(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w, h*0.5, tileRockColor)
	// Draw jagged boundary
	c.StrokeLine(x, y+h*0.5, x+w*0.3, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.3, y+h*0.45, x+w*0.6, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.6, y+h*0.55, x+w, y+h*0.5, 1, tileRockEdgeColor)
	// Crag lines inside
	c.StrokeLine(x+w*0.5, y+h*0.6, x+w*0.4, y+h*0.8, 1, tileRockEdgeColor)
}

func drawRockBottom(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h*0.5, tileRockColor)
	// Jagged boundary
	c.StrokeLine(x, y+h*0.5, x+w*0.4, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.4, y+h*0.55, x+w*0.7, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.7, y+h*0.45, x+w, y+h*0.5, 1, tileRockEdgeColor)
	// Crag lines
	c.StrokeLine(x+w*0.5, y+h*0.4, x+w*0.6, y+h*0.2, 1, tileRockEdgeColor)
}

func drawRockLeft(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileRockColor)
	// Jagged boundary
	c.StrokeLine(x+w*0.5, y, x+w*0.45, y+h*0.3, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.3, x+w*0.55, y+h*0.7, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.7, x+w*0.5, y+h, 1, tileRockEdgeColor)
	// Crag lines
	c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.6, 1, tileRockEdgeColor)
}

func drawRockRight(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileRockColor)
	// Jagged boundary
	c.StrokeLine(x+w*0.5, y, x+w*0.55, y+h*0.4, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.4, x+w*0.45, y+h*0.8, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.8, x+w*0.5, y+h, 1, tileRockEdgeColor)
	// Crag lines
	c.StrokeLine(x+w*0.2, y+h*0.5, x+w*0.3, y+h*0.4, 1, tileRockEdgeColor)
}

func drawRockNW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileRockColor)
	// Jagged edges
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.75, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.75, y+h*0.45, x+w, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.45, y+h*0.75, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.75, x+w*0.5, y+h, 1, tileRockEdgeColor)
}

func drawRockNE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileRockColor)
	// Jagged edges
	c.StrokeLine(x, y+h*0.5, x+w*0.25, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.25, y+h*0.45, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.55, y+h*0.75, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.75, x+w*0.5, y+h, 1, tileRockEdgeColor)
}

func drawRockSW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileRockColor)
	// Jagged edges
	c.StrokeLine(x+w*0.5, y, x+w*0.45, y+h*0.25, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.25, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.75, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.75, y+h*0.55, x+w, y+h*0.5, 1, tileRockEdgeColor)
}

func drawRockSE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileRockColor)
	// Jagged edges
	c.StrokeLine(x+w*0.5, y, x+w*0.55, y+h*0.25, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.25, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.25, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.25, y+h*0.55, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
}

func drawRockNEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileRockColor)
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileRockColor)
	// Jagged inner corner boundaries
	c.StrokeLine(x+w*0.5, y, x+w*0.45, y+h*0.25, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.25, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.75, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.75, y+h*0.55, x+w, y+h*0.5, 1, tileRockEdgeColor)
}

func drawRockNWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileRockColor)
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileRockColor)
	// Jagged inner corner boundaries
	c.StrokeLine(x+w*0.5, y, x+w*0.55, y+h*0.25, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.25, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x, y+h*0.5, x+w*0.25, y+h*0.55, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.25, y+h*0.55, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
}

func drawRockSWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileRockColor)
	c.FillRect(x, y, w*0.5, h*0.5, tileRockColor)
	// Jagged inner corner boundaries
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.75, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.75, y+h*0.45, x+w, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.55, y+h*0.75, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.55, y+h*0.75, x+w*0.5, y+h, 1, tileRockEdgeColor)
}

func drawRockSEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileRockColor)
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileRockColor)
	// Jagged inner corner boundaries
	c.StrokeLine(x, y+h*0.5, x+w*0.25, y+h*0.45, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.25, y+h*0.45, x+w*0.5, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.5, y+h*0.5, x+w*0.45, y+h*0.75, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.45, y+h*0.75, x+w*0.5, y+h, 1, tileRockEdgeColor)
}

func drawQuicksand(c Canvas, x, y, w, h float32) {
	// Base color of tan/light-brown sand
	c.FillRect(x, y, w, h, color.RGBA{0xdf, 0xc7, 0x94, 0xff})
	// Draw subtle wavy lines to simulate shifting quicksand texture
	c.StrokeLine(x+w*0.1, y+h*0.2, x+w*0.4, y+h*0.2, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
	c.StrokeLine(x+w*0.6, y+h*0.4, x+w*0.9, y+h*0.4, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.7, y+h*0.7, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
}

func drawMud(c Canvas, x, y, w, h float32) {
	// Dark brown mud base
	c.FillRect(x, y, w, h, color.RGBA{0x6b, 0x4c, 0x35, 0xff})
	// Draw mud spots
	c.FillCircle(x+w*0.3, y+h*0.4, w*0.12, color.RGBA{0x4a, 0x32, 0x22, 0xff})
	c.FillCircle(x+w*0.7, y+h*0.6, w*0.15, color.RGBA{0x4a, 0x32, 0x22, 0xff})
}

func drawIce(c Canvas, x, y, w, h float32) {
	// Light blue ice base
	c.FillRect(x, y, w, h, color.RGBA{0xa0, 0xd8, 0xef, 0xff})
	// Highlights and cracks
	c.StrokeLine(x+w*0.2, y+h*0.2, x+w*0.5, y+h*0.5, 1.5, color.RGBA{0xff, 0xff, 0xff, 0xff})
	c.StrokeLine(x+w*0.6, y+h*0.7, x+w*0.8, y+h*0.9, 1, color.RGBA{0xff, 0xff, 0xff, 0xff})
}

const (
	lavaBubble1Cycle       = 360
	lavaBubble1GrowStart   = 270
	lavaBubble1PopStart    = 330
	lavaBubble1PopDuration = 10

	lavaBubble2Cycle       = 480
	lavaBubble2GrowStart   = 350
	lavaBubble2PopStart    = 430
	lavaBubble2PopDuration = 10
)

func drawLava(c Canvas, x, y, w, h float32) {
	// Red-orange lava base
	c.FillRect(x, y, w, h, color.RGBA{0xd3, 0x54, 0x00, 0xff})

	tick := c.Tick()
	tx, ty := c.GridPos()

	// Bubble 1 (Orange bubble in bottom-left quadrant)
	phase1 := int(tx*37 + ty*17)
	t1 := (tick + phase1) % lavaBubble1Cycle
	if t1 >= lavaBubble1GrowStart && t1 < lavaBubble1PopStart {
		progress := float32(t1-lavaBubble1GrowStart) / float32(lavaBubble1PopStart-lavaBubble1GrowStart)
		c.FillCircle(x+w*0.3, y+h*0.6, progress*w*0.15, color.RGBA{0xe6, 0x7e, 0x22, 0xff})
	} else if t1 >= lavaBubble1PopStart && t1 < lavaBubble1PopStart+lavaBubble1PopDuration {
		c.StrokeCircle(x+w*0.3, y+h*0.6, w*0.18, 1, color.RGBA{0xe6, 0x7e, 0x22, 0xff})
	}

	// Bubble 2 (Yellow bubble in top-right quadrant)
	phase2 := int(tx*53 + ty*29)
	t2 := (tick + phase2) % lavaBubble2Cycle
	if t2 >= lavaBubble2GrowStart && t2 < lavaBubble2PopStart {
		progress := float32(t2-lavaBubble2GrowStart) / float32(lavaBubble2PopStart-lavaBubble2GrowStart)
		c.FillCircle(x+w*0.7, y+h*0.3, progress*w*0.18, color.RGBA{0xf1, 0xc4, 0x0f, 0xff})
	} else if t2 >= lavaBubble2PopStart && t2 < lavaBubble2PopStart+lavaBubble2PopDuration {
		c.StrokeCircle(x+w*0.7, y+h*0.3, w*0.21, 1, color.RGBA{0xf1, 0xc4, 0x0f, 0xff})
	}
}

func drawSign(c Canvas, x, y, w, h float32) {
	// Grass background
	c.FillRect(x, y, w, h, tileGrassColor)

	// Wood post in the center
	postW := w * 0.15
	postH := h * 0.6
	postX := x + (w-postW)*0.5
	postY := y + h*0.4
	c.FillRect(postX, postY, postW, postH, color.RGBA{0x6a, 0x3d, 0x1b, 0xff})

	// Plaque at the top
	plaqueW := w * 0.8
	plaqueH := h * 0.4
	plaqueX := x + w*0.1
	plaqueY := y + h*0.1
	c.FillRect(plaqueX, plaqueY, plaqueW, plaqueH, color.RGBA{0x8b, 0x5a, 0x2b, 0xff})
	c.StrokeRect(plaqueX, plaqueY, plaqueW, plaqueH, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})

	// Text lines/grain
	c.StrokeLine(plaqueX+w*0.1, plaqueY+h*0.15, plaqueX+plaqueW-w*0.1, plaqueY+h*0.15, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})
	c.StrokeLine(plaqueX+w*0.15, plaqueY+h*0.27, plaqueX+plaqueW-w*0.15, plaqueY+h*0.27, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})
}

func drawSand(c Canvas, x, y, w, h float32) {
	// A warm, sunny desert sand base color
	c.FillRect(x, y, w, h, color.RGBA{0xe5, 0xd3, 0xa3, 0xff})
	// Draw small sand grains
	c.FillRect(x+w*0.2, y+h*0.15, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.75, y+h*0.3, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.4, y+h*0.5, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.8, y+h*0.75, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.15, y+h*0.8, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	// Draw horizontal wind ripple lines (stable, not wavy/shifting like quicksand)
	c.StrokeLine(x+w*0.15, y+h*0.35, x+w*0.45, y+h*0.35, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.StrokeLine(x+w*0.55, y+h*0.65, x+w*0.85, y+h*0.65, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
}
