package tile

import "image/color"

// EdgeDirection represents the edge direction of a straight transition tile.
type EdgeDirection int

const (
	EdgeTop EdgeDirection = iota
	EdgeBottom
	EdgeLeft
	EdgeRight
)

// Corner represents the corner orientation of a corner transition tile.
type Corner int

const (
	CornerNE Corner = iota
	CornerNW
	CornerSW
	CornerSE
)

// TileStyle specifies the colors, stroke widths, and optional details for parametric vector drawing.
type TileStyle struct {
	FillColor   color.RGBA
	EdgeColor   color.RGBA
	DetailColor color.RGBA
	LineWidth   float32
	HasDetail   bool
	BaseDrawer  func(c Canvas, x, y, w, h float32)
}

func makeEdgeDrawer(dir EdgeDirection, style TileStyle) func(c Canvas, x, y, w, h float32) {
	lineWidth := style.LineWidth
	if lineWidth <= 0 {
		lineWidth = 1.0
	}
	hasEdgeColor := style.EdgeColor.A > 0

	switch dir {
	case EdgeTop:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y+h*0.5, w, h*0.5, style.FillColor)
			if hasEdgeColor {
				c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, lineWidth, style.EdgeColor)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.5, y+h*0.75, 1.5, style.DetailColor)
			}
		}
	case EdgeBottom:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w, h*0.5, style.FillColor)
			if hasEdgeColor {
				c.StrokeLine(x, y+h*0.5, x+w, y+h*0.5, lineWidth, style.EdgeColor)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.5, y+h*0.25, 1.5, style.DetailColor)
			}
		}
	case EdgeLeft:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x+w*0.5, y, w*0.5, h, style.FillColor)
			if hasEdgeColor {
				c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, lineWidth, style.EdgeColor)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.75, y+h*0.5, 1.5, style.DetailColor)
			}
		}
	case EdgeRight:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w*0.5, h, style.FillColor)
			if hasEdgeColor {
				c.StrokeLine(x+w*0.5, y, x+w*0.5, y+h, lineWidth, style.EdgeColor)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.25, y+h*0.5, 1.5, style.DetailColor)
			}
		}
	default:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w, h, style.FillColor)
		}
	}
}

func makeOuterCornerDrawer(corner Corner, style TileStyle) func(c Canvas, x, y, w, h float32) {
	lineWidth := style.LineWidth
	if lineWidth <= 0 {
		lineWidth = 1.0
	}
	hasEdgeColor := style.EdgeColor.A > 0

	switch corner {
	case CornerNW:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w, y+h*0.5)
				lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y+h)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.75, y+h*0.75, 1.5, style.DetailColor)
			}
		}
	case CornerNE:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y+h*0.5, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w*0.5, y+h)
				lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.25, y+h*0.75, 1.5, style.DetailColor)
			}
		}
	case CornerSW:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x+w*0.5, y, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w*0.5, y)
				lp.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.75, y+h*0.25, 1.5, style.DetailColor)
			}
		}
	case CornerSE:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x, y+h*0.5)
				lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.25, y+h*0.25, 1.5, style.DetailColor)
			}
		}
	default:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w, h, style.FillColor)
		}
	}
}

func makeInnerCornerDrawer(corner Corner, style TileStyle) func(c Canvas, x, y, w, h float32) {
	lineWidth := style.LineWidth
	if lineWidth <= 0 {
		lineWidth = 1.0
	}
	hasEdgeColor := style.EdgeColor.A > 0

	switch corner {
	case CornerNW:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x+w*0.5, y, w*0.5, h, style.FillColor)
			c.FillRect(x, y+h*0.5, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w*0.5, y)
				lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.7, y+h*0.3, 1.5, style.DetailColor)
				c.FillCircle(x+w*0.3, y+h*0.7, 1.5, style.DetailColor)
			}
		}
	case CornerNE:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w*0.5, h, style.FillColor)
			c.FillRect(x+w*0.5, y+h*0.5, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w, y+h*0.5)
				lp.QuadTo(x+w*0.5, y+h*0.5, x+w*0.5, y)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.3, y+h*0.3, 1.5, style.DetailColor)
				c.FillCircle(x+w*0.7, y+h*0.7, 1.5, style.DetailColor)
			}
		}
	case CornerSW:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x+w*0.5, y, w*0.5, h, style.FillColor)
			c.FillRect(x, y, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w*0.5, y+h)
				lp.QuadTo(x+w*0.5, y+h*0.5, x, y+h*0.5)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.7, y+h*0.7, 1.5, style.DetailColor)
				c.FillCircle(x+w*0.3, y+h*0.3, 1.5, style.DetailColor)
			}
		}
	case CornerSE:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w*0.5, h, style.FillColor)
			c.FillRect(x+w*0.5, y, w*0.5, h*0.5, style.FillColor)
			if hasEdgeColor {
				var lp Path
				lp.MoveTo(x+w*0.5, y+h)
				lp.QuadTo(x+w*0.5, y+h*0.5, x+w, y+h*0.5)
				c.DrawPath(lp, style.EdgeColor, false, lineWidth)
			}
			if style.HasDetail {
				c.FillCircle(x+w*0.3, y+h*0.7, 1.5, style.DetailColor)
				c.FillCircle(x+w*0.7, y+h*0.3, 1.5, style.DetailColor)
			}
		}
	default:
		return func(c Canvas, x, y, w, h float32) {
			c.FillRect(x, y, w, h, style.FillColor)
		}
	}
}

func makeBaseDrawer(style TileStyle) func(c Canvas, x, y, w, h float32) {
	if style.BaseDrawer != nil {
		return style.BaseDrawer
	}
	hasEdgeColor := style.EdgeColor.A > 0
	return func(c Canvas, x, y, w, h float32) {
		c.FillRect(x, y, w, h, style.FillColor)
		if hasEdgeColor {
			c.StrokeRect(x+0.5, y+0.5, w-1, h-1, 1, style.EdgeColor)
		}
	}
}
