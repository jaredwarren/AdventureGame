// renderer.go — Ebiten-backed services.Renderer implementation.
//
// This type is the ONLY place in the repo allowed to hold a *ebiten.Image or
// call ebiten/vector/ebitenutil directly on behalf of scene-layer draws.
// Scenes import services.Renderer (an interface) and never see Ebiten types.
package ebitenplat

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"

	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

const useTileSprites = false
const pickupFallbackPx = 12

var (
	gameTextFace *text.GoXFace
)

func init() {
	gameTextFace = text.NewGoXFace(basicfont.Face7x13)
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

	drawList []drawItem
}

// NewRenderer builds the renderer. The Camera is owned by the renderer so
// scenes that need to tweak shake or reduce-motion do so through Camera().
func NewRenderer(cam *render.Camera, assets services.AssetCache) *Renderer {
	return &Renderer{
		camera:   cam,
		assets:   assets,
		drawList: make([]drawItem, 0, 64),
	}
}

// BeginFrame snapshots the camera + shake offset for the current frame.
func (r *Renderer) BeginFrame(screen *ebiten.Image) {
	r.screen = screen
	sx, sy := r.camera.Offset()
	r.camOffX = r.camera.X - sx
	r.camOffY = r.camera.Y - sy
}

// EndFrame releases the screen reference.
func (r *Renderer) EndFrame() { r.screen = nil }

// Camera returns the active render camera.
func (r *Renderer) Camera() *render.Camera { return r.camera }

// DrawText renders text at (x, y) screen pixels using default options.
func (r *Renderer) DrawText(x, y int, str string) {
	r.DrawTextOpt(x, y, str, services.TextOptions{})
}

// DrawTextOpt renders text at (x, y) screen pixels with custom scale and color.
func (r *Renderer) DrawTextOpt(x, y int, str string, opts services.TextOptions) {
	if r.screen == nil {
		return
	}
	op := &text.DrawOptions{}
	if opts.Color != nil {
		op.ColorScale.ScaleWithColor(opts.Color)
	}
	scale := opts.Scale
	if scale > 0 && scale != 1.0 {
		op.GeoM.Scale(scale, scale)
	}
	op.GeoM.Translate(float64(x), float64(y))
	text.Draw(r.screen, str, gameTextFace, op)
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
func (r *Renderer) TileSwatchColor(gid int) color.RGBA {
	return world.TileDefOf(gid).SwatchColor
}

// DrawTileScreen draws one tile for UI (editor palette).
func (r *Renderer) DrawTileScreen(gid int, x, y, dw, dh float32) {
	if r.screen == nil || dw < 1 || dh < 1 {
		return
	}
	if !useTileSprites || gid == tile.GIDEmpty {
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

func (r *Renderer) drawTile(gid int, wx, wy float32) {
	if gid == tile.GIDEmpty {
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
	op.GeoM.Scale(float64(tile.Size)/sw, float64(tile.Size)/sh)
	op.GeoM.Translate(float64(wx), float64(wy))
	r.screen.DrawImage(src, op)
}

func (r *Renderer) fallbackTile(gid int, wx, wy float32) {
	c := r.TileSwatchColor(gid)
	vector.FillRect(r.screen, wx, wy, tile.Size, tile.Size, c, false)
}

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

func (r *Renderer) drawVectorTile(dst *ebiten.Image, gid int, x, y, w, h float32) {
	c := &ebitenCanvas{dst: dst}
	tile.DefOf(gid).DrawVector(c, x, y, w, h)
}
