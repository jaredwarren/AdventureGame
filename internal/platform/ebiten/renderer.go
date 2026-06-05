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
const useTileSprites = true

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
		vector.FillRect(r.screen, x, y, dw, dh, r.TileSwatchColor(gid), false)
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
				c := r.TileSwatchColor(gid)
				vector.FillRect(screen, x, y, world.TileSize, world.TileSize, c, false)
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
	px, py := p.X-ox, p.Y-oy

	atlas, err := r.assets.Atlas(services.AtlasPickup)
	if err != nil || atlas == nil {
		r.pickupFallback(px, py)
		return
	}
	idx := int(p.Kind)
	if idx < 0 || idx >= atlas.Count() {
		idx = 0
	}
	fr := atlas.Frame(idx)
	if fr.Skip || fr.Image == nil {
		r.pickupFallback(px, py)
		return
	}
	src := NativeImage(fr.Image)
	if src == nil {
		r.pickupFallback(px, py)
		return
	}
	b := src.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(fr.DstW/sw, fr.DstH/sh)
	op.GeoM.Translate(px+fr.OffsetX, py+fr.OffsetY)
	r.screen.DrawImage(src, op)
}

func (r *Renderer) pickupFallback(px, py float64) {
	vector.FillRect(r.screen, float32(px), float32(py), pickupFallbackPx, pickupFallbackPx,
		color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
}
