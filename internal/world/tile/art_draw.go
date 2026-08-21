package tile

import "image/color"

// SpatialHash is the deterministic grid hash used by spatial variation.
func SpatialHash(tx, ty int) uint32 {
	return uint32(tx*374761393+ty*668265263) ^ 0x5bf03635
}

// PickSpatialVariant returns the variant index for hash%100 given cumulative weights.
// Weights must sum to 100 (enforced by Validate).
func PickSpatialVariant(variants []SpatialVariant, roll uint32) int {
	r := int(roll % 100)
	cum := 0
	for i, v := range variants {
		cum += v.Weight
		if r < cum {
			return i
		}
	}
	if len(variants) == 0 {
		return -1
	}
	return len(variants) - 1
}

// DrawArt renders art at the destination rect, applying spatial variants when present.
func DrawArt(c Canvas, art *Art, x, y, w, h float32) {
	if art == nil {
		return
	}
	size := art.Size
	if size <= 0 {
		size = Size
	}
	sx := w / float32(size)
	sy := h / float32(size)

	drawShapes(c, art.Layers, x, y, sx, sy)

	if art.Spatial == nil || len(art.Spatial.Variants) == 0 {
		return
	}
	tx, ty := c.GridPos()
	idx := PickSpatialVariant(art.Spatial.Variants, SpatialHash(tx, ty))
	if idx < 0 {
		return
	}
	drawShapes(c, art.Spatial.Variants[idx].Layers, x, y, sx, sy)
}

func drawShapes(c Canvas, shapes []Shape, ox, oy, sx, sy float32) {
	for _, s := range shapes {
		drawShape(c, s, ox, oy, sx, sy)
	}
}

func drawShape(c Canvas, s Shape, ox, oy, sx, sy float32) {
	sw := s.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	switch s.Type {
	case "rect":
		rx, ry := ox+s.X*sx, oy+s.Y*sy
		rw, rh := s.W*sx, s.H*sy
		if s.Fill != "" {
			if col, err := ParseHexColor(s.Fill); err == nil {
				c.FillRect(rx, ry, rw, rh, col)
			}
		}
		if s.Stroke != "" {
			if col, err := ParseHexColor(s.Stroke); err == nil {
				c.StrokeRect(rx, ry, rw, rh, sw, col)
			}
		}
	case "line":
		if s.Stroke == "" {
			return
		}
		col, err := ParseHexColor(s.Stroke)
		if err != nil {
			return
		}
		c.StrokeLine(ox+s.X1*sx, oy+s.Y1*sy, ox+s.X2*sx, oy+s.Y2*sy, sw, col)
	case "circle":
		cx, cy, r := ox+s.CX*sx, oy+s.CY*sy, s.R*((sx+sy)*0.5)
		if s.Fill != "" {
			if col, err := ParseHexColor(s.Fill); err == nil {
				c.FillCircle(cx, cy, r, col)
			}
		}
		if s.Stroke != "" {
			if col, err := ParseHexColor(s.Stroke); err == nil {
				c.StrokeCircle(cx, cy, r, sw, col)
			}
		}
	case "path":
		var p Path
		for _, seg := range s.Segs {
			switch seg.Op {
			case "M":
				p.MoveTo(ox+seg.X*sx, oy+seg.Y*sy)
			case "L":
				p.LineTo(ox+seg.X*sx, oy+seg.Y*sy)
			case "Q":
				p.QuadTo(ox+seg.CX*sx, oy+seg.CY*sy, ox+seg.X*sx, oy+seg.Y*sy)
			case "Z":
				p.Close()
			}
		}
		fill := s.Fill != ""
		if s.Filled != nil {
			fill = *s.Filled
		}
		var col color.RGBA
		var err error
		if fill && s.Fill != "" {
			col, err = ParseHexColor(s.Fill)
		} else if s.Stroke != "" {
			col, err = ParseHexColor(s.Stroke)
		} else {
			return
		}
		if err != nil {
			return
		}
		c.DrawPath(p, col, fill, sw)
	}
}

// ArtDrawer returns a VectorDraw closure for the given art (copied by value).
func ArtDrawer(art Art) func(c Canvas, x, y, w, h float32) {
	a := art
	return func(c Canvas, x, y, w, h float32) {
		DrawArt(c, &a, x, y, w, h)
	}
}
