package editorweb

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Cross-language contract test for the tile art pipeline.
//
// The editor's whole art strategy rests on one claim: the six tile.Canvas
// primitives map 1:1 onto Canvas2D, so replaying recorded ops in the browser
// reproduces the game's vector art exactly. Everything else in the pipeline is
// covered by Go tests, but the JavaScript half of that claim is not — a swapped
// argument or a wrong opcode arity would render subtly wrong tiles and no Go
// test would notice.
//
// So: derive the expected sequence of Canvas2D calls directly from the drawers
// in Go, run the real static/js/tileart.js against a recording mock context in
// Node, and require the two traces to match for every registered GID.

// canvas2dTrace implements tile.Canvas and emits the Canvas2D calls the browser
// is expected to make. It is written from the Canvas2D semantics, independently
// of the wire format, so a bug in the ops encoding shows up as a mismatch rather
// than cancelling out.
type canvas2dTrace struct {
	calls [][]any
}

func (t *canvas2dTrace) push(op string, args ...any) {
	row := append([]any{op}, args...)
	t.calls = append(t.calls, row)
}

func (t *canvas2dTrace) FillRect(x, y, w, h float32, c color.RGBA) {
	t.push("fillRect", n(x), n(y), n(w), n(h), hexColor(c))
}

func (t *canvas2dTrace) StrokeRect(x, y, w, h, sw float32, c color.RGBA) {
	t.push("strokeRect", n(x), n(y), n(w), n(h), n(sw), hexColor(c))
}

func (t *canvas2dTrace) StrokeLine(x1, y1, x2, y2, sw float32, c color.RGBA) {
	t.push("beginPath")
	t.push("moveTo", n(x1), n(y1))
	t.push("lineTo", n(x2), n(y2))
	t.push("stroke", n(sw), hexColor(c))
}

func (t *canvas2dTrace) FillCircle(cx, cy, r float32, c color.RGBA) {
	t.push("beginPath")
	t.push("arc", n(cx), n(cy), n(r))
	t.push("fill", hexColor(c), "nonzero")
}

func (t *canvas2dTrace) StrokeCircle(cx, cy, r, sw float32, c color.RGBA) {
	t.push("beginPath")
	t.push("arc", n(cx), n(cy), n(r))
	t.push("stroke", n(sw), hexColor(c))
}

func (t *canvas2dTrace) DrawPath(p tile.Path, c color.RGBA, fill bool, sw float32) {
	t.push("beginPath")
	for _, o := range p.Ops {
		switch o.Kind {
		case tile.OpMoveTo:
			t.push("moveTo", n(o.X), n(o.Y))
		case tile.OpLineTo:
			t.push("lineTo", n(o.X), n(o.Y))
		case tile.OpQuadTo:
			t.push("quadTo", n(o.ControlX), n(o.ControlY), n(o.X), n(o.Y))
		case tile.OpClose:
			t.push("closePath")
		}
	}
	if fill {
		t.push("fill", hexColor(c), "nonzero")
	} else {
		t.push("stroke", n(sw), hexColor(c))
	}
}

func (t *canvas2dTrace) Tick() int           { return 0 }
func (t *canvas2dTrace) GridPos() (int, int) { return 0, 0 }

// n matches the rounding the JS harness applies, so float32 noise cannot cause
// a spurious mismatch.
func n(v float32) float64 { return math.Round(float64(v)*1000) / 1000 }

func TestJSReplayMatchesGoDrawers(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping the JS replay contract test")
	}

	cached, err := newCachedSchema(animOpts{})
	if err != nil {
		t.Fatalf("build schema: %v", err)
	}
	schemaPath := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(schemaPath, cached.body, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(node, "jstest/replay.mjs", schemaPath).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("replay.mjs failed: %v\n%s", err, ee.Stderr)
		}
		t.Fatalf("replay.mjs failed: %v", err)
	}

	var jsTraces map[string][][]any
	if err := json.Unmarshal(out, &jsTraces); err != nil {
		t.Fatalf("decode replay output: %v", err)
	}

	for _, gid := range tile.RegisteredGIDs() {
		want := &canvas2dTrace{}
		tile.DefOf(gid).DrawVector(want, 0, 0, tile.Size, tile.Size)

		got := jsTraces[fmt.Sprint(gid)]
		if got == nil {
			t.Errorf("gid %d (%s): the JS replay produced no trace", gid, tile.DefOf(gid).Name)
			continue
		}
		compareTrace(t, gid, tile.DefOf(gid).Name, want.calls, got)
	}
}

func compareTrace(t *testing.T, gid int, name string, want, got [][]any) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("gid %d (%s): JS made %d Canvas2D calls, the drawer implies %d\n  want: %v\n  got:  %v",
			gid, name, len(got), len(want), summarize(want), summarize(got))
		return
	}
	for i := range want {
		// JSON round-trips every number as float64, so normalize before compare.
		if !reflect.DeepEqual(normalizeRow(want[i]), normalizeRow(got[i])) {
			t.Errorf("gid %d (%s): call %d differs\n  want: %v\n  got:  %v", gid, name, i, want[i], got[i])
			return
		}
	}
}

func normalizeRow(row []any) []any {
	out := make([]any, len(row))
	for i, v := range row {
		switch x := v.(type) {
		case float64:
			out[i] = math.Round(x*1000) / 1000
		case int:
			out[i] = float64(x)
		default:
			out[i] = v
		}
	}
	return out
}

func summarize(calls [][]any) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, fmt.Sprint(c[0]))
	}
	return out
}
