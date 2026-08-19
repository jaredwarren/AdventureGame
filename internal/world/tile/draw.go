package tile

import (
	"image/color"
	"math"
)

var (
	tileGrassColor        = color.RGBA{0x55, 0x88, 0x55, 0xff}
	tileWaterColor        = color.RGBA{0x2a, 0x4a, 0x8a, 0xff}
	tileShoreLineColor    = color.RGBA{0xe0, 0xd0, 0xa0, 0xff}
	tileWallColor         = color.RGBA{0x40, 0x40, 0x50, 0xff}
	tileWallEdgeColor     = color.RGBA{0x20, 0x20, 0x28, 0xff}
	tileRockColor         = color.RGBA{0x8b, 0x4d, 0x3a, 0xff}
	tileRockEdgeColor     = color.RGBA{0x5a, 0x2a, 0x1d, 0xff}
	tileDirtPathColor     = color.RGBA{0x8b, 0x6a, 0x3a, 0xff}
	tileDirtPathEdgeColor = color.RGBA{0x6b, 0x4e, 0x28, 0xff}
	tileDirtPebbleColor   = color.RGBA{0xa8, 0x8a, 0x58, 0xff}
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

func drawRock(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileRockColor)
	c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.2, y+h*0.3, x+w*0.4, y+h*0.4, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.4, y+h*0.4, x+w*0.3, y+h*0.7, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.7, y+h*0.2, x+w*0.6, y+h*0.5, 1, tileRockEdgeColor)
	c.StrokeLine(x+w*0.6, y+h*0.5, x+w*0.8, y+h*0.8, 1, tileRockEdgeColor)
}

func drawDirtPath(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, tileDirtPathColor)
	c.StrokeLine(x, y+h*0.35, x+w, y+h*0.35, 1, tileDirtPathEdgeColor)
	c.StrokeLine(x, y+h*0.65, x+w, y+h*0.65, 1, tileDirtPathEdgeColor)
	c.FillRect(x+w*0.2, y+h*0.2, 2, 2, tileDirtPebbleColor)
	c.FillRect(x+w*0.7, y+h*0.5, 2, 2, tileDirtPebbleColor)
	c.FillRect(x+w*0.45, y+h*0.75, 2, 2, tileDirtPebbleColor)
}

var (
	tileCobblePathColor     = color.RGBA{0x72, 0x76, 0x82, 0xff} // Primary round stone color
	tileCobblePathEdgeColor = color.RGBA{0x3a, 0x3d, 0x45, 0xff} // Dark mortar / gap color
	tileCobblePebbleColor   = color.RGBA{0x92, 0x98, 0xa6, 0xff} // Top highlight on stones
	tileCobbleDarkStone     = color.RGBA{0x5e, 0x62, 0x6e, 0xff} // Darker stone variant
)

func drawCobblePath(c Canvas, x, y, w, h float32) {
	// Dark mortar background
	c.FillRect(x, y, w, h, tileCobblePathEdgeColor)

	// Round stone 1 (Top-Left, medium-large)
	c.FillCircle(x+w*0.30, y+h*0.30, w*0.22, tileCobblePathColor)
	c.StrokeCircle(x+w*0.30, y+h*0.30, w*0.22, 1, tileCobblePathEdgeColor)
	c.FillCircle(x+w*0.26, y+h*0.26, w*0.08, tileCobblePebbleColor)

	// Round stone 2 (Top-Right, darker varied round stone)
	c.FillCircle(x+w*0.75, y+h*0.25, w*0.19, tileCobbleDarkStone)
	c.StrokeCircle(x+w*0.75, y+h*0.25, w*0.19, 1, tileCobblePathEdgeColor)
	c.FillCircle(x+w*0.72, y+h*0.22, w*0.06, tileCobblePebbleColor)

	// Round stone 3 (Bottom-Left, medium)
	c.FillCircle(x+w*0.26, y+h*0.74, w*0.18, tileCobbleDarkStone)
	c.StrokeCircle(x+w*0.26, y+h*0.74, w*0.18, 1, tileCobblePathEdgeColor)
	c.FillCircle(x+w*0.24, y+h*0.71, w*0.06, tileCobblePebbleColor)

	// Round stone 4 (Bottom-Right, large round stone)
	c.FillCircle(x+w*0.72, y+h*0.72, w*0.23, tileCobblePathColor)
	c.StrokeCircle(x+w*0.72, y+h*0.72, w*0.23, 1, tileCobblePathEdgeColor)
	c.FillCircle(x+w*0.68, y+h*0.68, w*0.09, tileCobblePebbleColor)

	// Round stone 5 (Center filler pebble)
	c.FillCircle(x+w*0.50, y+h*0.50, w*0.11, tileCobblePathColor)
	c.StrokeCircle(x+w*0.50, y+h*0.50, w*0.11, 1, tileCobblePathEdgeColor)
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

func drawTree(c Canvas, x, y, w, h float32) {
	c.FillRect(x+w*0.4, y+h*0.6, w*0.2, h*0.3, color.RGBA{0x6b, 0x4a, 0x2a, 0xff})
	c.FillCircle(x+w*0.5, y+h*0.4, w*0.3, color.RGBA{0x2d, 0x7a, 0x2a, 0xff})
	c.StrokeCircle(x+w*0.5, y+h*0.4, w*0.3, 1, color.RGBA{0x1d, 0x5a, 0x1a, 0xff})
}

func drawQuicksand(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0xdf, 0xc7, 0x94, 0xff})
	c.StrokeLine(x+w*0.1, y+h*0.2, x+w*0.4, y+h*0.2, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
	c.StrokeLine(x+w*0.6, y+h*0.4, x+w*0.9, y+h*0.4, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.7, y+h*0.7, 1, color.RGBA{0xb8, 0x9e, 0x6a, 0xff})
}

func drawMud(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0x6b, 0x4c, 0x35, 0xff})
	c.FillCircle(x+w*0.3, y+h*0.4, w*0.12, color.RGBA{0x4a, 0x32, 0x22, 0xff})
	c.FillCircle(x+w*0.7, y+h*0.6, w*0.15, color.RGBA{0x4a, 0x32, 0x22, 0xff})
}

func drawIce(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0xa0, 0xd8, 0xef, 0xff})
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
	c.FillRect(x, y, w, h, color.RGBA{0xd3, 0x54, 0x00, 0xff})

	tick := c.Tick()
	tx, ty := c.GridPos()

	phase1 := int(tx*37 + ty*17)
	t1 := (tick + phase1) % lavaBubble1Cycle
	if t1 >= lavaBubble1GrowStart && t1 < lavaBubble1PopStart {
		progress := float32(t1-lavaBubble1GrowStart) / float32(lavaBubble1PopStart-lavaBubble1GrowStart)
		c.FillCircle(x+w*0.3, y+h*0.6, progress*w*0.15, color.RGBA{0xe6, 0x7e, 0x22, 0xff})
	} else if t1 >= lavaBubble1PopStart && t1 < lavaBubble1PopStart+lavaBubble1PopDuration {
		c.StrokeCircle(x+w*0.3, y+h*0.6, w*0.18, 1, color.RGBA{0xe6, 0x7e, 0x22, 0xff})
	}

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
	c.FillRect(x, y, w, h, tileGrassColor)

	postW := w * 0.15
	postH := h * 0.6
	postX := x + (w-postW)*0.5
	postY := y + h*0.4
	c.FillRect(postX, postY, postW, postH, color.RGBA{0x6a, 0x3d, 0x1b, 0xff})

	plaqueW := w * 0.8
	plaqueH := h * 0.4
	plaqueX := x + w*0.1
	plaqueY := y + h*0.1
	c.FillRect(plaqueX, plaqueY, plaqueW, plaqueH, color.RGBA{0x8b, 0x5a, 0x2b, 0xff})
	c.StrokeRect(plaqueX, plaqueY, plaqueW, plaqueH, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})

	c.StrokeLine(plaqueX+w*0.1, plaqueY+h*0.15, plaqueX+plaqueW-w*0.1, plaqueY+h*0.15, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})
	c.StrokeLine(plaqueX+w*0.15, plaqueY+h*0.27, plaqueX+plaqueW-w*0.15, plaqueY+h*0.27, 1, color.RGBA{0x4a, 0x27, 0x0c, 0xff})
}

func drawSand(c Canvas, x, y, w, h float32) {
	c.FillRect(x, y, w, h, color.RGBA{0xe5, 0xd3, 0xa3, 0xff})
	c.FillRect(x+w*0.2, y+h*0.15, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.75, y+h*0.3, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.4, y+h*0.5, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.8, y+h*0.75, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.FillRect(x+w*0.15, y+h*0.8, 1, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.StrokeLine(x+w*0.15, y+h*0.35, x+w*0.45, y+h*0.35, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
	c.StrokeLine(x+w*0.55, y+h*0.65, x+w*0.85, y+h*0.65, 1, color.RGBA{0xc8, 0xb6, 0x86, 0xff})
}
