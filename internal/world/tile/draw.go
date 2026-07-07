package tile

import "image/color"

var (
	tileGrassColor     = color.RGBA{0x55, 0x88, 0x55, 0xff}
	tileWaterColor     = color.RGBA{0x2a, 0x4a, 0x8a, 0xff}
	tileShoreLineColor = color.RGBA{0xe0, 0xd0, 0xa0, 0xff}
	tileWallColor      = color.RGBA{0x40, 0x40, 0x50, 0xff}
	tileWallEdgeColor  = color.RGBA{0x20, 0x20, 0x28, 0xff}
)

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
	c.StrokeLine(x+w*0.2, y+h*0.3, x+w*0.4, y+h*0.3, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff})
	c.StrokeLine(x+w*0.5, y+h*0.7, x+w*0.8, y+h*0.7, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff})
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
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, 1, tileShoreLineColor)
}

func drawShoreBottom(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h*0.5, tileWaterColor)
	c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, 1, tileShoreLineColor)
}

func drawShoreLeft(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, 1, tileShoreLineColor)
}

func drawShoreRight(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, 1, tileShoreLineColor)
}

func drawShoreNW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreNE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y+h)
	lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreSW(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreSE(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x, y+h*0.5)
	lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreNWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	c.FillRect(x, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y)
	lp.QuadTo(x, y, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreNEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w, y, x+w*0.5, y)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreSWInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h*0.5, tileWaterColor)
	c.FillRect(x+w*0.5, y, w*0.5, h, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w*0.5, y+h)
	lp.QuadTo(x, y+h, x, y+h*0.5)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
}

func drawShoreSEInner(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w*0.5, h, tileWaterColor)
	c.FillRect(x+w*0.5, y, w*0.5, h*0.5, tileWaterColor)
	var lp Path
	lp.MoveTo(x+w, y+h*0.5)
	lp.QuadTo(x+w, y+h, x+w*0.5, y+h)
	c.DrawPath(lp, tileShoreLineColor, false, 1)
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
