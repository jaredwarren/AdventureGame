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
		r.drawPickup(p, ox, oy)
	}

	for _, e := range w.Enemies {
		if e.HP <= 0 {
			continue
		}
		col := color.RGBA{0xc0, 0x30, 0x30, 0xff}
		if e.IsBoss {
			col = color.RGBA{0x70, 0x10, 0x10, 0xff}
		}
		vector.FillRect(screen,
			float32(e.X-ox), float32(e.Y-oy), 14, 12, col, false)
	}

	pr := w.PlayerRect()
	vector.FillRect(screen,
		float32(pr.X-ox), float32(pr.Y-oy), float32(pr.W), float32(pr.H),
		color.RGBA{0x40, 0x70, 0xff, 0xff}, false)

	if hb, ok := w.SwordHitbox(); ok {
		vector.FillRect(screen,
			float32(hb.X-ox), float32(hb.Y-oy), float32(hb.W), float32(hb.H),
			color.RGBA{0xff, 0xff, 0xff, 0x80}, false)
	}

	for _, d := range w.Doors {
		rd := d.Rect
		vector.FillRect(screen,
			float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H),
			color.RGBA{0xe0, 0x90, 0x18, 0x55}, false)
		vector.StrokeRect(screen,
			float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H), 2,
			color.RGBA{0xff, 0xc8, 0x40, 0xff}, false)
	}

	for _, sh := range w.Shrines {
		rs := sh.Rect
		vector.StrokeRect(screen,
			float32(rs.X-ox), float32(rs.Y-oy), float32(rs.W), float32(rs.H), 2,
			color.RGBA{0x80, 0xe0, 0xff, 0xff}, false)
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

// drawPickup renders one pickup entity using the pickup atlas. (ox,oy) are
// the camera offsets; (p.X,p.Y) is world space.
func (r *Renderer) drawPickup(p world.Pickup, ox, oy float64) {
	px := float32(p.X - ox)
	py := float32(p.Y - oy)
	w := float32(p.W)
	h := float32(p.H)

	switch p.Kind {
	case world.PickupCoin:
		cx, cy := px+w*0.5, py+h*0.5
		rad := w * 0.5
		// Body: Filled gold circle
		vector.FillCircle(r.screen, cx, cy, rad, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
		// Border: Orange/brown thin circular stroke
		vector.StrokeCircle(r.screen, cx, cy, rad-0.5, 1, color.RGBA{0xd8, 0x7a, 0x00, 0xff}, true)
		// Detail: vertical slot line in center
		vector.StrokeLine(r.screen, cx, cy-rad*0.5, cx, cy+rad*0.5, 1, color.RGBA{0xd8, 0x7a, 0x00, 0xff}, true)

	case world.PickupHeart:
		// Heart shape using vector.Path
		path := &vector.Path{}
		path.MoveTo(px+w*0.5, py+h*0.35)
		path.QuadTo(px+w*0.1, py+h*0.05, px+w*0.1, py+h*0.48)
		path.LineTo(px+w*0.5, py+h*0.95)
		path.LineTo(px+w*0.9, py+h*0.48)
		path.QuadTo(px+w*0.9, py+h*0.05, px+w*0.5, py+h*0.35)
		path.Close()

		drawPath(r.screen, path, color.RGBA{0xe0, 0x30, 0x30, 0xff}, true, 0)
		drawPath(r.screen, path, color.RGBA{0x90, 0x10, 0x10, 0xff}, false, 1)

	case world.PickupBomb:
		// Body: Dark/black circle
		cx, cy := px+w*0.5, py+h*0.6
		rad := w * 0.35
		vector.FillCircle(r.screen, cx, cy, rad, color.RGBA{0x1a, 0x1a, 0x1a, 0xff}, true)
		vector.StrokeCircle(r.screen, cx, cy, rad, 1, color.RGBA{0x33, 0x33, 0x33, 0xff}, true)
		// Fuse nozzle: Grey rectangle
		vector.FillRect(r.screen, px+w*0.4, py+h*0.2, w*0.2, h*0.1, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
		// Fuse line: curved line
		vector.StrokeLine(r.screen, px+w*0.5, py+h*0.2, px+w*0.7, py+h*0.05, 1.2, color.RGBA{0xd0, 0xc0, 0x90, 0xff}, true)
		// Spark: small orange circle
		vector.FillCircle(r.screen, px+w*0.7, py+h*0.05, 1.5, color.RGBA{0xff, 0x50, 0x00, 0xff}, true)

	case world.PickupSmallKey:
		// Handle/Head: hollow gold circle
		vector.StrokeCircle(r.screen, px+w*0.5, py+h*0.25, w*0.2, 1.5, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
		// Stem/Shaft: gold rectangle
		vector.FillRect(r.screen, px+w*0.45, py+h*0.45, w*0.1, h*0.4, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
		// Teeth: two gold rectangles
		vector.FillRect(r.screen, px+w*0.55, py+h*0.6, w*0.15, h*0.08, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
		vector.FillRect(r.screen, px+w*0.55, py+h*0.75, w*0.15, h*0.08, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)

	case world.PickupTorch:
		// Handle: brown rectangle
		vector.FillRect(r.screen, px+w*0.42, py+h*0.45, w*0.16, h*0.5, color.RGBA{0x8b, 0x5a, 0x2b, 0xff}, false)
		// Nozzle: grey rect
		vector.FillRect(r.screen, px+w*0.35, py+h*0.38, w*0.3, h*0.08, color.RGBA{0x4a, 0x4a, 0x4a, 0xff}, false)
		// Flame (outer): red teardrop path
		flame := &vector.Path{}
		flame.MoveTo(px+w*0.5, py+h*0.05)
		flame.QuadTo(px+w*0.2, py+h*0.3, px+w*0.3, py+h*0.38)
		flame.LineTo(px+w*0.7, py+h*0.38)
		flame.QuadTo(px+w*0.8, py+h*0.3, px+w*0.5, py+h*0.05)
		flame.Close()
		drawPath(r.screen, flame, color.RGBA{0xff, 0x3b, 0x00, 0xff}, true, 0)
		// Flame (inner): orange teardrop path
		innerFlame := &vector.Path{}
		innerFlame.MoveTo(px+w*0.5, py+h*0.15)
		innerFlame.QuadTo(px+w*0.3, py+h*0.3, px+w*0.35, py+h*0.38)
		innerFlame.LineTo(px+w*0.65, py+h*0.38)
		innerFlame.QuadTo(px+w*0.7, py+h*0.3, px+w*0.5, py+h*0.15)
		innerFlame.Close()
		drawPath(r.screen, innerFlame, color.RGBA{0xff, 0xa5, 0x00, 0xff}, true, 0)

	default:
		// Fallback
		r.pickupFallback(float64(px), float64(py))
	}
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
	grassColor := color.RGBA{0x2b, 0x4a, 0x2b, 0xff}
	waterColor := color.RGBA{0x2a, 0x4a, 0x8a, 0xff}
	shoreLineColor := color.RGBA{0xe0, 0xd0, 0xa0, 0xff}

	switch gid {
	case world.GIDEmpty:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)

	case world.GIDGrass:
		vector.FillRect(dst, x, y, w, h, grassColor, false)
		// Draw stylized grass blades
		vector.StrokeLine(dst, x+w*0.3, y+h*0.7, x+w*0.3, y+h*0.4, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.3, y+h*0.7, x+w*0.45, y+h*0.5, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.7, y+h*0.5, x+w*0.7, y+h*0.2, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.7, y+h*0.5, x+w*0.8, y+h*0.3, 1, color.RGBA{0x3e, 0x6e, 0x3e, 0xff}, false)

	case world.GIDWall:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x40, 0x40, 0x50, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x60, 0x60, 0x70, 0xff}, false)
		// Horizontal brick line
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		// Vertical brick lines
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h*0.5, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.25, y+h*0.5, x+w*0.25, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.75, y+h*0.5, x+w*0.75, y+h, 1, color.RGBA{0x25, 0x25, 0x30, 0xff}, false)

	case world.GIDCracked:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8b, 0x6a, 0x4a, 0xff}, false)
		// Jagged cracks
		vector.StrokeLine(dst, x+w*0.2, y+h*0.2, x+w*0.5, y+h*0.4, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.5, y+h*0.4, x+w*0.4, y+h*0.7, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.4, y+h*0.7, x+w*0.8, y+h*0.8, 1.2, color.RGBA{0x20, 0x10, 0x08, 0xff}, false)

	case world.GIDDoor:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x3a, 0x5a, 0x3a, 0xff}, false)
		// Door arch
		vector.FillRect(dst, x+w*0.2, y+h*0.2, w*0.6, h*0.8, color.RGBA{0x15, 0x25, 0x15, 0xff}, false)
		vector.StrokeRect(dst, x+w*0.2, y+h*0.2, w*0.6, h*0.8, 1, color.RGBA{0xe0, 0xc0, 0x30, 0xff}, false)

	case world.GIDWater:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		// Waves
		vector.StrokeLine(dst, x+w*0.2, y+h*0.3, x+w*0.4, y+h*0.3, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff}, false)
		vector.StrokeLine(dst, x+w*0.5, y+h*0.7, x+w*0.8, y+h*0.7, 1, color.RGBA{0x4a, 0x6a, 0xaa, 0xff}, false)

	case world.GIDLock:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x6a, 0x2a, 0x7a, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x8a, 0x4a, 0x9a, 0xff}, false)
		// Gold keyhole lock
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.18, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
		vector.FillRect(dst, x+w*0.42, y+h*0.4, w*0.16, h*0.3, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.07, color.RGBA{0, 0, 0, 255}, true)
		vector.FillRect(dst, x+w*0.47, y+h*0.42, w*0.06, h*0.18, color.RGBA{0, 0, 0, 255}, false)

	case world.GIDFloor2:
		vector.FillRect(dst, x, y, w, h, color.RGBA{0x3a, 0x3a, 0x44, 0xff}, false)
		vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, color.RGBA{0x4a, 0x4a, 0x54, 0xff}, false)
		vector.StrokeRect(dst, x+w*0.25, y+h*0.25, w*0.5, h*0.5, 1, color.RGBA{0x2a, 0x2a, 0x34, 0xff}, false)

	case world.GIDTree:
		vector.FillRect(dst, x, y, w, h, grassColor, false)
		vector.FillRect(dst, x+w*0.4, y+h*0.6, w*0.2, h*0.3, color.RGBA{0x6b, 0x4a, 0x2a, 0xff}, false)
		vector.FillCircle(dst, x+w*0.5, y+h*0.4, w*0.3, color.RGBA{0x2d, 0x7a, 0x2a, 0xff}, true)
		vector.StrokeCircle(dst, x+w*0.5, y+h*0.4, w*0.3, 1, color.RGBA{0x1d, 0x5a, 0x1a, 0xff}, true)

	// Straight shores
	case world.GIDWaterShoreTop:
		vector.FillRect(dst, x, y, w, h*0.5, grassColor, false)
		vector.FillRect(dst, x, y+h*0.5, w, h*0.5, waterColor, false)
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, shoreLineColor, false)
	case world.GIDWaterShoreBottom:
		vector.FillRect(dst, x, y, w, h*0.5, waterColor, false)
		vector.FillRect(dst, x, y+h*0.5, w, h*0.5, grassColor, false)
		vector.StrokeLine(dst, x, y+h*0.5, x+w, y+h*0.5, 1, shoreLineColor, false)
	case world.GIDWaterShoreLeft:
		vector.FillRect(dst, x, y, w*0.5, h, grassColor, false)
		vector.FillRect(dst, x+w*0.5, y, w*0.5, h, waterColor, false)
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h, 1, shoreLineColor, false)
	case world.GIDWaterShoreRight:
		vector.FillRect(dst, x, y, w*0.5, h, waterColor, false)
		vector.FillRect(dst, x+w*0.5, y, w*0.5, h, grassColor, false)
		vector.StrokeLine(dst, x+w*0.5, y, x+w*0.5, y+h, 1, shoreLineColor, false)

	// Convex outer corners
	case world.GIDWaterShoreNW:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
		path.LineTo(x, y+h)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreNE:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h)
		path.LineTo(x+w*0.5, y+h)
		path.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y+h)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreSW:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w*0.5, y)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
		path.LineTo(x+w, y+h)
		path.LineTo(x, y+h)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreSE:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x+w*0.5, y)
		path.LineTo(x+w, y)
		path.LineTo(x+w, y+h)
		path.LineTo(x, y+h)
		path.LineTo(x, y+h*0.5)
		path.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x, y+h*0.5)
		linePath.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	// Concave inner corners
	case world.GIDWaterShoreNWInner:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x, y)
		path.LineTo(x+w*0.5, y)
		path.QuadTo(x, y, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y)
		linePath.QuadTo(x, y, x, y+h*0.5)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreNEInner:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x+w, y)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w, y, x+w*0.5, y)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w, y, x+w*0.5, y)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreSWInner:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x, y+h)
		path.LineTo(x+w*0.5, y+h)
		path.QuadTo(x, y+h, x, y+h*0.5)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w*0.5, y+h)
		linePath.QuadTo(x, y+h, x, y+h*0.5)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	case world.GIDWaterShoreSEInner:
		vector.FillRect(dst, x, y, w, h, waterColor, false)
		var path vector.Path
		path.MoveTo(x+w, y+h)
		path.LineTo(x+w, y+h*0.5)
		path.QuadTo(x+w, y+h, x+w*0.5, y+h)
		path.Close()
		drawPath(dst, &path, grassColor, true, 0)
		var linePath vector.Path
		linePath.MoveTo(x+w, y+h*0.5)
		linePath.QuadTo(x+w, y+h, x+w*0.5, y+h)
		drawPath(dst, &linePath, shoreLineColor, false, 1)

	default:
		// Fallback to TileSwatchColor
		vector.FillRect(dst, x, y, w, h, r.TileSwatchColor(gid), false)
	}
}
