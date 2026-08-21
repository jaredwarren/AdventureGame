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

	tx, ty := c.GridPos()
	// Deterministic spatial hash for subtle organic meadow variation
	hash := uint32(tx*374761393+ty*668265263) ^ 0x5bf03635
	variant := hash % 100

	bladeColor := color.RGBA{0x3e, 0x6e, 0x3e, 0xff}

	switch {
	case variant < 40: // 40%: Standard double tuft
		c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.3, y+h*0.4, 1, bladeColor)
		c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.45, y+h*0.5, 1, bladeColor)
		c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.7, y+h*0.2, 1, bladeColor)
		c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.3, 1, bladeColor)

	case variant < 70: // 30%: Single central high blade tuft
		c.StrokeLine(x+w*0.5, y+h*0.75, x+w*0.5, y+h*0.35, 1, bladeColor)
		c.StrokeLine(x+w*0.5, y+h*0.75, x+w*0.65, y+h*0.45, 1, bladeColor)
		c.StrokeLine(x+w*0.5, y+h*0.75, x+w*0.35, y+h*0.5, 1, bladeColor)

	case variant < 88: // 18%: Subtle sparse lawn with tiny blade specks
		c.StrokeLine(x+w*0.25, y+h*0.4, x+w*0.25, y+h*0.25, 1, bladeColor)
		c.StrokeLine(x+w*0.75, y+h*0.8, x+w*0.75, y+h*0.65, 1, bladeColor)

	default: // 12%: Tiny wildflower blossom
		c.StrokeLine(x+w*0.4, y+h*0.7, x+w*0.4, y+h*0.4, 1, bladeColor)
		c.StrokeLine(x+w*0.4, y+h*0.7, x+w*0.55, y+h*0.5, 1, bladeColor)
		if variant%2 == 0 {
			c.FillCircle(x+w*0.4, y+h*0.35, 1.2, color.RGBA{0xf1, 0xc4, 0x0f, 0xff}) // Sunny yellow petal
		} else {
			c.FillCircle(x+w*0.4, y+h*0.35, 1.2, color.RGBA{0xf8, 0xf9, 0xfa, 0xff}) // White daisy petal
		}
	}
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

	// Deterministic spatial hash for natural lake & river variation
	hash := uint32(tx*374761393+ty*668265263) ^ 0x5bf03635
	variant := hash % 100

	// Harmonic wave animations (periods: 180 and 240 ticks, both divide into 1440)
	angle1 := float64(tick+tx*17+ty*37) * 2 * math.Pi / 180.0
	offset1 := float32(math.Sin(angle1)) * w * 0.08

	angle2 := float64(tick+tx*53+ty*29) * 2 * math.Pi / 240.0
	offset2 := float32(math.Sin(angle2)) * w * 0.08

	waveColor := color.RGBA{0x4a, 0x6a, 0xaa, 0xff}
	foamColor := color.RGBA{0x6a, 0x8e, 0xcc, 0xff}
	sparkleColor := color.RGBA{0xd8, 0xec, 0xff, 0xff}
	lilyColor := color.RGBA{0x3a, 0x7d, 0x3e, 0xff}

	switch {
	case variant < 40: // 40%: Standard dual wave lines
		c.StrokeLine(x+w*0.2+offset1, y+h*0.3, x+w*0.4+offset1, y+h*0.3, 1, waveColor)
		c.StrokeLine(x+w*0.5+offset2, y+h*0.7, x+w*0.8+offset2, y+h*0.7, 1, waveColor)

	case variant < 70: // 30%: Staggered 3-ripple current
		c.StrokeLine(x+w*0.15+offset1, y+h*0.25, x+w*0.35+offset1, y+h*0.25, 1, waveColor)
		c.StrokeLine(x+w*0.45+offset2, y+h*0.55, x+w*0.75+offset2, y+h*0.55, 1, foamColor)
		c.StrokeLine(x+w*0.25+offset1, y+h*0.80, x+w*0.55+offset1, y+h*0.80, 1, waveColor)

	case variant < 85: // 15%: Sunlight glint sparkle
		c.StrokeLine(x+w*0.2+offset1, y+h*0.35, x+w*0.4+offset1, y+h*0.35, 1, waveColor)
		c.StrokeLine(x+w*0.5+offset2, y+h*0.75, x+w*0.8+offset2, y+h*0.75, 1, waveColor)
		c.FillCircle(x+w*0.6+offset1, y+h*0.35, 0.9, sparkleColor)

	default: // 15%: Floating Lily Pad
		c.StrokeLine(x+w*0.15+offset1, y+h*0.3, x+w*0.4+offset1, y+h*0.3, 1, waveColor)
		lilyX := x + w*0.65 + offset2*0.5
		lilyY := y + h*0.65
		c.FillCircle(lilyX, lilyY, w*0.13, lilyColor)
		c.FillCircle(lilyX+w*0.04, lilyY-h*0.04, w*0.05, color.RGBA{0x4e, 0x9a, 0x52, 0xff})
	}
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
	tx, ty := c.GridPos()
	// Deterministic spatial hash for natural forest canopy variation
	hash := uint32(tx*374761393+ty*668265263) ^ 0x5bf03635
	variant := hash % 100

	trunkColor := color.RGBA{0x6b, 0x4a, 0x2a, 0xff}
	foliageBase := color.RGBA{0x2d, 0x7a, 0x2a, 0xff}
	foliageOutline := color.RGBA{0x1d, 0x5a, 0x1a, 0xff}
	foliageHighlight := color.RGBA{0x3e, 0x96, 0x3b, 0xff}

	switch {
	case variant < 40: // 40%: Standard lush round oak
		c.FillRect(x+w*0.42, y+h*0.6, w*0.16, h*0.32, trunkColor)
		c.FillCircle(x+w*0.5, y+h*0.38, w*0.32, foliageBase)
		c.StrokeCircle(x+w*0.5, y+h*0.38, w*0.32, 1, foliageOutline)
		c.FillCircle(x+w*0.42, y+h*0.30, w*0.14, foliageHighlight)

	case variant < 70: // 30%: Multi-lobed bushy tree
		c.FillRect(x+w*0.4, y+h*0.62, w*0.2, h*0.3, trunkColor)
		c.FillCircle(x+w*0.38, y+h*0.42, w*0.24, foliageBase)
		c.FillCircle(x+w*0.62, y+h*0.42, w*0.24, foliageBase)
		c.FillCircle(x+w*0.5, y+h*0.32, w*0.26, foliageBase)
		c.StrokeCircle(x+w*0.5, y+h*0.38, w*0.34, 1, foliageOutline)
		c.FillCircle(x+w*0.48, y+h*0.26, w*0.12, foliageHighlight)
		c.FillCircle(x+w*0.62, y+h*0.38, w*0.10, foliageHighlight)

	case variant < 85: // 15%: Tall slender woodland tree with root base
		c.FillRect(x+w*0.44, y+h*0.58, w*0.12, h*0.34, trunkColor)
		c.StrokeLine(x+w*0.38, y+h*0.9, x+w*0.62, y+h*0.9, 1, trunkColor)
		c.FillCircle(x+w*0.5, y+h*0.34, w*0.28, foliageBase)
		c.StrokeCircle(x+w*0.5, y+h*0.34, w*0.28, 1, foliageOutline)
		c.FillCircle(x+w*0.44, y+h*0.28, w*0.12, foliageHighlight)

	default: // 15%: Apple / Berry Blossom Tree
		c.FillRect(x+w*0.42, y+h*0.6, w*0.16, h*0.32, trunkColor)
		c.FillCircle(x+w*0.5, y+h*0.38, w*0.32, foliageBase)
		c.StrokeCircle(x+w*0.5, y+h*0.38, w*0.32, 1, foliageOutline)
		c.FillCircle(x+w*0.42, y+h*0.30, w*0.14, foliageHighlight)
		fruitColor := color.RGBA{0xd6, 0x30, 0x31, 0xff}
		c.FillCircle(x+w*0.36, y+h*0.38, 1.1, fruitColor)
		c.FillCircle(x+w*0.62, y+h*0.34, 1.1, fruitColor)
		c.FillCircle(x+w*0.50, y+h*0.48, 1.1, fruitColor)
	}
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

func drawSnow(c Canvas, x, y, w, h float32) {
	snowBase := color.RGBA{0xf0, 0xf8, 0xff, 0xff}
	shadow := color.RGBA{0xb8, 0xd0, 0xe8, 0xff}
	highlight := color.RGBA{0xd4, 0xe4, 0xf4, 0xff}

	c.FillRect(x, y, w, h, snowBase)
	// Powder sparkles and drift ripple
	c.FillCircle(x+w*0.25, y+h*0.25, 1.2, highlight)
	c.FillCircle(x+w*0.75, y+h*0.35, 1.2, highlight)
	c.FillCircle(x+w*0.40, y+h*0.70, 1.2, highlight)
	c.StrokeLine(x+w*0.15, y+h*0.5, x+w*0.45, y+h*0.5, 1, shadow)
	c.StrokeLine(x+w*0.55, y+h*0.8, x+w*0.85, y+h*0.8, 1, shadow)
}

func drawDarkGrass(c Canvas, x, y, w, h float32) {
	darkGrassColor := color.RGBA{0x35, 0x5a, 0x35, 0xff}
	mossColor := color.RGBA{0x22, 0x3d, 0x22, 0xff}
	highlight := color.RGBA{0x45, 0x6e, 0x45, 0xff}

	c.FillRect(x, y, w, h, darkGrassColor)
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.3, y+h*0.4, 1, mossColor)
	c.StrokeLine(x+w*0.3, y+h*0.7, x+w*0.45, y+h*0.5, 1, mossColor)
	c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.7, y+h*0.2, 1, mossColor)
	c.StrokeLine(x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.3, 1, mossColor)
	c.FillCircle(x+w*0.5, y+h*0.3, 1.2, highlight)
	c.FillCircle(x+w*0.2, y+h*0.8, 1.2, highlight)
}

func drawDeepWater(c Canvas, x, y, w, h float32) {
	deepWaterColor := color.RGBA{0x18, 0x2a, 0x5a, 0xff}
	c.FillRect(x, y, w, h, deepWaterColor)

	tick := c.Tick()
	tx, ty := c.GridPos()

	// Harmonic wave periods: 240 and 360 (both divide evenly into 1440)
	angle1 := float64(tick+tx*17+ty*37) * 2 * math.Pi / 240.0
	offset1 := float32(math.Sin(angle1)) * w * 0.06

	angle2 := float64(tick+tx*53+ty*29) * 2 * math.Pi / 360.0
	offset2 := float32(math.Sin(angle2)) * w * 0.06

	c.StrokeLine(x+w*0.15+offset1, y+h*0.35, x+w*0.45+offset1, y+h*0.35, 1.2, color.RGBA{0x2e, 0x4e, 0x8e, 0xff})
	c.StrokeLine(x+w*0.50+offset2, y+h*0.75, x+w*0.85+offset2, y+h*0.75, 1.2, color.RGBA{0x2e, 0x4e, 0x8e, 0xff})
}

func drawSwampWater(c Canvas, x, y, w, h float32) {
	swampColor := color.RGBA{0x32, 0x44, 0x2e, 0xff}
	c.FillRect(x, y, w, h, swampColor)

	tick := c.Tick()
	tx, ty := c.GridPos()

	// Harmonic bubble cycle (360 ticks, divides evenly into 1440)
	phase := int(tx*41 + ty*19)
	t := (tick + phase) % 360
	if t >= 240 && t < 310 {
		progress := float32(t-240) / 70.0
		c.FillCircle(x+w*0.4, y+h*0.5, progress*w*0.12, color.RGBA{0x55, 0x75, 0x40, 0xff})
	} else if t >= 310 && t < 325 {
		c.StrokeCircle(x+w*0.4, y+h*0.5, w*0.14, 1, color.RGBA{0x6e, 0x94, 0x50, 0xff})
	}

	c.StrokeLine(x+w*0.6, y+h*0.3, x+w*0.8, y+h*0.3, 1, color.RGBA{0x48, 0x62, 0x38, 0xff})
	c.StrokeLine(x+w*0.2, y+h*0.8, x+w*0.5, y+h*0.8, 1, color.RGBA{0x48, 0x62, 0x38, 0xff})
}
