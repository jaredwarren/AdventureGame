package editorweb

import (
	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Animation recording window.
//
// Only drawWater and drawLava consult Tick()/GridPos(); every other drawer is a
// pure function of (x, y, w, h). That invariant is what lets the browser build a
// single static atlas, and TestOnlyWaterAndLavaAreAnimated pins it.
//
// For the two that do animate, recording a single tick=0 frame is not good
// enough: drawLava's bubbles only grow from t=270 (lavaBubble1GrowStart) and
// t=350, so tick 0 renders lava as a flat orange square.
//
// The window is one full loop of both drawers. 1440 ticks is the LCM of every
// period involved — drawWater's 180 and 240-tick waves, drawLava's 360 and
// 480-tick bubble cycles — so frame animFrames lands exactly back on frame 0.
// TestAnimationWindowIsAWholePeriod verifies that seamlessness against the real
// drawers, so retuning a constant in draw.go fails loudly here instead of
// producing a visible hitch in the editor.
//
// The stride must not share large divisors with those periods or the sweep
// aliases: at stride 120, drawWater collapses to three repeated frames.
// gcd(8,180)=4 and gcd(8,240)=8 give 45 and 30 distinct phases respectively.
// At the game's 60 TPS a stride of 8 is a 7.5fps preview.
const (
	animStride = 8
	animFrames = 180 // animStride * animFrames = 1440 ticks
)

// animOpts controls the animated-tile frame sweep. The zero value uses the
// package defaults.
type animOpts struct {
	Stride int
	Frames int
}

func (a animOpts) withDefaults() animOpts {
	if a.Stride <= 0 {
		a.Stride = animStride
	}
	if a.Frames <= 0 {
		a.Frames = animFrames
	}
	return a
}

// animInfo tells the client how to play an animated tile's frames.
type animInfo struct {
	UsesTick    bool `json:"usesTick"`
	UsesGridPos bool `json:"usesGridPos"`
	Frames      int  `json:"frames"`
	Stride      int  `json:"stride"`
	PeriodTicks int  `json:"periodTicks"`
}

// tileArt is one tile's full description: behavior for the UI, and vector art
// for the atlas builder. Everything here is read off tile.DefOf, never retyped.
type tileArt struct {
	GID           int         `json:"gid"`
	Name          string      `json:"name"`
	Tags          []string    `json:"tags"`
	Solid         bool        `json:"solid"`
	Wall          bool        `json:"wall"`
	Water         bool        `json:"water"`
	WaterShore    bool        `json:"waterShore"`
	Floor         bool        `json:"floor"`
	Land          bool        `json:"land"`
	OpenableByKey bool        `json:"openableByKey"`
	Destructible  bool        `json:"destructible"`
	DamageKinds   []string    `json:"damageKinds"`
	DestroyedGID  int         `json:"destroyedGid"`
	FloorWeight   float64     `json:"floorWeight"`
	Swatch        string      `json:"swatch"`
	SolidRects    []rect      `json:"solidRects"`
	Surface       surfaceInfo `json:"surface"`
	Ops           [][]any     `json:"ops"`
	Anim          *animInfo   `json:"anim,omitempty"`
	Frames        [][][]any   `json:"frames,omitempty"`
}

type surfaceInfo struct {
	Type            string   `json:"type"`
	SpeedMultiplier float64  `json:"speed"`
	Friction        float64  `json:"friction"`
	HazardDamage    int      `json:"hazardDamage"`
	HazardInterval  int      `json:"hazardInterval"`
	Tags            []string `json:"tags"`
}

var damageKindNames = map[tile.DamageKind]string{
	tile.DamageBomb: "bomb",
	tile.DamageFire: "fire",
}

// recordTile drives one tile's vector drawer through a recorder and packages the
// result together with the tile's behavior flags.
func recordTile(gid int, colors *colorTable, opts animOpts) tileArt {
	opts = opts.withDefaults()
	def := tile.DefOf(gid)

	art := tileArt{
		GID:           gid,
		Name:          def.Name,
		Tags:          tagNames(def.Tags),
		Solid:         def.Solid(),
		Wall:          def.Wall(),
		Water:         def.Water(),
		WaterShore:    def.WaterShore(),
		Floor:         def.IsFloor(),
		Land:          def.IsLand(),
		OpenableByKey: def.OpenableByKey,
		Destructible:  def.Destroyable(),
		DamageKinds:   damageNames(def.DamageKinds),
		// ResolvedDestroyedGID encodes the "0 means grass" rule, so the client
		// never has to know about it.
		DestroyedGID: def.ResolvedDestroyedGID(),
		FloorWeight:  def.FloorWeight,
		Swatch:       hexColor(def.SwatchColor),
		SolidRects:   solidRects(def.SolidRects),
		Surface: surfaceInfo{
			Type:            string(def.Surface.Type),
			SpeedMultiplier: def.Surface.SpeedMultiplier,
			Friction:        def.Surface.Friction,
			HazardDamage:    def.Surface.HazardDamage,
			HazardInterval:  def.Surface.HazardInterval,
			Tags:            strSlice(def.Surface.Tags),
		},
	}

	probe := newRecorder(colors)
	def.DrawVector(probe, 0, 0, tile.Size, tile.Size)
	art.Ops = opList(probe.ops)

	if !probe.usedTick {
		return art
	}

	art.Frames = recordFrames(def, colors, opts)
	art.Anim = &animInfo{
		UsesTick:    probe.usedTick,
		UsesGridPos: probe.usedGrid,
		Frames:      len(art.Frames),
		Stride:      opts.Stride,
		PeriodTicks: len(art.Frames) * opts.Stride,
	}
	art.Ops = art.Frames[0]
	return art
}

// recordFrames samples a drawer across one full animation loop.
//
// GridPos stays pinned at (0,0). Honoring it would need one recording per map
// cell (W*H op lists) for what is only a per-tile phase offset; the client
// badges these tiles as animated instead.
func recordFrames(def tile.Tile, colors *colorTable, opts animOpts) [][][]any {
	frames := make([][][]any, opts.Frames)
	for i := range frames {
		frames[i] = recordFrameAt(def, colors, i*opts.Stride)
	}
	return frames
}

// recordFrameAt records a single frame at an absolute tick.
func recordFrameAt(def tile.Tile, colors *colorTable, tick int) [][]any {
	r := newRecorder(colors)
	r.tick = tick
	def.DrawVector(r, 0, 0, tile.Size, tile.Size)
	return opList(r.ops)
}

// recordAllTiles records every registered GID in sorted order, sharing one color
// table across all of them.
func recordAllTiles(colors *colorTable, opts animOpts) []tileArt {
	gids := tile.RegisteredGIDs()
	out := make([]tileArt, 0, len(gids))
	for _, gid := range gids {
		out = append(out, recordTile(gid, colors, opts))
	}
	return out
}

// opList normalizes a nil op slice to an empty one so the JSON is [] not null.
func opList(ops [][]any) [][]any {
	if ops == nil {
		return [][]any{}
	}
	return ops
}

func tagNames(tags []tile.TileTag) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, string(t))
	}
	return out
}

func damageNames(kinds []tile.DamageKind) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if n, ok := damageKindNames[k]; ok {
			out = append(out, n)
		}
	}
	return out
}

// rect is geom.Rect with lowercase JSON keys for the wire.
type rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

func rectOf(r geom.Rect) rect { return rect{X: r.X, Y: r.Y, W: r.W, H: r.H} }

func solidRects(rects []geom.Rect) []rect {
	out := make([]rect, 0, len(rects))
	for _, r := range rects {
		out = append(out, rectOf(r))
	}
	return out
}

func strSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// AnimOptions builds animOpts from command-line flags. Zero means "use the
// package default", which is what callers should normally pass.
func AnimOptions(stride, frames int) animOpts {
	return animOpts{Stride: stride, Frames: frames}
}
