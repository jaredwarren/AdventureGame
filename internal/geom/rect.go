// Package geom holds tiny shared math types to keep collision code decoupled from world/game packages.
package geom

// Rect is an axis-aligned box in world pixel space (origin top-left of the box).
type Rect struct {
	X, Y, W, H float64
}

func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.X+o.W && r.X+r.W > o.X && r.Y < o.Y+o.H && r.Y+r.H > o.Y
}

// OverlapsExpanded is true if r overlaps o after o is expanded by margin on
// every side (counts flush / nearly-touching boxes as overlapping).
func (r Rect) OverlapsExpanded(o Rect, margin float64) bool {
	if margin <= 0 {
		return r.Overlaps(o)
	}
	return r.Overlaps(Rect{X: o.X - margin, Y: o.Y - margin, W: o.W + 2*margin, H: o.H + 2*margin})
}

func (r Rect) Center() (float64, float64) {
	return r.X + r.W*0.5, r.Y + r.H*0.5
}
