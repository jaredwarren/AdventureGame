package editorweb

import (
	"fmt"
	"strconv"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// synthesizeArtFromDrawer records the current VectorDraw at GridPos (0,0) and
// converts the op list into editable Art shapes. Used when no .tile.json exists.
func synthesizeArtFromDrawer(gid int) (tile.Art, error) {
	d := tile.DefOf(gid)
	if d.Name == "unknown" {
		return tile.Art{}, fmt.Errorf("unknown gid %d", gid)
	}
	colors := newColorTable()
	r := newRecorder(colors)
	d.DrawVector(r, 0, 0, float32(tile.Size), float32(tile.Size))

	layers := make([]tile.Shape, 0, len(r.ops))
	for i, op := range r.ops {
		id := "s" + strconv.Itoa(i)
		sh, err := opToShape(id, op, colors.list)
		if err != nil {
			return tile.Art{}, fmt.Errorf("op %d: %w", i, err)
		}
		layers = append(layers, sh)
	}
	return tile.Art{
		GID:    gid,
		Name:   d.Name,
		Size:   tile.Size,
		Layers: layers,
	}, nil
}

func opToShape(id string, op []any, colors []string) (tile.Shape, error) {
	if len(op) == 0 {
		return tile.Shape{}, fmt.Errorf("empty op")
	}
	code, _ := op[0].(string)
	switch code {
	case "fr":
		return tile.Shape{
			ID: id, Type: "rect",
			X: f32(op[1]), Y: f32(op[2]), W: f32(op[3]), H: f32(op[4]),
			Fill: colorAt(colors, op[5]),
		}, nil
	case "sr":
		return tile.Shape{
			ID: id, Type: "rect",
			X: f32(op[1]), Y: f32(op[2]), W: f32(op[3]), H: f32(op[4]),
			StrokeWidth: f32(op[5]),
			Stroke:      colorAt(colors, op[6]),
		}, nil
	case "sl":
		return tile.Shape{
			ID: id, Type: "line",
			X1: f32(op[1]), Y1: f32(op[2]), X2: f32(op[3]), Y2: f32(op[4]),
			StrokeWidth: f32(op[5]),
			Stroke:      colorAt(colors, op[6]),
		}, nil
	case "fc":
		return tile.Shape{
			ID: id, Type: "circle",
			CX: f32(op[1]), CY: f32(op[2]), R: f32(op[3]),
			Fill: colorAt(colors, op[4]),
		}, nil
	case "sc":
		return tile.Shape{
			ID: id, Type: "circle",
			CX: f32(op[1]), CY: f32(op[2]), R: f32(op[3]),
			StrokeWidth: f32(op[4]),
			Stroke:      colorAt(colors, op[5]),
		}, nil
	case "p":
		segsF, _ := op[1].([]float32)
		fill01 := f32(op[2])
		sw := f32(op[3])
		col := colorAt(colors, op[4])
		filled := fill01 >= 0.5
		sh := tile.Shape{
			ID: id, Type: "path",
			StrokeWidth: sw,
			Filled:      &filled,
			Segs:        wireSegsToPath(segsF),
		}
		if filled {
			sh.Fill = col
		} else {
			sh.Stroke = col
		}
		return sh, nil
	default:
		return tile.Shape{}, fmt.Errorf("unknown opcode %q", code)
	}
}

func wireSegsToPath(segs []float32) []tile.PathSeg {
	out := make([]tile.PathSeg, 0, 8)
	for i := 0; i < len(segs); {
		switch int(segs[i]) {
		case int(tile.OpMoveTo):
			out = append(out, tile.PathSeg{Op: "M", X: segs[i+1], Y: segs[i+2]})
			i += 3
		case int(tile.OpLineTo):
			out = append(out, tile.PathSeg{Op: "L", X: segs[i+1], Y: segs[i+2]})
			i += 3
		case int(tile.OpQuadTo):
			out = append(out, tile.PathSeg{Op: "Q", CX: segs[i+1], CY: segs[i+2], X: segs[i+3], Y: segs[i+4]})
			i += 5
		case int(tile.OpClose):
			out = append(out, tile.PathSeg{Op: "Z"})
			i++
		default:
			return out
		}
	}
	return out
}

func f32(v any) float32 {
	switch n := v.(type) {
	case float32:
		return n
	case float64:
		return float32(n)
	case int:
		return float32(n)
	default:
		return 0
	}
}

func colorAt(colors []string, idx any) string {
	i := int(f32(idx))
	if i < 0 || i >= len(colors) {
		return "#000000"
	}
	return colors[i]
}
