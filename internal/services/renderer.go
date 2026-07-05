// renderer.go — Renderer port for drawing into the current frame.
//
// The Renderer is the only draw surface scenes (and scene-layer helpers in
// internal/scenes) are allowed to touch. It abstracts over the underlying
// graphics framework (Ebiten today, something else tomorrow) and over the
// per-frame screen target so scene code never handles a raw *ebiten.Image.
//
// Lifetime conventions:
//
//   - App calls the platform-specific BeginFrame/EndFrame around each Draw;
//     those are concrete methods on the implementation, not on this
//     interface. All Renderer methods below are safe to call between a
//     BeginFrame and its matching EndFrame — and UB outside that window.
//
//   - Camera() returns a pointer into the renderer's own camera; scenes
//     mutate it directly (Follow, shake, reduce-motion). The renderer reads
//     from that same camera during DrawWorld and DrawTileScreen.
//
//   - All Draw methods use screen-space pixels in the engine's internal
//     resolution (320x240 today). DrawWorld is the one exception: it takes a
//     World and applies the camera offset internally so callers write
//     world-space coordinates in entities/tiles.
package services

import (
	"image/color"

	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/world"
)

// TextOptions configures scale and color for text rendering.
type TextOptions struct {
	Scale float64     // 0 defaults to 1.0 (normal size)
	Color color.Color // nil defaults to default white text
}

// Renderer is the scene-facing draw API.
//
// Implementations must be safe to call with a nil world/screen in DrawWorld
// (no-op) — scenes don't always have live gameplay state (title, editor pre-
// load), and a runtime panic in that case would be worse than a blank frame.
type Renderer interface {
	// Camera exposes the active render camera. Scene logic that needs to
	// manipulate camera state (Follow, AddShake, ReduceShake, etc.) does so
	// through this accessor; the renderer reads from the same camera when
	// it draws.
	Camera() *render.Camera

	// DrawWorld renders the tilemap + entities for w using the camera
	// offset captured at BeginFrame. No-op when w is nil.
	DrawWorld(w *world.World)

	// DrawTileScreen draws one ground tile GID in screen space at (x, y)
	// with size (dw, dh). Used for the editor brush palette; uses the same
	// tile atlas as DrawWorld when available, otherwise TileSwatchColor.
	DrawTileScreen(gid int, x, y, dw, dh float32)

	// DrawPickupScreen draws one pickup kind in screen space at (x, y)
	// with size (dw, dh). Used for HUD and toast notifications.
	DrawPickupScreen(kind *world.PickupKind, x, y, dw, dh float32)

	// DrawText renders a single line of debug-style text in screen space.
	// (x, y) is the top-left in internal resolution pixels. Font, size, and
	// color are renderer-defined (small pixel font today).
	DrawText(x, y int, text string)

	// DrawTextOpt renders text in screen space with custom scale and color.
	DrawTextOpt(x, y int, text string, opts TextOptions)

	// FillRect fills a screen-space rectangle.
	FillRect(x, y, w, h float32, c color.RGBA)

	// StrokeRect outlines a screen-space rectangle with the given line width.
	StrokeRect(x, y, w, h, lw float32, c color.RGBA)

	// StrokeLine draws a line segment in screen space with the given width.
	StrokeLine(x1, y1, x2, y2, lw float32, c color.RGBA)

	// TileSwatchColor returns the flat fallback color a GID renders as when
	// the atlas is unavailable or for solid fills (e.g. GIDEmpty in palette).
	TileSwatchColor(gid int) color.RGBA
}
