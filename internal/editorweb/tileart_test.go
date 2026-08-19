package editorweb

import (
	"reflect"

	"encoding/json"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// TestOnlyWaterAndLavaAreAnimated pins the invariant the entire art-export
// design rests on: every tile drawer except water and lava is a pure function of
// (x, y, w, h), so a single static atlas frame is a faithful rendering of it.
//
// If this fails, someone added Tick()/GridPos() to a drawer. Either add the GID
// below (and confirm the animation window in tileart.go still covers its
// period), or the editor will silently render that tile with a stale frame.
func TestOnlyWaterAndLavaAreAnimated(t *testing.T) {
	want := map[int]bool{tile.GIDWater: true, tile.GIDLava: true}

	colors := newColorTable()
	for _, gid := range tile.RegisteredGIDs() {
		r := newRecorder(colors)
		tile.DefOf(gid).DrawVector(r, 0, 0, tile.Size, tile.Size)
		animated := r.usedTick || r.usedGrid
		if animated != want[gid] {
			t.Errorf("gid %d (%s): animated=%v, want %v", gid, tile.DefOf(gid).Name, animated, want[gid])
		}
	}
}

// TestRecordedTilesAreWellFormed checks every drawer produces usable art and
// stays inside its own cell. A drawer that bleeds past its 16x16 box would paint
// over its neighbors in the atlas strip, which is a subtle, ugly failure.
func TestRecordedTilesAreWellFormed(t *testing.T) {
	colors := newColorTable()
	for _, art := range recordAllTiles(colors, animOpts{}) {
		if len(art.Ops) == 0 {
			t.Errorf("gid %d (%s): recorded no draw ops", art.GID, art.Name)
			continue
		}
		checkOps(t, art, art.Ops)
		for i, frame := range art.Frames {
			if len(frame) == 0 {
				t.Errorf("gid %d (%s): frame %d is empty", art.GID, art.Name, i)
			}
			checkOps(t, art, frame)
		}
	}
}

// checkOps validates opcode arity and coordinate sanity for one op list.
func checkOps(t *testing.T, art tileArt, ops [][]any) {
	t.Helper()
	// Slack on both sides: drawers legitimately use strokeRect(x+0.5, ...) and
	// stroke widths straddle the path, so a stroke at x=0 covers x=-0.5.
	const lo, hi = -2.0, 18.0

	arity := map[string]int{"fr": 6, "sr": 7, "sl": 7, "fc": 5, "sc": 6, "p": 5}

	for _, op := range ops {
		if len(op) == 0 {
			t.Fatalf("gid %d: empty op", art.GID)
		}
		code, ok := op[0].(string)
		if !ok {
			t.Fatalf("gid %d: op code is %T, want string", art.GID, op[0])
		}
		want, known := arity[code]
		if !known {
			t.Errorf("gid %d (%s): unknown opcode %q — add it to the wire format doc in canvasrec.go and to the JS replay", art.GID, art.Name, code)
			continue
		}
		if len(op) != want {
			t.Errorf("gid %d (%s): op %q has %d elements, want %d", art.GID, art.Name, code, len(op), want)
			continue
		}
		// Color index is always last.
		if ci, ok := op[len(op)-1].(int); !ok || ci < 0 {
			t.Errorf("gid %d: op %q color index is %v (%T), want a non-negative int", art.GID, code, op[len(op)-1], op[len(op)-1])
		}

		if code == "p" {
			segs, ok := op[1].([]float32)
			if !ok {
				t.Errorf("gid %d: path segs are %T, want []float32", art.GID, op[1])
				continue
			}
			checkCoords(t, art, code, segs, lo, hi)
			continue
		}
		for _, v := range op[1 : len(op)-1] {
			f, ok := v.(float32)
			if !ok {
				t.Errorf("gid %d: op %q has non-float32 arg %v (%T)", art.GID, code, v, v)
				continue
			}
			checkCoords(t, art, code, []float32{f}, lo, hi)
		}
	}
}

func checkCoords(t *testing.T, art tileArt, code string, vals []float32, lo, hi float64) {
	t.Helper()
	for _, v := range vals {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Errorf("gid %d (%s): op %q produced non-finite value %v", art.GID, art.Name, code, v)
			continue
		}
		if f < lo || f > hi {
			t.Errorf("gid %d (%s): op %q coordinate %v escapes the tile cell [%v,%v] and will bleed into neighbors in the atlas", art.GID, art.Name, code, v, lo, hi)
		}
	}
}

// TestAllTileColorsAreOpaque documents that hexColor's un-premultiply branch is
// currently unreachable. If this fails, a translucent tile color was added:
// verify it renders correctly in the browser, then relax the assertion.
func TestAllTileColorsAreOpaque(t *testing.T) {
	colors := newColorTable()
	arts := recordAllTiles(colors, animOpts{})

	for _, art := range arts {
		if len(art.Swatch) != 7 {
			t.Errorf("gid %d (%s): swatch %q is not opaque #rrggbb", art.GID, art.Name, art.Swatch)
		}
	}
	for i, c := range colors.list {
		if len(c) != 7 {
			t.Errorf("color table entry %d = %q is not opaque #rrggbb", i, c)
		}
	}
}

func TestHexColorUnPremultiplies(t *testing.T) {
	tests := []struct {
		name string
		in   color.RGBA
		want string
	}{
		{"opaque", color.RGBA{0x55, 0x88, 0x55, 0xff}, "#558855"},
		{"transparent", color.RGBA{0, 0, 0, 0}, "#00000000"},
		// Premultiplied half-alpha white: 0x80 over 0x80 alpha is full white.
		{"half alpha white", color.RGBA{0x80, 0x80, 0x80, 0x80}, "#ffffff80"},
	}
	for _, tc := range tests {
		if got := hexColor(tc.in); got != tc.want {
			t.Errorf("%s: hexColor(%v) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// TestTileArtMatchesRegistry proves the payload is read off tile.DefOf rather
// than retyped, for every registered GID.
func TestTileArtMatchesRegistry(t *testing.T) {
	arts := recordAllTiles(newColorTable(), animOpts{})
	if len(arts) != len(tile.RegisteredGIDs()) {
		t.Fatalf("recorded %d tiles, registry has %d", len(arts), len(tile.RegisteredGIDs()))
	}
	for _, art := range arts {
		def := tile.DefOf(art.GID)
		if def.Name == "unknown" {
			t.Errorf("gid %d is not in the registry but was recorded", art.GID)
		}
		if art.Name != def.Name {
			t.Errorf("gid %d: name %q != registry %q", art.GID, art.Name, def.Name)
		}
		if art.Solid != def.Solid() || art.Wall != def.Wall() || art.Water != def.Water() {
			t.Errorf("gid %d (%s): behavior flags do not match the registry", art.GID, art.Name)
		}
		if art.Floor != def.IsFloor() {
			t.Errorf("gid %d (%s): floor flag does not match the registry", art.GID, art.Name)
		}
		if art.DestroyedGID != def.ResolvedDestroyedGID() {
			t.Errorf("gid %d (%s): destroyedGid %d != ResolvedDestroyedGID %d", art.GID, art.Name, art.DestroyedGID, def.ResolvedDestroyedGID())
		}
		if art.Swatch != hexColor(def.SwatchColor) {
			t.Errorf("gid %d (%s): swatch %q != registry %q", art.GID, art.Name, art.Swatch, hexColor(def.SwatchColor))
		}
		if len(art.SolidRects) != len(def.SolidRects) {
			t.Errorf("gid %d (%s): %d solid rects, registry has %d", art.GID, art.Name, len(art.SolidRects), len(def.SolidRects))
		}
	}
}

// TestAnimatedTilesSweepTheirPeriod guards the specific bug a naive tick=0
// recording would ship: drawLava's bubbles only appear from t=270, so a single
// tick-0 frame renders lava as a flat orange square. It also guards the aliasing
// trap on the other side — a stride sharing a divisor with a drawer's period
// collapses the sweep to a handful of repeated frames.
func TestAnimatedTilesSweepTheirPeriod(t *testing.T) {
	byGID := map[int]tileArt{}
	for _, a := range recordAllTiles(newColorTable(), animOpts{}) {
		byGID[a.GID] = a
	}

	lava := byGID[tile.GIDLava]
	if lava.Anim == nil {
		t.Fatal("lava has no animation info")
	}
	// The whole point: frame 0 has no bubble, but the sweep must find one.
	if hasCircle(lava.Frames[0]) {
		t.Error("lava frame 0 unexpectedly has a bubble; the sweep assumption changed")
	}
	if !anyFrameHasCircle(lava.Frames) {
		t.Errorf("no lava frame contains a bubble across %d frames of stride %d — the recording window no longer covers the bubble cycles", len(lava.Frames), lava.Anim.Stride)
	}

	water := byGID[tile.GIDWater]
	if water.Anim == nil {
		t.Fatal("water has no animation info")
	}

	// Every animated tile must yield a well-distributed sweep. A stride that
	// aliases against a drawer's period collapses this to a handful of phases
	// (stride 120 against drawWater's 180-tick wave gives exactly three).
	for _, art := range []tileArt{water, lava} {
		if got := len(art.Frames); got != art.Anim.Frames {
			t.Errorf("%s: %d frames recorded, anim reports %d", art.Name, got, art.Anim.Frames)
		}
		if art.Anim.PeriodTicks != art.Anim.Frames*art.Anim.Stride {
			t.Errorf("%s: periodTicks %d != frames*stride", art.Name, art.Anim.PeriodTicks)
		}
		distinct := map[string]bool{}
		for _, f := range art.Frames {
			b, err := json.Marshal(f)
			if err != nil {
				t.Fatalf("marshal frame: %v", err)
			}
			distinct[string(b)] = true
		}
		// Lava legitimately repeats during its bubble dead zones, so this is a
		// floor rather than an equality. It still catches gross aliasing.
		if len(distinct) < 10 {
			t.Errorf("%s: %d frames but only %d distinct — stride %d aliases against this drawer's period", art.Name, len(art.Frames), len(distinct), art.Anim.Stride)
		}
	}
}

// TestAnimationWindowIsAWholePeriod verifies the recorded loop is seamless: the
// frame one step past the end must be identical to frame 0. If a wave or bubble
// constant in internal/world/tile/draw.go changes so that 1440 ticks is no
// longer a whole number of periods, this fails instead of the editor showing a
// visible hitch every time the loop wraps.
func TestAnimationWindowIsAWholePeriod(t *testing.T) {
	opts := animOpts{}.withDefaults()
	colors := newColorTable()

	for _, art := range recordAllTiles(colors, animOpts{}) {
		if art.Anim == nil {
			continue
		}
		def := tile.DefOf(art.GID)
		wrapped := recordFrameAt(def, colors, opts.Frames*opts.Stride)
		if !reflect.DeepEqual(wrapped, art.Frames[0]) {
			t.Errorf("%s (gid %d): tick %d does not match tick 0, so the %d-tick window is not a whole period — retune animFrames/animStride in tileart.go to the LCM of this drawer's cycles",
				art.Name, art.GID, opts.Frames*opts.Stride, art.Anim.PeriodTicks)
		}
	}
}

func anyFrameHasCircle(frames [][][]any) bool {
	for _, f := range frames {
		if hasCircle(f) {
			return true
		}
	}
	return false
}

func hasCircle(ops [][]any) bool {
	for _, op := range ops {
		if code, ok := op[0].(string); ok && (code == "fc" || code == "sc") {
			return true
		}
	}
	return false
}

// TestTileArtJSONIsStable guards against a map[...] sneaking into the payload,
// which would make the schema bytes nondeterministic and break ETag caching.
func TestTileArtJSONIsStable(t *testing.T) {
	first, err := json.Marshal(recordAllTiles(newColorTable(), animOpts{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	second, err := json.Marshal(recordAllTiles(newColorTable(), animOpts{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(first) != string(second) {
		t.Error("tile art JSON is not deterministic across builds")
	}
	// float32 must not widen: 16*0.3 should serialize compactly.
	if strings.Contains(string(first), "0000000") {
		t.Error("float64 widening detected in the payload — keep recorder values as float32 so encoding/json emits the short form")
	}
}
