package editorweb

import (
	"fmt"
	"math"
	"strings"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/pickup"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Marker schema derivation.
//
// The browser needs two things from internal/world/marker.go: where a marker's
// selection box sits relative to its (x, y) — needed every frame for hover and
// hit testing, so it has to be a client-side formula — and what properties a new
// marker of each type gets.
//
// Neither is retyped in JavaScript. Instead both are *probed* out of the real
// functions: call world.MarkerObjectHitRect / world.InitMarkerObject and read
// the answers back. A marker handler added in Go therefore shows up in the
// editor with no JS change at all.
//
// Probing a formula is only safe if you verify the result, so every derived hit
// rect model is checked against the real function over a dense grid of inputs
// before the server will start. See verifyHitRectModel.

// axisModel expresses one rect component as an affine function of the object:
//
//	value = K + Cx*o.X + Cy*o.Y + Cw*o.Width + Ch*o.Height
//
// Every ObjectHitRect implementation in internal/world is affine in those four
// inputs, so four probes recover the coefficients exactly.
type axisModel struct {
	K  float64 `json:"k"`
	Cx float64 `json:"x,omitempty"`
	Cy float64 `json:"y,omitempty"`
	Cw float64 `json:"w,omitempty"`
	Ch float64 `json:"h,omitempty"`
}

func (m axisModel) eval(o tiled.Object) float64 {
	return m.K + m.Cx*o.X + m.Cy*o.Y + m.Cw*o.Width + m.Ch*o.Height
}

// hitRectModel is the full client-side selection-box formula for one marker type.
type hitRectModel struct {
	X axisModel `json:"x"`
	Y axisModel `json:"y"`
	W axisModel `json:"w"`
	H axisModel `json:"h"`

	// WWhenNonPositive / HWhenNonPositive capture the degenerate-size clamp that
	// doorMarker, shrineMarker and signMarker apply ("if w <= 0 { w = 16 }").
	// Zero means the type has no clamp.
	WWhenNonPositive float64 `json:"wWhenNonPositive,omitempty"`
	HWhenNonPositive float64 `json:"hWhenNonPositive,omitempty"`
}

// eval mirrors what the JS does, and is what verifyHitRectModel checks.
func (m hitRectModel) eval(o tiled.Object) (x, y, w, h float64) {
	x, y = m.X.eval(o), m.Y.eval(o)
	if o.Width <= 0 && m.WWhenNonPositive != 0 {
		w = m.WWhenNonPositive
	} else {
		w = m.W.eval(o)
	}
	if o.Height <= 0 && m.HWhenNonPositive != 0 {
		h = m.HWhenNonPositive
	} else {
		h = m.H.eval(o)
	}
	return
}

// probeHitRect recovers the affine model for one marker type.
func probeHitRect(typ string) hitRectModel {
	// Deliberately nonzero and non-degenerate so no clamp fires while measuring
	// slopes, and each input is distinct so a transposed coefficient shows up.
	base := tiled.Object{Type: typ, X: 1000, Y: 2000, Width: 3, Height: 5}
	r0 := world.MarkerObjectHitRect(base)

	slope := func(mut func(*tiled.Object)) (dx, dy, dw, dh float64) {
		o := base
		mut(&o)
		r := world.MarkerObjectHitRect(o)
		return r.X - r0.X, r.Y - r0.Y, r.W - r0.W, r.H - r0.H
	}
	xX, xY, xW, xH := slope(func(o *tiled.Object) { o.X++ })
	yX, yY, yW, yH := slope(func(o *tiled.Object) { o.Y++ })
	wX, wY, wW, wH := slope(func(o *tiled.Object) { o.Width++ })
	hX, hY, hW, hH := slope(func(o *tiled.Object) { o.Height++ })

	m := hitRectModel{
		X: axisModel{Cx: xX, Cy: yX, Cw: wX, Ch: hX},
		Y: axisModel{Cx: xY, Cy: yY, Cw: wY, Ch: hY},
		W: axisModel{Cx: xW, Cy: yW, Cw: wW, Ch: hW},
		H: axisModel{Cx: xH, Cy: yH, Cw: wH, Ch: hH},
	}
	// Back out each constant from the base observation.
	m.X.K = r0.X - m.X.eval(base)
	m.Y.K = r0.Y - m.Y.eval(base)
	m.W.K = r0.W - m.W.eval(base)
	m.H.K = r0.H - m.H.eval(base)

	// Degenerate probe: detect the "<= 0 means 16" clamp.
	z := tiled.Object{Type: typ, X: 1000, Y: 2000, Width: 0, Height: 0}
	rz := world.MarkerObjectHitRect(z)
	if !nearlyEqual(m.W.eval(z), rz.W) {
		m.WWhenNonPositive = rz.W
	}
	if !nearlyEqual(m.H.eval(z), rz.H) {
		m.HWhenNonPositive = rz.H
	}
	return m
}

// hitRectProbeGrid is the input space verifyHitRectModel sweeps. It covers
// negatives, zero, fractions, the degenerate clamp boundary, and large values.
var (
	hitRectProbePos  = []float64{-64, -0.5, 0, 1, 16, 137.25, 4096}
	hitRectProbeSize = []float64{-1, 0, 1, 15.5, 16, 32}
)

// verifyHitRectModel re-evaluates a derived model against the real function over
// the whole probe grid. A selection box silently off by 12px on one marker type
// is exactly the failure this design must not have, so a mismatch is fatal at
// startup rather than a subtle visual bug.
func verifyHitRectModel(typ string, m hitRectModel) error {
	for _, x := range hitRectProbePos {
		for _, y := range hitRectProbePos {
			for _, w := range hitRectProbeSize {
				for _, h := range hitRectProbeSize {
					o := tiled.Object{Type: typ, X: x, Y: y, Width: w, Height: h}
					want := world.MarkerObjectHitRect(o)
					gx, gy, gw, gh := m.eval(o)
					if nearlyEqual(gx, want.X) && nearlyEqual(gy, want.Y) &&
						nearlyEqual(gw, want.W) && nearlyEqual(gh, want.H) {
						continue
					}
					return fmt.Errorf(
						"marker hit-rect model for type %q does not match world.MarkerObjectHitRect\n"+
							"  at {X:%v Y:%v W:%v H:%v}: model {%v %v %v %v}, actual {%v %v %v %v}\n"+
							"  (a MarkerHandler.ObjectHitRect is no longer affine — extend hitRectModel in internal/editorweb/markers.go)", typ, x, y, w, h, gx, gy, gw, gh, want.X, want.Y, want.W, want.H)
				}
			}
		}
	}
	return nil
}

func nearlyEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// snapProbeX / snapProbeY are fractional and off-grid on purpose: a handler that
// snaps must visibly move them onto a tile boundary.
const (
	snapProbeX = 133.5
	snapProbeY = 97.25
)

// probeTemplate returns the object world.InitMarkerObject produces for a new
// marker of this type, plus whether that handler snapped it to the tile grid.
//
// snapsToGrid is derived rather than hardcoded, so a future snapping handler is
// picked up automatically.
func probeTemplate(typ string) (tiled.Object, bool) {
	ctx := world.MarkerEditorContext{
		TileWidth:             tile.Size,
		TileHeight:            tile.Size,
		ActivePickupTiledName: pickup.All[0].TiledName(),
	}

	// doorMarker and signMarker overwrite X/Y from (wx, wy); the others leave
	// whatever the caller seeded. Seed both so either path is observable.
	probe := tiled.Object{Type: typ, X: snapProbeX, Y: snapProbeY}
	world.InitMarkerObject(&probe, snapProbeX, snapProbeY, ctx)
	moved := probe.X != snapProbeX || probe.Y != snapProbeY
	snapped := moved &&
		math.Mod(probe.X, tile.Size) == 0 &&
		math.Mod(probe.Y, tile.Size) == 0

	// The shipped template is anchored at the origin so the client can place it
	// by assigning x/y (or, for snapping types, by asking POST /api/marker).
	tmpl := tiled.Object{Type: typ}
	world.InitMarkerObject(&tmpl, 0, 0, ctx)
	return tmpl, snapped
}

// NewMarkerObject builds a marker object exactly the way the game's own editor
// hooks would, including per-type grid snapping and default properties. This is
// the authoritative path behind POST /api/marker; the schema's template exists
// only for optimistic client-side placement.
func NewMarkerObject(typ string, id int, name string, wx, wy float64, pickupKind string) (tiled.Object, error) {
	if world.MarkerHandlerFor(typ) == nil {
		return tiled.Object{}, fmt.Errorf("unknown marker type %q (known: %s)", typ, strings.Join(world.MarkerTypeNames(), ", "))
	}
	if pickupKind == "" {
		pickupKind = pickup.All[0].TiledName()
	}
	o := tiled.Object{ID: id, Name: name, Type: typ, X: wx, Y: wy}
	world.InitMarkerObject(&o, wx, wy, world.MarkerEditorContext{
		TileWidth:             tile.Size,
		TileHeight:            tile.Size,
		ActivePickupTiledName: pickupKind,
	})
	return o, nil
}

// HitRectOf returns a marker's selection box, for responses that carry one.
func HitRectOf(o tiled.Object) rect { return rectOf(world.MarkerObjectHitRect(o)) }
