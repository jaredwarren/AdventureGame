// pickupdraw.go — vector fallback draw functions for each pickup kind.
//
// To add a new pickup: declare a singleton in world/pickup_kind.go, then add
// one entry to pickupDrawers below. No changes to renderer.go are required.
package ebitenplat

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/world"
)

type pickupDrawer func(dst *ebiten.Image, px, py, w, h float32)

var pickupDrawers = map[*world.PickupKind]pickupDrawer{
	world.PickupCoin:     drawPickupCoin,
	world.PickupHeart:    drawPickupHeart,
	world.PickupBomb:     drawPickupBomb,
	world.PickupSmallKey: drawPickupSmallKey,
	world.PickupTorch:        drawPickupTorch,
	world.PickupPegasusBoots: drawPickupPegasusBoots,
}

func drawPickupCoin(dst *ebiten.Image, px, py, w, h float32) {
	cx, cy := px+w*0.5, py+h*0.5
	rad := w * 0.5
	vector.FillCircle(dst, cx, cy, rad, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
	vector.StrokeCircle(dst, cx, cy, rad-0.5, 1, color.RGBA{0xd8, 0x7a, 0x00, 0xff}, true)
	vector.StrokeLine(dst, cx, cy-rad*0.5, cx, cy+rad*0.5, 1, color.RGBA{0xd8, 0x7a, 0x00, 0xff}, true)
}

func drawPickupHeart(dst *ebiten.Image, px, py, w, h float32) {
	path := &vector.Path{}
	path.MoveTo(px+w*0.5, py+h*0.35)
	path.QuadTo(px+w*0.1, py+h*0.05, px+w*0.1, py+h*0.48)
	path.LineTo(px+w*0.5, py+h*0.95)
	path.LineTo(px+w*0.9, py+h*0.48)
	path.QuadTo(px+w*0.9, py+h*0.05, px+w*0.5, py+h*0.35)
	path.Close()
	drawPath(dst, path, color.RGBA{0xe0, 0x30, 0x30, 0xff}, true, 0)
	drawPath(dst, path, color.RGBA{0x90, 0x10, 0x10, 0xff}, false, 1)
}

func drawPickupBomb(dst *ebiten.Image, px, py, w, h float32) {
	cx, cy := px+w*0.5, py+h*0.6
	rad := w * 0.35
	vector.FillCircle(dst, cx, cy, rad, color.RGBA{0x1a, 0x1a, 0x1a, 0xff}, true)
	vector.StrokeCircle(dst, cx, cy, rad, 1, color.RGBA{0x33, 0x33, 0x33, 0xff}, true)
	vector.FillRect(dst, px+w*0.4, py+h*0.2, w*0.2, h*0.1, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
	vector.StrokeLine(dst, px+w*0.5, py+h*0.2, px+w*0.7, py+h*0.05, 1.2, color.RGBA{0xd0, 0xc0, 0x90, 0xff}, true)
	vector.FillCircle(dst, px+w*0.7, py+h*0.05, 1.5, color.RGBA{0xff, 0x50, 0x00, 0xff}, true)
}

func drawPickupSmallKey(dst *ebiten.Image, px, py, w, h float32) {
	vector.StrokeCircle(dst, px+w*0.5, py+h*0.25, w*0.2, 1.5, color.RGBA{0xff, 0xd7, 0x00, 0xff}, true)
	vector.FillRect(dst, px+w*0.45, py+h*0.45, w*0.1, h*0.4, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
	vector.FillRect(dst, px+w*0.55, py+h*0.6, w*0.15, h*0.08, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
	vector.FillRect(dst, px+w*0.55, py+h*0.75, w*0.15, h*0.08, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
}

func drawPickupTorch(dst *ebiten.Image, px, py, w, h float32) {
	vector.FillRect(dst, px+w*0.42, py+h*0.45, w*0.16, h*0.5, color.RGBA{0x8b, 0x5a, 0x2b, 0xff}, false)
	vector.FillRect(dst, px+w*0.35, py+h*0.38, w*0.3, h*0.08, color.RGBA{0x4a, 0x4a, 0x4a, 0xff}, false)
	flame := &vector.Path{}
	flame.MoveTo(px+w*0.5, py+h*0.05)
	flame.QuadTo(px+w*0.2, py+h*0.3, px+w*0.3, py+h*0.38)
	flame.LineTo(px+w*0.7, py+h*0.38)
	flame.QuadTo(px+w*0.8, py+h*0.3, px+w*0.5, py+h*0.05)
	flame.Close()
	drawPath(dst, flame, color.RGBA{0xff, 0x3b, 0x00, 0xff}, true, 0)
	innerFlame := &vector.Path{}
	innerFlame.MoveTo(px+w*0.5, py+h*0.15)
	innerFlame.QuadTo(px+w*0.3, py+h*0.3, px+w*0.35, py+h*0.38)
	innerFlame.LineTo(px+w*0.65, py+h*0.38)
	innerFlame.QuadTo(px+w*0.7, py+h*0.3, px+w*0.5, py+h*0.15)
	innerFlame.Close()
	drawPath(dst, innerFlame, color.RGBA{0xff, 0xa5, 0x00, 0xff}, true, 0)
}

func drawPickupPegasusBoots(dst *ebiten.Image, px, py, w, h float32) {
	// Boot body (red/brown speed boot)
	vector.FillRect(dst, px+w*0.25, py+h*0.3, w*0.3, h*0.5, color.RGBA{0xb0, 0x25, 0x25, 0xff}, false)
	vector.FillRect(dst, px+w*0.25, py+h*0.65, w*0.55, h*0.25, color.RGBA{0xb0, 0x25, 0x25, 0xff}, false)
	// Gold trim & wing accent
	vector.FillRect(dst, px+w*0.25, py+h*0.3, w*0.3, h*0.1, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
	wing := &vector.Path{}
	wing.MoveTo(px+w*0.15, py+h*0.35)
	wing.LineTo(px+w*0.35, py+h*0.45)
	wing.LineTo(px+w*0.2, py+h*0.55)
	wing.Close()
	drawPath(dst, wing, color.RGBA{0xff, 0xf0, 0xaa, 0xff}, true, 0)
}
