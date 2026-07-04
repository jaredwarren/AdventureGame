package ebitenplat

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

type ebitenCanvas struct {
	dst *ebiten.Image
}

func (c *ebitenCanvas) FillRect(x, y, w, h float32, clr color.RGBA) {
	vector.FillRect(c.dst, x, y, w, h, clr, false)
}

func (c *ebitenCanvas) StrokeRect(x, y, w, h float32, strokeWidth float32, clr color.RGBA) {
	vector.StrokeRect(c.dst, x, y, w, h, strokeWidth, clr, false)
}

func (c *ebitenCanvas) StrokeLine(x1, y1, x2, y2 float32, strokeWidth float32, clr color.RGBA) {
	vector.StrokeLine(c.dst, x1, y1, x2, y2, strokeWidth, clr, false)
}

func (c *ebitenCanvas) FillCircle(cx, cy, r float32, clr color.RGBA) {
	vector.FillCircle(c.dst, cx, cy, r, clr, true)
}

func (c *ebitenCanvas) StrokeCircle(cx, cy, r float32, strokeWidth float32, clr color.RGBA) {
	vector.StrokeCircle(c.dst, cx, cy, r, strokeWidth, clr, true)
}

func (c *ebitenCanvas) DrawPath(p tile.Path, clr color.RGBA, fill bool, strokeWidth float32) {
	var ep vector.Path
	for _, op := range p.Ops {
		switch op.Kind {
		case tile.OpMoveTo:
			ep.MoveTo(op.X, op.Y)
		case tile.OpLineTo:
			ep.LineTo(op.X, op.Y)
		case tile.OpQuadTo:
			ep.QuadTo(op.ControlX, op.ControlY, op.X, op.Y)
		case tile.OpClose:
			ep.Close()
		}
	}
	drawPath(c.dst, &ep, clr, fill, strokeWidth)
}
