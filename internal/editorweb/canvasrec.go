package editorweb

import (
	"fmt"
	"image/color"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// The draw-op wire format.
//
// Tile art in this project is procedural vector code (see
// internal/world/tile/draw.go): ~54 drawer funcs that write to the headless
// tile.Canvas interface. Reimplementing those in JavaScript would be the whole
// project, and would drift the moment anyone edited a drawer.
//
// Instead, recorder implements tile.Canvas by *capturing* the draw calls
// instead of rasterizing them. Each call becomes one compact JSON array whose
// first element is a short opcode and whose last element is an index into a
// shared color table. The browser replays those arrays onto a Canvas2D, whose
// primitives map 1:1 onto tile.Canvas:
//
//	fr  ["fr", x, y, w, h, c]              -> fillRect
//	sr  ["sr", x, y, w, h, sw, c]          -> strokeRect
//	sl  ["sl", x1, y1, x2, y2, sw, c]      -> beginPath/moveTo/lineTo/stroke
//	fc  ["fc", cx, cy, r, c]               -> arc(...,0,2*PI) + fill
//	sc  ["sc", cx, cy, r, sw, c]           -> arc + stroke
//	p   ["p", segs, fill01, sw, c]         -> path; see pathSegs below
//
// Coordinates are kept as float32 all the way to the marshaler on purpose:
// encoding/json emits the shortest round-trippable float32 form, so a drawer's
// `x + w*0.3` with w=16 serializes as 4.8, not 4.800000190734863. Widening to
// float64 would roughly triple the payload for no gain.
//
// Unknown opcodes are ignored by the client, so adding a primitive to
// tile.Canvas can never white-screen the editor.

// colorTable dedups colors across every recorded tile so ops carry a small
// integer instead of a repeated "#rrggbb" string. It also keeps the replay hot
// loop allocation-free: the client indexes a pre-built string array.
type colorTable struct {
	index map[string]int
	list  []string
}

func newColorTable() *colorTable {
	return &colorTable{index: make(map[string]int)}
}

func (t *colorTable) intern(c color.RGBA) int {
	s := hexColor(c)
	if i, ok := t.index[s]; ok {
		return i
	}
	i := len(t.list)
	t.index[s] = i
	t.list = append(t.list, s)
	return i
}

// hexColor formats a color.RGBA as a CSS hex string.
//
// color.RGBA is alpha-premultiplied; CSS #rrggbbaa is not. Every color in
// internal/world/tile is currently fully opaque (pinned by
// TestAllTileColorsAreOpaque), so this always takes the 7-char branch in
// practice. The un-premultiply branch exists so that introducing a translucent
// tile someday produces correct colors in the browser instead of silently
// darkened ones.
func hexColor(c color.RGBA) string {
	switch c.A {
	case 0xff:
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	case 0x00:
		return "#00000000"
	}
	un := func(v uint8) uint8 {
		x := int(v) * 255 / int(c.A)
		if x > 255 {
			x = 255
		}
		return uint8(x)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", un(c.R), un(c.G), un(c.B), c.A)
}

// recorder implements tile.Canvas by capturing draw calls as wire-format ops.
//
// It is the only place in the repo that knows how tile vector art crosses the
// Go/JS boundary.
type recorder struct {
	colors *colorTable
	ops    [][]any

	tick   int
	gx, gy int

	// usedTick / usedGrid record whether the drawer actually consulted the
	// animation inputs. tileart.go uses them to decide whether a tile needs a
	// frame sweep, and TestOnlyWaterAndLavaAreAnimated pins the resulting set —
	// that invariant is what makes a static atlas correct for every other tile.
	usedTick bool
	usedGrid bool
}

func newRecorder(colors *colorTable) *recorder {
	return &recorder{colors: colors}
}

func (r *recorder) FillRect(x, y, w, h float32, c color.RGBA) {
	r.ops = append(r.ops, []any{"fr", x, y, w, h, r.colors.intern(c)})
}

func (r *recorder) StrokeRect(x, y, w, h, strokeWidth float32, c color.RGBA) {
	r.ops = append(r.ops, []any{"sr", x, y, w, h, strokeWidth, r.colors.intern(c)})
}

func (r *recorder) StrokeLine(x1, y1, x2, y2, strokeWidth float32, c color.RGBA) {
	r.ops = append(r.ops, []any{"sl", x1, y1, x2, y2, strokeWidth, r.colors.intern(c)})
}

func (r *recorder) FillCircle(cx, cy, rad float32, c color.RGBA) {
	r.ops = append(r.ops, []any{"fc", cx, cy, rad, r.colors.intern(c)})
}

func (r *recorder) StrokeCircle(cx, cy, rad, strokeWidth float32, c color.RGBA) {
	r.ops = append(r.ops, []any{"sc", cx, cy, rad, strokeWidth, r.colors.intern(c)})
}

func (r *recorder) DrawPath(p tile.Path, c color.RGBA, fill bool, strokeWidth float32) {
	f := float32(0)
	if fill {
		f = 1
	}
	r.ops = append(r.ops, []any{"p", pathSegs(p), f, strokeWidth, r.colors.intern(c)})
}

func (r *recorder) Tick() int {
	r.usedTick = true
	return r.tick
}

func (r *recorder) GridPos() (int, int) {
	r.usedGrid = true
	return r.gx, r.gy
}

// pathSegs flattens a tile.Path into a single float32 array so the client can
// walk it with one index instead of allocating a sub-array per segment:
//
//	0, x, y            moveTo
//	1, x, y            lineTo
//	2, cx, cy, x, y    quadraticCurveTo   (control point first, matching Canvas2D)
//	3                  closePath
//
// The leading kind values are tile.PathOpKind's own iota values, so there is no
// translation table to drift.
func pathSegs(p tile.Path) []float32 {
	segs := make([]float32, 0, len(p.Ops)*3)
	for _, o := range p.Ops {
		switch o.Kind {
		case tile.OpMoveTo:
			segs = append(segs, float32(tile.OpMoveTo), o.X, o.Y)
		case tile.OpLineTo:
			segs = append(segs, float32(tile.OpLineTo), o.X, o.Y)
		case tile.OpQuadTo:
			segs = append(segs, float32(tile.OpQuadTo), o.ControlX, o.ControlY, o.X, o.Y)
		case tile.OpClose:
			segs = append(segs, float32(tile.OpClose))
		}
	}
	return segs
}

// compile-time proof that recorder satisfies the interface the drawers expect.
var _ tile.Canvas = (*recorder)(nil)
