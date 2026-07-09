package ebitenplat

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/world"
)

func (r *Renderer) drawCharacter(worldObj *world.World, x, y, w, h float32, dir world.Dir, isPlayer bool, isBoss bool) {
	var baseCol color.RGBA
	var borderCol color.RGBA
	var eyeCol color.RGBA = color.RGBA{0xff, 0xff, 0xff, 0xff}
	var pupilCol color.RGBA = color.RGBA{0x00, 0x00, 0x00, 0xff}

	if isPlayer {
		baseCol = color.RGBA{0x40, 0x70, 0xff, 0xff} // vibrant blue
		borderCol = color.RGBA{0x1b, 0x2b, 0x5a, 0xff}
	} else {
		if isBoss {
			baseCol = color.RGBA{0x70, 0x10, 0x10, 0xff} // dark red
			borderCol = color.RGBA{0x2b, 0x05, 0x05, 0xff}
		} else {
			baseCol = color.RGBA{0xc0, 0x30, 0x30, 0xff} // red
			borderCol = color.RGBA{0x4a, 0x12, 0x12, 0xff}
		}
	}

	r.FillRect(x, y, w, h, baseCol)
	r.StrokeRect(x, y, w, h, 1, borderCol)

	switch dir {
	case world.DirDown:
		r.FillRect(x+2, y+4, 2, 2, eyeCol)
		r.FillRect(x+w-4, y+4, 2, 2, eyeCol)
		r.FillRect(x+2, y+5, 1, 1, pupilCol)
		r.FillRect(x+w-4, y+5, 1, 1, pupilCol)
	case world.DirUp:
		r.FillRect(x+1, y+1, w-2, 4, borderCol)
	case world.DirLeft:
		r.FillRect(x+2, y+4, 2, 2, eyeCol)
		r.FillRect(x+2, y+5, 1, 1, pupilCol)
	case world.DirRight:
		r.FillRect(x+w-4, y+4, 2, 2, eyeCol)
		r.FillRect(x+w-4, y+5, 1, 1, pupilCol)
	}

	if isPlayer && worldObj != nil && worldObj.ShieldLevel > 0 {
		var shieldFill, shieldBorder color.RGBA
		if worldObj.ShieldLevel >= 2 {
			// Mirror Shield colors
			shieldFill = color.RGBA{0x50, 0xd0, 0xff, 0xff}   // vibrant cyan
			shieldBorder = color.RGBA{0xff, 0xd7, 0x00, 0xff} // gold
		} else {
			// Basic Shield colors
			shieldFill = color.RGBA{0x8b, 0x5a, 0x2b, 0xff}   // brown wood
			shieldBorder = color.RGBA{0xa0, 0xa0, 0xa0, 0xff} // silver/grey
		}

		// Draw shield depending on direction
		switch dir {
		case world.DirDown:
			r.FillRect(x+w*0.5-4, y+h-2, 8, 3, shieldFill)
			r.StrokeRect(x+w*0.5-4, y+h-2, 8, 3, 1, shieldBorder)
		case world.DirUp:
			r.FillRect(x+w*0.5-4, y-1, 8, 3, shieldFill)
			r.StrokeRect(x+w*0.5-4, y-1, 8, 3, 1, shieldBorder)
		case world.DirLeft:
			r.FillRect(x-2, y+h*0.5-4, 3, 8, shieldFill)
			r.StrokeRect(x-2, y+h*0.5-4, 3, 8, 1, shieldBorder)
		case world.DirRight:
			r.FillRect(x+w-1, y+h*0.5-4, 3, 8, shieldFill)
			r.StrokeRect(x+w-1, y+h*0.5-4, 3, 8, 1, shieldBorder)
		}
	}
}

func (r *Renderer) drawSwordSwing(w *world.World, px, py, pw, ph float64, ox, oy float64) {
	if w.Player.Swing <= 0 {
		return
	}
	var baseAngle float64
	switch w.Player.Dir {
	case world.DirDown:
		baseAngle = math.Pi / 2
	case world.DirUp:
		baseAngle = -math.Pi / 2
	case world.DirLeft:
		baseAngle = math.Pi
	default:
		baseAngle = 0
	}

	dur := float64(w.Player.EffectiveSwingDuration())
	if dur <= 0 {
		dur = 8
	}
	t := 1.0 - float64(w.Player.Swing)/dur
	const sweep = 1.1
	angle := baseAngle - sweep + t*2.0*sweep

	screen := r.screen
	const numTrails = 4
	const step = 0.12
	for i := numTrails; i > 0; i-- {
		trailAngle := angle - float64(i)*step
		if trailAngle >= baseAngle-sweep {
			trailAlpha := float32(0.12) * float32(numTrails-i+1) / float32(numTrails)
			cos := float32(math.Cos(trailAngle))
			sin := float32(math.Sin(trailAngle))
			vector.StrokeLine(screen,
				float32(px-ox+pw*0.5)+4*cos, float32(py-oy+ph*0.5)+4*sin,
				float32(px-ox+pw*0.5)+17*cos, float32(py-oy+ph*0.5)+17*sin,
				1.5, color.RGBA{0xa0, 0xe0, 0xff, uint8(255 * trailAlpha)}, true)
		}
	}

	cx := float32(px - ox + pw*0.5)
	cy := float32(py - oy + ph*0.5)
	drawSwordAt(screen, cx, cy, angle, 1.0)
}

func (r *Renderer) drawTorchSwing(w *world.World, px, py, pw, ph float64, ox, oy float64) {
	if w.Player.TorchSwing <= 0 {
		return
	}
	var baseAngle float64
	switch w.Player.Dir {
	case world.DirDown:
		baseAngle = math.Pi / 2
	case world.DirUp:
		baseAngle = -math.Pi / 2
	case world.DirLeft:
		baseAngle = math.Pi
	default:
		baseAngle = 0
	}

	dur := float64(w.Player.EffectiveTorchSwingDuration())
	if dur <= 0 {
		dur = 8
	}
	t := 1.0 - float64(w.Player.TorchSwing)/dur
	const sweep = 1.1
	angle := baseAngle - sweep + t*2.0*sweep

	screen := r.screen
	const numTrails = 4
	const step = 0.12
	for i := numTrails; i > 0; i-- {
		trailAngle := angle - float64(i)*step
		if trailAngle >= baseAngle-sweep {
			trailAlpha := float32(0.12) * float32(numTrails-i+1) / float32(numTrails)
			cos := float32(math.Cos(trailAngle))
			sin := float32(math.Sin(trailAngle))
			vector.StrokeLine(screen,
				float32(px-ox+pw*0.5)+4*cos, float32(py-oy+ph*0.5)+4*sin,
				float32(px-ox+pw*0.5)+15*cos, float32(py-oy+ph*0.5)+15*sin,
				2.0, color.RGBA{0xff, 0x70, 0x00, uint8(255 * trailAlpha)}, true)
		}
	}

	cx := float32(px - ox + pw*0.5)
	cy := float32(py - oy + ph*0.5)
	drawTorchAt(screen, cx, cy, angle, 1.0, w.Tick)
}

func drawSwordAt(screen *ebiten.Image, cx, cy float32, angle float64, alpha float32) {
	cos := float32(math.Cos(angle))
	sin := float32(math.Sin(angle))

	perpCos := -sin
	perpSin := cos

	bladeCol := color.RGBA{0xd8, 0xdf, 0xe5, uint8(255 * alpha)}
	bladeEdgeCol := color.RGBA{0xff, 0xff, 0xff, uint8(255 * alpha)}
	guardCol := color.RGBA{0xd8, 0xa0, 0x20, uint8(255 * alpha)}
	handleCol := color.RGBA{0x5c, 0x3a, 0x21, uint8(255 * alpha)}
	pommelCol := color.RGBA{0x3e, 0x27, 0x16, uint8(255 * alpha)}

	const handleStart = 2.0
	const handleEnd = -2.0
	const guardDist = 3.0
	const bladeEnd = 16.0

	vector.StrokeLine(screen,
		cx+handleStart*cos, cy+handleStart*sin,
		cx+handleEnd*cos, cy+handleEnd*sin,
		1.5, handleCol, true)
	vector.FillCircle(screen, cx+handleEnd*cos, cy+handleEnd*sin, 1.2, pommelCol, true)

	const guardHalfWidth = 4.0
	vector.StrokeLine(screen,
		cx+guardDist*cos-guardHalfWidth*perpCos, cy+guardDist*sin-guardHalfWidth*perpSin,
		cx+guardDist*cos+guardHalfWidth*perpCos, cy+guardDist*sin+guardHalfWidth*perpSin,
		1.8, guardCol, true)

	vector.StrokeLine(screen,
		cx+guardDist*cos, cy+guardDist*sin,
		cx+bladeEnd*cos, cy+bladeEnd*sin,
		2.2, bladeCol, true)

	vector.StrokeLine(screen,
		cx+guardDist*cos+0.5*perpCos, cy+guardDist*sin+0.5*perpSin,
		cx+bladeEnd*cos, cy+bladeEnd*sin,
		1.0, bladeEdgeCol, true)
}

func drawTorchAt(screen *ebiten.Image, cx, cy float32, angle float64, alpha float32, tick int) {
	cos := float32(math.Cos(angle))
	sin := float32(math.Sin(angle))

	handleCol := color.RGBA{0x8b, 0x5a, 0x2b, uint8(255 * alpha)}
	tipCol := color.RGBA{0x4a, 0x4a, 0x4a, uint8(255 * alpha)}
	flameCol := color.RGBA{0xff, 0x50, 0x00, uint8(230 * alpha)}
	flameCoreCol := color.RGBA{0xff, 0xd0, 0x00, uint8(255 * alpha)}

	const handleStart = 2.0
	const handleEnd = -2.0
	const tipStart = 2.0
	const tipEnd = 5.0
	const flameEnd = 13.0

	vector.StrokeLine(screen,
		cx+handleStart*cos, cy+handleStart*sin,
		cx+handleEnd*cos, cy+handleEnd*sin,
		1.5, handleCol, true)

	vector.StrokeLine(screen,
		cx+tipStart*cos, cy+tipStart*sin,
		cx+tipEnd*cos, cy+tipEnd*sin,
		2.2, tipCol, true)

	flicker := float32(math.Sin(float64(tick)*0.45)) * 1.5
	actualFlameEnd := flameEnd + flicker
	vector.StrokeLine(screen,
		cx+tipEnd*cos, cy+tipEnd*sin,
		cx+actualFlameEnd*cos, cy+actualFlameEnd*sin,
		3.0, flameCol, true)

	vector.StrokeLine(screen,
		cx+tipEnd*cos, cy+tipEnd*sin,
		cx+(actualFlameEnd-3.0)*cos, cy+(actualFlameEnd-3.0)*sin,
		1.5, flameCoreCol, true)
}

func (r *Renderer) drawPickup(p world.Pickup, ox, oy float64) {
	px := float32(p.X - ox)
	py := float32(p.Y - oy)
	r.DrawPickupScreen(p.Kind, px, py, float32(p.W), float32(p.H))
}

func (r *Renderer) drawClosedChest(p world.Pickup, ox, oy float64) {
	px := float32(p.X - ox)
	py := float32(p.Y - oy)
	w := float32(p.W)
	h := float32(p.H)

	r.FillRect(px, py, w, h, color.RGBA{0x5c, 0x3a, 0x21, 0xff})
	r.StrokeRect(px, py, w, h, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})
	r.FillRect(px+2, py, 2, h, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+8, py, 2, h, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+5, py+4, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+5, py+6, 2, 2, color.RGBA{0x2b, 0x2b, 0x2b, 0xff})
}

func (r *Renderer) drawOpenedChest(p world.Pickup, ox, oy float64) {
	px := float32(p.X - ox)
	py := float32(p.Y - oy)
	w := float32(p.W)

	r.FillRect(px, py+5, w, 7, color.RGBA{0x5c, 0x3a, 0x21, 0xff})
	r.StrokeRect(px, py+5, w, 7, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})
	r.FillRect(px+2, py+6, w-4, 4, color.RGBA{0x10, 0x10, 0x10, 0xff})
	r.FillRect(px, py+1, w, 4, color.RGBA{0x5c, 0x3a, 0x21, 0xff})
	r.StrokeRect(px, py+1, w, 4, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})
	r.FillRect(px+2, py+1, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+8, py+1, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+5, py+3, 2, 3, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
}

func (r *Renderer) pickupFallback(px, py float64) {
	vector.FillRect(r.screen, float32(px), float32(py), pickupFallbackPx, pickupFallbackPx,
		color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
}

func (r *Renderer) drawShrine(sh world.Shrine, ox, oy float64, tick int) {
	sx := float32(sh.Rect.X - ox)
	sy := float32(sh.Rect.Y - oy)
	sw := float32(sh.Rect.W)
	shh := float32(sh.Rect.H)

	r.FillRect(sx+1, sy+shh-3, sw-2, 3, color.RGBA{0x55, 0x55, 0x5a, 0xff})
	r.FillRect(sx+2, sy+shh-6, sw-4, 3, color.RGBA{0x70, 0x70, 0x75, 0xff})
	r.FillRect(sx+5, sy+shh-11, sw-10, 5, color.RGBA{0x80, 0x80, 0x85, 0xff})
	r.FillRect(sx+5, sy+shh-11, 1, 5, color.RGBA{0xa0, 0xa0, 0xa5, 0xff})
	r.FillRect(sx+sw-6, sy+shh-11, 1, 5, color.RGBA{0x55, 0x55, 0x5a, 0xff})
	r.FillRect(sx+2, sy+shh-13, sw-4, 2, color.RGBA{0x90, 0x90, 0x98, 0xff})
	r.FillRect(sx+2, sy+shh-13, sw-4, 1, color.RGBA{0xc0, 0xc0, 0xc8, 0xff})

	if !sh.Active {
		return
	}

	offsetY := float32(math.Sin(float64(tick)*0.08) * 1.5)
	cy := sy + 2 + offsetY
	cx := sx + sw*0.5

	vector.FillCircle(r.screen, cx, cy+1, 5, color.RGBA{0x00, 0xe0, 0xff, 0x33}, true)
	vector.FillCircle(r.screen, cx, cy+1, 3, color.RGBA{0x00, 0xe0, 0xff, 0x55}, true)

	var path vector.Path
	path.MoveTo(cx, cy-4)
	path.LineTo(cx+3, cy+1)
	path.LineTo(cx, cy+6)
	path.LineTo(cx-3, cy+1)
	path.Close()
	drawPath(r.screen, &path, color.RGBA{0x00, 0xb0, 0xe0, 0xff}, true, 0)
	drawPath(r.screen, &path, color.RGBA{0x00, 0xe0, 0xff, 0xff}, false, 1.0)

	var corePath vector.Path
	corePath.MoveTo(cx, cy-2)
	corePath.LineTo(cx+1.5, cy+1)
	corePath.LineTo(cx, cy+4)
	corePath.LineTo(cx-1.5, cy+1)
	corePath.Close()
	drawPath(r.screen, &corePath, color.RGBA{0xe0, 0xfb, 0xff, 0xff}, true, 0)

	sparkOffset1 := float32(math.Sin(float64(tick)*0.05) * 4)
	sparkOffset2 := float32(math.Cos(float64(tick)*0.06) * 4)
	r.FillRect(cx-5+sparkOffset1, cy-3+sparkOffset2, 1, 1, color.RGBA{0x00, 0xff, 0xff, 0xbb})
	r.FillRect(cx+4-sparkOffset2, cy-1-sparkOffset1, 1, 1, color.RGBA{0x00, 0xff, 0xff, 0x99})
}

func (r *Renderer) drawBomb(b world.ActiveBomb, ox, oy float64) {
	sx := float32(b.X - ox)
	sy := float32(b.Y - oy)

	var cycleLen int
	if b.Timer > 60 {
		cycleLen = 20
	} else if b.Timer > 30 {
		cycleLen = 10
	} else if b.Timer > 10 {
		cycleLen = 4
	} else {
		cycleLen = 2
	}
	isFlash := (b.Timer/(cycleLen/2))%2 == 0

	bodyColor := color.RGBA{20, 20, 20, 255}
	if isFlash {
		bodyColor = color.RGBA{240, 50, 50, 255}
	}

	r.FillRect(sx-5, sy-4, 10, 8, bodyColor)
	r.FillRect(sx-4, sy-5, 8, 10, bodyColor)
	r.FillRect(sx-2, sy-7, 4, 2, color.RGBA{60, 60, 60, 255})
	r.StrokeLine(sx, sy-7, sx+3, sy-9, 1, color.RGBA{130, 110, 90, 255})
}

func (r *Renderer) drawFlame(f world.ActiveFlame, ox, oy float64, tick int) {
	sx := float32(f.X - ox)
	sy := float32(f.Y - oy)
	flicker := (tick / 4) % 3

	r.FillRect(sx-6, sy-3+float32(flicker), 12, 8-float32(flicker), color.RGBA{230, 50, 20, 220})
	r.FillRect(sx-4, sy-7+float32(flicker), 8, 12-float32(flicker), color.RGBA{230, 50, 20, 220})
	r.FillRect(sx-4, sy-1+float32(flicker), 8, 6-float32(flicker), color.RGBA{255, 120, 20, 240})
	r.FillRect(sx-2, sy-5+float32(flicker), 4, 10-float32(flicker), color.RGBA{255, 120, 20, 240})
	r.FillRect(sx-2, sy+1, 4, 3, color.RGBA{255, 220, 40, 255})
	r.FillRect(sx-1, sy-2+float32(flicker), 2, 6-float32(flicker), color.RGBA{255, 220, 40, 255})
}

func fillTriangle(dst *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, clr color.Color) {
	var path vector.Path
	path.MoveTo(x1, y1)
	path.LineTo(x2, y2)
	path.LineTo(x3, y3)
	path.Close()
	var op vector.DrawPathOptions
	op.ColorScale.ScaleWithColor(clr)
	op.AntiAlias = true
	vector.FillPath(dst, &path, nil, &op)
}

func drawPath(dst *ebiten.Image, path *vector.Path, clr color.Color, fill bool, strokeWidth float32) {
	var op vector.DrawPathOptions
	op.ColorScale.ScaleWithColor(clr)
	op.AntiAlias = true
	if fill {
		vector.FillPath(dst, path, nil, &op)
	} else {
		var sop vector.StrokeOptions
		sop.Width = strokeWidth
		vector.StrokePath(dst, path, &sop, &op)
	}
}
