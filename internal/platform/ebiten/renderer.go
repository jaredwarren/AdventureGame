// renderer.go — Ebiten-backed services.Renderer implementation.
//
// This type is the ONLY place in the repo allowed to hold a *ebiten.Image or
// call ebiten/vector/ebitenutil directly on behalf of scene-layer draws.
// Scenes import services.Renderer (an interface) and never see Ebiten types.
//
// Frame lifetime:
//
//	app.Draw(screen):
//	  r.BeginFrame(screen)    // captures screen + camera offset
//	  manager.Draw(ctx)       // scenes call r.DrawWorld / DrawText / ...
//	  r.EndFrame()            // releases screen ref so a stray post-frame
//	                          // draw nil-derefs deterministically
package ebitenplat

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

// useTileSprites enables PNG tile rendering (overworld-spritesheet.png +
// tile_atlas.json). When false, tiles use TileSwatchColor fills only.
const useTileSprites = false

// pickupFallbackPx is the draw size used when the atlas is unavailable; it
// matches the atlas's authored default (see defaultPickupDst).
const pickupFallbackPx = 12

const nightShaderCode = `
//kage:unit pixels
package main

var LightSource vec2
var LightRadius float
var PersonalRadius float
var AmbientColor vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	pixelPos := position.xy - imageDstOrigin()
	toPixel := pixelPos - LightSource
	dist := length(toPixel)

	// Personal ambient circle surrounding the player
	personalIntensity := 0.0
	if dist < PersonalRadius {
		personalIntensity = 1.0 - (dist / max(PersonalRadius, 0.001))
		personalIntensity = personalIntensity * personalIntensity * (3.0 - 2.0*personalIntensity)
	}

	// Flashlight/torch circle
	torchIntensity := 0.0
	if dist < LightRadius {
		torchIntensity = 1.0 - (dist / max(LightRadius, 0.001))
		torchIntensity = torchIntensity * torchIntensity * (3.0 - 2.0*torchIntensity)
	}

	totalLight := max(personalIntensity, torchIntensity)
	totalLight = clamp(totalLight, 0.0, 1.0)

	// Return ambient color tinted by (1.0 - totalLight)
	return AmbientColor * (1.0 - totalLight)
}
`

var nightShader *ebiten.Shader

func init() {
	var err error
	nightShader, err = ebiten.NewShader([]byte(nightShaderCode))
	if err != nil {
		panic("failed to compile night shader: " + err.Error())
	}
}

// Renderer is the Ebiten-backed services.Renderer. Construct via
// NewRenderer; Begin/EndFrame each frame from App.
type Renderer struct {
	camera *render.Camera
	assets services.AssetCache

	// per-frame state set by BeginFrame.
	screen  *ebiten.Image
	camOffX float64 // camera top-left + shake dx (subtract from world x)
	camOffY float64
}

// NewRenderer builds the renderer. The Camera is owned by the renderer so
// scenes that need to tweak shake or reduce-motion do so through Camera().
func NewRenderer(cam *render.Camera, assets services.AssetCache) *Renderer {
	return &Renderer{camera: cam, assets: assets}
}

// BeginFrame snapshots the camera + shake offset for the current frame. Call
// once per Draw from App; subsequent world-space draws read the cached
// offset instead of re-rolling shake on every tile.
func (r *Renderer) BeginFrame(screen *ebiten.Image) {
	r.screen = screen
	sx, sy := r.camera.Offset()
	r.camOffX = r.camera.X - sx
	r.camOffY = r.camera.Y - sy
}

// EndFrame releases the screen reference so accidental post-frame draws
// nil-deref deterministically instead of racing with the next frame.
func (r *Renderer) EndFrame() { r.screen = nil }

// Camera returns the active render camera. Scenes manipulate it directly.
func (r *Renderer) Camera() *render.Camera { return r.camera }

// DrawText renders one line of debug-style text at (x, y) screen pixels.
func (r *Renderer) DrawText(x, y int, text string) {
	ebitenutil.DebugPrintAt(r.screen, text, x, y)
}

// FillRect fills a screen-space rectangle.
func (r *Renderer) FillRect(x, y, w, h float32, c color.RGBA) {
	vector.FillRect(r.screen, x, y, w, h, c, false)
}

// StrokeRect outlines a screen-space rectangle with the given line width.
func (r *Renderer) StrokeRect(x, y, w, h, lw float32, c color.RGBA) {
	vector.StrokeRect(r.screen, x, y, w, h, lw, c, false)
}

// StrokeLine draws a line segment in screen space with the given width.
func (r *Renderer) StrokeLine(x1, y1, x2, y2, lw float32, c color.RGBA) {
	vector.StrokeLine(r.screen, x1, y1, x2, y2, lw, c, false)
}

// TileSwatchColor returns the flat fallback color a GID renders as.
// The table of colors lives on world.TileDef so new tile types register
// their palette in one place (see internal/world/tiledef.go).
func (r *Renderer) TileSwatchColor(gid int) color.RGBA {
	return world.TileDefOf(gid).SwatchColor
}

// DrawTileScreen draws one tile for UI (editor palette). Scales atlas art
// to (dw,dh); falls back to TileSwatchColor when sprites are off or the
// frame is missing. GIDEmpty uses swatch so the palette cell is visible.
func (r *Renderer) DrawTileScreen(gid int, x, y, dw, dh float32) {
	if r.screen == nil || dw < 1 || dh < 1 {
		return
	}
	if !useTileSprites || gid == world.GIDEmpty {
		r.drawVectorTile(r.screen, gid, x, y, dw, dh)
		return
	}
	atlas, err := r.assets.Atlas(services.AtlasTile)
	if err != nil || atlas == nil {
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
		return
	}
	if gid < 0 || gid >= atlas.Count() {
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
		return
	}
	fr := atlas.Frame(gid)
	if fr.Skip || fr.Image == nil {
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
		return
	}
	src := NativeImage(fr.Image)
	if src == nil {
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
		return
	}
	b := src.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(float64(dw)/sw, float64(dh)/sh)
	op.GeoM.Translate(float64(x), float64(y))
	r.screen.DrawImage(src, op)
}

// DrawWorld renders the tilemap + entities in camera space.
func (r *Renderer) DrawWorld(w *world.World) {
	if w == nil {
		return
	}
	screen := r.screen
	ox, oy := r.camOffX, r.camOffY
	vp := screen.Bounds()

	for ty := 0; ty < w.MapH; ty++ {
		for tx := 0; tx < w.MapW; tx++ {
			idx := ty*w.MapW + tx
			gid := w.Tiles[idx]
			if w.DestroyedTiles[idx] {
				if def := world.TileDefOf(gid); def.Destroyable() {
					gid = def.ResolvedDestroyedGID()
				}
			}
			x := float32(float64(tx*world.TileSize) - ox)
			y := float32(float64(ty*world.TileSize) - oy)
			if float64(x)+world.TileSize < 0 || float64(y)+world.TileSize < 0 ||
				float64(x) > float64(vp.Dx()) || float64(y) > float64(vp.Dy()) {
				continue
			}
			if useTileSprites {
				r.drawTile(gid, x, y)
			} else {
				r.drawVectorTile(screen, gid, x, y, world.TileSize, world.TileSize)
			}
		}
	}

	for _, p := range w.Pickups {
		if p.Gone {
			continue
		}
		if w.IsEditor || p.PersistentSaveKey == "" {
			r.drawPickup(p, ox, oy)
		} else {
			if p.Opened {
				r.drawOpenedChest(p, ox, oy)
			} else {
				r.drawClosedChest(p, ox, oy)
			}
		}
	}

	for _, e := range w.Enemies {
		if e.HP <= 0 {
			continue
		}
		r.drawCharacter(float32(e.X-ox), float32(e.Y-oy), 14, 12, e.Dir, false, e.IsBoss)
	}

	pr := w.PlayerRect()
	r.drawCharacter(float32(pr.X-ox), float32(pr.Y-oy), float32(pr.W), float32(pr.H), w.Player.Dir, true, false)

	// Draw sword swing animation and trail if swinging
	if w.Player.Swing > 0 {
		var baseAngle float64
		switch w.Player.Dir {
		case world.DirDown:
			baseAngle = math.Pi / 2
		case world.DirUp:
			baseAngle = -math.Pi / 2
		case world.DirLeft:
			baseAngle = math.Pi
		default: // DirRight
			baseAngle = 0
		}

		t := float64(8-w.Player.Swing) / 7.0
		const sweep = 1.1
		angle := baseAngle - sweep + t*2.0*sweep

		// Draw motion blur trail
		const numTrails = 4
		const step = 0.12
		for i := numTrails; i > 0; i-- {
			trailAngle := angle - float64(i)*step
			if trailAngle >= baseAngle-sweep {
				trailAlpha := float32(0.12) * float32(numTrails-i+1) / float32(numTrails)
				cos := float32(math.Cos(trailAngle))
				sin := float32(math.Sin(trailAngle))
				vector.StrokeLine(screen,
					float32(pr.X-ox+pr.W*0.5)+4*cos, float32(pr.Y-oy+pr.H*0.5)+4*sin,
					float32(pr.X-ox+pr.W*0.5)+17*cos, float32(pr.Y-oy+pr.H*0.5)+17*sin,
					1.5, color.RGBA{0xa0, 0xe0, 0xff, uint8(255 * trailAlpha)}, true)
			}
		}

		// Draw main sword
		cx := float32(pr.X - ox + pr.W*0.5)
		cy := float32(pr.Y - oy + pr.H*0.5)
		drawSwordAt(screen, cx, cy, angle, 1.0)
	}

	// Draw torch swing animation and trail if swinging
	if w.Player.TorchSwing > 0 {
		var baseAngle float64
		switch w.Player.Dir {
		case world.DirDown:
			baseAngle = math.Pi / 2
		case world.DirUp:
			baseAngle = -math.Pi / 2
		case world.DirLeft:
			baseAngle = math.Pi
		default: // DirRight
			baseAngle = 0
		}

		t := float64(8-w.Player.TorchSwing) / 7.0
		const sweep = 1.1
		angle := baseAngle - sweep + t*2.0*sweep

		// Draw flaming motion blur trail
		const numTrails = 4
		const step = 0.12
		for i := numTrails; i > 0; i-- {
			trailAngle := angle - float64(i)*step
			if trailAngle >= baseAngle-sweep {
				trailAlpha := float32(0.12) * float32(numTrails-i+1) / float32(numTrails)
				cos := float32(math.Cos(trailAngle))
				sin := float32(math.Sin(trailAngle))
				vector.StrokeLine(screen,
					float32(pr.X-ox+pr.W*0.5)+4*cos, float32(pr.Y-oy+pr.H*0.5)+4*sin,
					float32(pr.X-ox+pr.W*0.5)+15*cos, float32(pr.Y-oy+pr.H*0.5)+15*sin,
					2.0, color.RGBA{0xff, 0x70, 0x00, uint8(255 * trailAlpha)}, true)
			}
		}

		// Draw main torch
		cx := float32(pr.X - ox + pr.W*0.5)
		cy := float32(pr.Y - oy + pr.H*0.5)
		drawTorchAt(screen, cx, cy, angle, 1.0, w.Tick)
	}

	if w.IsEditor {
		for _, d := range w.Doors {
			rd := d.Rect
			vector.FillRect(screen,
				float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H),
				color.RGBA{0xe0, 0x90, 0x18, 0x55}, false)
			vector.StrokeRect(screen,
				float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H), 2,
				color.RGBA{0xff, 0xc8, 0x40, 0xff}, false)
		}
	}

	for _, sh := range w.Shrines {
		r.drawShrine(sh, ox, oy, w.Tick)
	}

	// Apply Night Overlay Shader
	if (w.HasAmbientLightOverride && w.AmbientLightOverride < 1.0) || w.TimeOfDay < 1200 || w.TimeOfDay >= 9600 {
		mult := w.LightMultiplier()
		alpha := float32((1.0 - mult) * 1.175)
		if alpha > 0.94 {
			alpha = 0.94
		}
		if alpha < 0 {
			alpha = 0
		}

		if alpha > 0.01 {
			pr := w.PlayerRect()
			px := float32(pr.X - ox + pr.W*0.5)
			py := float32(pr.Y - oy + pr.H*0.5)

			lightR := float32(0.0)
			if w.HasTorch {
				lightR = 85.0
			}

			op := &ebiten.DrawRectShaderOptions{}
			op.Uniforms = map[string]any{
				"LightSource":    []float32{px, py},
				"PersonalRadius": float32(35.0),
				"LightRadius":    lightR,
				"AmbientColor":   []float32{0.03 * alpha, 0.03 * alpha, 0.15 * alpha, alpha},
			}
			r.screen.DrawRectShader(vp.Dx(), vp.Dy(), nightShader, op)
		}
	}
}

// drawTile renders one ground tile using the tile atlas. Falls back to a
// solid TileSwatchColor if the atlas is unavailable. (wx, wy) are screen-
// space — callers have already subtracted the camera offset.
func (r *Renderer) drawTile(gid int, wx, wy float32) {
	if gid == world.GIDEmpty {
		return
	}
	atlas, err := r.assets.Atlas(services.AtlasTile)
	if err != nil || atlas == nil {
		r.fallbackTile(gid, wx, wy)
		return
	}
	if gid < 0 || gid >= atlas.Count() {
		r.fallbackTile(gid, wx, wy)
		return
	}
	fr := atlas.Frame(gid)
	if fr.Skip || fr.Image == nil {
		r.fallbackTile(gid, wx, wy)
		return
	}
	src := NativeImage(fr.Image)
	if src == nil {
		r.fallbackTile(gid, wx, wy)
		return
	}
	b := src.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		r.fallbackTile(gid, wx, wy)
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(float64(world.TileSize)/sw, float64(world.TileSize)/sh)
	op.GeoM.Translate(float64(wx), float64(wy))
	r.screen.DrawImage(src, op)
}

func (r *Renderer) fallbackTile(gid int, wx, wy float32) {
	c := r.TileSwatchColor(gid)
	vector.FillRect(r.screen, wx, wy, world.TileSize, world.TileSize, c, false)
}

// DrawPickupScreen draws one pickup kind in screen space at (px, py) with size (w, h).
func (r *Renderer) DrawPickupScreen(kind *world.PickupKind, px, py, w, h float32) {
	if r.screen == nil {
		return
	}
	if draw, ok := pickupDrawers[kind]; ok {
		draw(r.screen, px, py, w, h)
		return
	}
	r.pickupFallback(float64(px), float64(py))
}

// drawPickup renders one pickup entity using the pickup atlas. (ox,oy) are
// the camera offsets; (p.X,p.Y) is world space.
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

	// Main wood chest body: 12x12 brown box
	r.FillRect(px, py, w, h, color.RGBA{0x5c, 0x3a, 0x21, 0xff})

	// Outline/border in darker brown for definition
	r.StrokeRect(px, py, w, h, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})

	// Gold iron bands on the sides
	r.FillRect(px+2, py, 2, h, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+8, py, 2, h, color.RGBA{0xd8, 0xa0, 0x20, 0xff})

	// Gold latch/lock in the center
	r.FillRect(px+5, py+4, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+5, py+6, 2, 2, color.RGBA{0x2b, 0x2b, 0x2b, 0xff}) // Dark keyhole
}

func (r *Renderer) drawOpenedChest(p world.Pickup, ox, oy float64) {
	px := float32(p.X - ox)
	py := float32(p.Y - oy)
	w := float32(p.W)

	// Wood box base: 12x7 brown box at the bottom
	r.FillRect(px, py+5, w, 7, color.RGBA{0x5c, 0x3a, 0x21, 0xff})
	r.StrokeRect(px, py+5, w, 7, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})

	// Dark/black interior showing it's empty
	r.FillRect(px+2, py+6, w-4, 4, color.RGBA{0x10, 0x10, 0x10, 0xff})

	// Lid: drawn angled/above the chest base.
	r.FillRect(px, py+1, w, 4, color.RGBA{0x5c, 0x3a, 0x21, 0xff})
	r.StrokeRect(px, py+1, w, 4, 1, color.RGBA{0x3e, 0x27, 0x16, 0xff})

	// Gold bands on the tilted lid
	r.FillRect(px+2, py+1, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
	r.FillRect(px+8, py+1, 2, 4, color.RGBA{0xd8, 0xa0, 0x20, 0xff})

	// Latch hanging down from the lid
	r.FillRect(px+5, py+3, 2, 3, color.RGBA{0xd8, 0xa0, 0x20, 0xff})
}

func (r *Renderer) pickupFallback(px, py float64) {
	vector.FillRect(r.screen, float32(px), float32(py), pickupFallbackPx, pickupFallbackPx,
		color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
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

func (r *Renderer) drawVectorTile(dst *ebiten.Image, gid int, x, y, w, h float32) {
	if fn, ok := tileDrawers[gid]; ok {
		fn(dst, x, y, w, h)
		return
	}
	// Fallback for unregistered GIDs: flat SwatchColor rect.
	vector.FillRect(dst, x, y, w, h, r.TileSwatchColor(gid), false)
}

func (r *Renderer) drawShrine(sh world.Shrine, ox, oy float64, tick int) {
	sx := float32(sh.Rect.X - ox)
	sy := float32(sh.Rect.Y - oy)
	sw := float32(sh.Rect.W)
	shh := float32(sh.Rect.H)

	// Pedestal Base Steps (14x3 at bottom)
	r.FillRect(sx+1, sy+shh-3, sw-2, 3, color.RGBA{0x55, 0x55, 0x5a, 0xff})
	r.FillRect(sx+2, sy+shh-6, sw-4, 3, color.RGBA{0x70, 0x70, 0x75, 0xff})

	// Pedestal Pillar Shaft (6x6 in middle)
	r.FillRect(sx+5, sy+shh-11, sw-10, 5, color.RGBA{0x80, 0x80, 0x85, 0xff})
	
	// Pillar side shadow and highlight lines
	r.FillRect(sx+5, sy+shh-11, 1, 5, color.RGBA{0xa0, 0xa0, 0xa5, 0xff}) // left highlight
	r.FillRect(sx+sw-6, sy+shh-11, 1, 5, color.RGBA{0x55, 0x55, 0x5a, 0xff}) // right shadow

	// Pedestal Top Altar slab (12x2)
	r.FillRect(sx+2, sy+shh-13, sw-4, 2, color.RGBA{0x90, 0x90, 0x98, 0xff})
	r.FillRect(sx+2, sy+shh-13, sw-4, 1, color.RGBA{0xc0, 0xc0, 0xc8, 0xff}) // Top lip highlight

	if !sh.Active {
		return
	}

	// Floating Crystal Animation (sine wave hover)
	offsetY := float32(math.Sin(float64(tick)*0.08) * 1.5)
	cy := sy + 2 + offsetY // center of floating area
	cx := sx + sw*0.5

	// Soft aura glow (behind crystal)
	vector.FillCircle(r.screen, cx, cy+1, 5, color.RGBA{0x00, 0xe0, 0xff, 0x33}, true)
	vector.FillCircle(r.screen, cx, cy+1, 3, color.RGBA{0x00, 0xe0, 0xff, 0x55}, true)

	// Draw diamond crystal path
	var path vector.Path
	path.MoveTo(cx, cy-4)
	path.LineTo(cx+3, cy+1)
	path.LineTo(cx, cy+6)
	path.LineTo(cx-3, cy+1)
	path.Close()
	drawPath(r.screen, &path, color.RGBA{0x00, 0xb0, 0xe0, 0xff}, true, 0)
	drawPath(r.screen, &path, color.RGBA{0x00, 0xe0, 0xff, 0xff}, false, 1.0)

	// Core reflection: small inner white diamond
	var corePath vector.Path
	corePath.MoveTo(cx, cy-2)
	corePath.LineTo(cx+1.5, cy+1)
	corePath.LineTo(cx, cy+4)
	corePath.LineTo(cx-1.5, cy+1)
	corePath.Close()
	drawPath(r.screen, &corePath, color.RGBA{0xe0, 0xfb, 0xff, 0xff}, true, 0)

	// Floating spark pixels (magic glow)
	sparkOffset1 := float32(math.Sin(float64(tick)*0.05) * 4)
	sparkOffset2 := float32(math.Cos(float64(tick)*0.06) * 4)
	r.FillRect(cx-5+sparkOffset1, cy-3+sparkOffset2, 1, 1, color.RGBA{0x00, 0xff, 0xff, 0xbb})
	r.FillRect(cx+4-sparkOffset2, cy-1-sparkOffset1, 1, 1, color.RGBA{0x00, 0xff, 0xff, 0x99})
}

func (r *Renderer) drawCharacter(x, y, w, h float32, dir world.Dir, isPlayer bool, isBoss bool) {
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

	// Main body rect
	r.FillRect(x, y, w, h, baseCol)
	r.StrokeRect(x, y, w, h, 1, borderCol)

	// Draw details based on direction
	switch dir {
	case world.DirDown:
		// Draw two eyes facing forward/down
		r.FillRect(x+2, y+4, 2, 2, eyeCol)
		r.FillRect(x+w-4, y+4, 2, 2, eyeCol)
		// Pupils
		r.FillRect(x+2, y+5, 1, 1, pupilCol)
		r.FillRect(x+w-4, y+5, 1, 1, pupilCol)
	case world.DirUp:
		// Draw hair / head cover (back of head) - a dark cap at the top
		r.FillRect(x+1, y+1, w-2, 4, borderCol)
	case world.DirLeft:
		// Draw eye looking left
		r.FillRect(x+2, y+4, 2, 2, eyeCol)
		r.FillRect(x+2, y+5, 1, 1, pupilCol)
	case world.DirRight:
		// Draw eye looking right
		r.FillRect(x+w-4, y+4, 2, 2, eyeCol)
		r.FillRect(x+w-4, y+5, 1, 1, pupilCol)
	}
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

	// Draw Handle (grip)
	vector.StrokeLine(screen,
		cx+handleStart*cos, cy+handleStart*sin,
		cx+handleEnd*cos, cy+handleEnd*sin,
		1.5, handleCol, true)

	// Draw Pommel
	vector.FillCircle(screen, cx+handleEnd*cos, cy+handleEnd*sin, 1.2, pommelCol, true)

	// Draw Crossguard
	const guardHalfWidth = 4.0
	vector.StrokeLine(screen,
		cx+guardDist*cos-guardHalfWidth*perpCos, cy+guardDist*sin-guardHalfWidth*perpSin,
		cx+guardDist*cos+guardHalfWidth*perpCos, cy+guardDist*sin+guardHalfWidth*perpSin,
		1.8, guardCol, true)

	// Draw Blade
	vector.StrokeLine(screen,
		cx+guardDist*cos, cy+guardDist*sin,
		cx+bladeEnd*cos, cy+bladeEnd*sin,
		2.2, bladeCol, true)

	// Draw Blade edge highlight
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

	// Draw Handle (grip)
	vector.StrokeLine(screen,
		cx+handleStart*cos, cy+handleStart*sin,
		cx+handleEnd*cos, cy+handleEnd*sin,
		1.5, handleCol, true)

	// Draw tip / metal casing
	vector.StrokeLine(screen,
		cx+tipStart*cos, cy+tipStart*sin,
		cx+tipEnd*cos, cy+tipEnd*sin,
		2.2, tipCol, true)

	// Draw flame (dynamic length based on tick/wobble)
	flicker := float32(math.Sin(float64(tick)*0.45)) * 1.5
	actualFlameEnd := flameEnd + flicker
	vector.StrokeLine(screen,
		cx+tipEnd*cos, cy+tipEnd*sin,
		cx+actualFlameEnd*cos, cy+actualFlameEnd*sin,
		3.0, flameCol, true)

	// Draw flame core
	vector.StrokeLine(screen,
		cx+tipEnd*cos, cy+tipEnd*sin,
		cx+(actualFlameEnd-3.0)*cos, cy+(actualFlameEnd-3.0)*sin,
		1.5, flameCoreCol, true)
}

