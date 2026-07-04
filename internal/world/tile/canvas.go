package tile

import "image/color"

// Canvas provides a headless vector drawing interface for tile vector fallbacks.
type Canvas interface {
	FillRect(x, y, w, h float32, c color.RGBA)
	StrokeRect(x, y, w, h float32, strokeWidth float32, c color.RGBA)
	StrokeLine(x1, y1, x2, y2 float32, strokeWidth float32, c color.RGBA)
	FillCircle(cx, cy, r float32, c color.RGBA)
	StrokeCircle(cx, cy, r float32, strokeWidth float32, c color.RGBA)
	DrawPath(p Path, c color.RGBA, fill bool, strokeWidth float32)
}

// PathOpKind specifies the operation type in a Path segment.
type PathOpKind int

const (
	OpMoveTo PathOpKind = iota
	OpLineTo
	OpQuadTo
	OpClose
)

// PathOp describes a single path operation segment.
type PathOp struct {
	Kind             PathOpKind
	X, Y             float32
	ControlX, ControlY float32
}

// Path represents a sequence of 2D vector path operations.
type Path struct {
	Ops []PathOp
}

func (p *Path) MoveTo(x, y float32) {
	p.Ops = append(p.Ops, PathOp{Kind: OpMoveTo, X: x, Y: y})
}

func (p *Path) LineTo(x, y float32) {
	p.Ops = append(p.Ops, PathOp{Kind: OpLineTo, X: x, Y: y})
}

func (p *Path) QuadTo(controlX, controlY, x, y float32) {
	p.Ops = append(p.Ops, PathOp{Kind: OpQuadTo, X: x, Y: y, ControlX: controlX, ControlY: controlY})
}

func (p *Path) Close() {
	p.Ops = append(p.Ops, PathOp{Kind: OpClose})
}
