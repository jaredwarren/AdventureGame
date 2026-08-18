package ebitenplat

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jaredwarren/game-test/internal/services"
)

const (
	mousePointerID = -1

	// Virtual stick (bottom-left). Intentionally unlabeled.
	stickCX         = float32(34)
	stickCY         = float32(206)
	stickRadius     = float32(24)
	stickGrabRadius = float32(30)
	stickDeadzone   = float32(8)
	stickKnobRadius = float32(9)
	stickAxisSnap   = 0.38
)

type touchZone struct {
	x, y, w, h float32
	cx, cy, r  float32 // circle when r > 0
	action     services.Action
	label      string
}

type pointer struct {
	id int
	x  float32
	y  float32
}

// TouchControls maps on-screen regions to logical Actions. It supports both
// real touch input (WASM / Android) and mouse clicks for desktop testing.
type TouchControls struct {
	enabled bool
	zones   []touchZone

	curDown  map[services.Action]bool
	prevDown map[services.Action]bool

	stickID     int
	stickHeld   bool
	stickKnobDX float32
	stickKnobDY float32
}

// NewTouchControls builds the default overlay layout for 320×240.
func NewTouchControls(enabled bool) *TouchControls {
	tc := &TouchControls{
		enabled:  enabled,
		curDown:  make(map[services.Action]bool),
		prevDown: make(map[services.Action]bool),
	}
	tc.zones = defaultTouchZones()
	return tc
}

func circleBtn(cx, cy, r float32, action services.Action, label string) touchZone {
	return touchZone{cx: cx, cy: cy, r: r, action: action, label: label}
}

func rectBtn(x, y, w, h float32, action services.Action, label string) touchZone {
	return touchZone{x: x, y: y, w: w, h: h, action: action, label: label}
}

func defaultTouchZones() []touchZone {
	const btnR = float32(13)
	return []touchZone{
		// Title / menu (compact, away from stick and combat cluster).
		rectBtn(104, 222, 112, 16, services.ActionConfirm, "START"),
		rectBtn(4, 28, 40, 16, services.ActionQuickLoad, "LOAD"),

		// Combat cluster (bottom-right, circular — no overlap with stick).
		circleBtn(278, 162, btnR, services.ActionInteract, "E"),
		circleBtn(278, 188, btnR, services.ActionAttack, "ATK"),
		circleBtn(278, 214, btnR, services.ActionBomb, "BOMB"),
		circleBtn(248, 188, btnR, services.ActionDodge, "ROLL"),
		circleBtn(248, 214, btnR, services.ActionSprint, "RUN"),

		// Meta (top-right pills).
		circleBtn(302, 16, 11, services.ActionPause, "PAUSE"),
		circleBtn(276, 16, 11, services.ActionItemMenu, "ITEM"),
	}
}

// Enabled reports whether the overlay is active.
func (tc *TouchControls) Enabled() bool { return tc != nil && tc.enabled }

// BeginFrame samples touch/mouse pointers and updates edge state. Call once
// at the top of each simulation tick before scenes read input.
func (tc *TouchControls) BeginFrame() {
	if tc == nil || !tc.enabled {
		return
	}
	tc.prevDown = tc.curDown
	tc.curDown = make(map[services.Action]bool, len(tc.zones)+4)

	pts := tc.activePointers()
	stickPtr, used := tc.updateStick(pts)
	for _, pt := range pts {
		if used && pt.id == stickPtr {
			continue
		}
		for _, z := range tc.zones {
			if zoneHit(pt.x, pt.y, z) {
				tc.curDown[z.action] = true
			}
		}
	}
}

func zoneHit(px, py float32, z touchZone) bool {
	if z.r > 0 {
		dx, dy := px-z.cx, py-z.cy
		return dx*dx+dy*dy <= z.r*z.r
	}
	return pointInRect(px, py, z.x, z.y, z.w, z.h)
}

func (tc *TouchControls) updateStick(pts []pointer) (stickID int, used bool) {
	tc.stickKnobDX = 0
	tc.stickKnobDY = 0

	if tc.stickHeld {
		for _, pt := range pts {
			if pt.id == tc.stickID {
				tc.applyStickDelta(pt.x-stickCX, pt.y-stickCY)
				return tc.stickID, true
			}
		}
		tc.stickHeld = false
	}

	for _, pt := range pts {
		dx, dy := pt.x-stickCX, pt.y-stickCY
		if dx*dx+dy*dy <= stickGrabRadius*stickGrabRadius {
			tc.stickHeld = true
			tc.stickID = pt.id
			tc.applyStickDelta(dx, dy)
			return pt.id, true
		}
	}
	return 0, false
}

func (tc *TouchControls) applyStickDelta(dx, dy float32) {
	len2 := dx*dx + dy*dy
	if len2 < stickDeadzone*stickDeadzone {
		tc.stickKnobDX = 0
		tc.stickKnobDY = 0
		return
	}
	length := float32(math.Sqrt(float64(len2)))
	nx, ny := dx/length, dy/length
	if length > stickRadius {
		dx, dy = nx*stickRadius, ny*stickRadius
	}
	tc.stickKnobDX = dx
	tc.stickKnobDY = dy

	up, down, left, right := stickDirFromUnit(nx, ny)
	if up {
		tc.curDown[services.ActionMoveUp] = true
	}
	if down {
		tc.curDown[services.ActionMoveDown] = true
	}
	if left {
		tc.curDown[services.ActionMoveLeft] = true
	}
	if right {
		tc.curDown[services.ActionMoveRight] = true
	}
}

func stickDirFromUnit(nx, ny float32) (up, down, left, right bool) {
	if ny <= -stickAxisSnap {
		up = true
	}
	if ny >= stickAxisSnap {
		down = true
	}
	if nx <= -stickAxisSnap {
		left = true
	}
	if nx >= stickAxisSnap {
		right = true
	}
	return
}

func (tc *TouchControls) activePointers() []pointer {
	var pts []pointer
	ids := ebiten.AppendTouchIDs(nil)
	for _, id := range ids {
		x, y := ebiten.TouchPosition(id)
		pts = append(pts, pointer{id: int(id), x: float32(x), y: float32(y)})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pts = append(pts, pointer{id: mousePointerID, x: float32(x), y: float32(y)})
	}
	return pts
}

func pointInRect(px, py, x, y, w, h float32) bool {
	return px >= x && px < x+w && py >= y && py < y+h
}

func (tc *TouchControls) IsDown(a services.Action) bool {
	if tc == nil || !tc.enabled {
		return false
	}
	return tc.curDown[a]
}

func (tc *TouchControls) JustPressed(a services.Action) bool {
	if tc == nil || !tc.enabled {
		return false
	}
	return tc.curDown[a] && !tc.prevDown[a]
}

func (tc *TouchControls) JustReleased(a services.Action) bool {
	if tc == nil || !tc.enabled {
		return false
	}
	return !tc.curDown[a] && tc.prevDown[a]
}

// Draw renders semi-transparent button overlays in screen space.
func (tc *TouchControls) Draw(r *Renderer) {
	if tc == nil || !tc.enabled || r == nil {
		return
	}

	stickIdle := color.RGBA{12, 16, 22, 38}
	stickActive := color.RGBA{30, 42, 58, 72}
	stickStroke := color.RGBA{130, 150, 175, 70}

	btnIdle := color.RGBA{14, 18, 26, 42}
	btnPressed := color.RGBA{36, 50, 68, 88}
	btnStroke := color.RGBA{150, 170, 195, 85}
	btnLabel := color.RGBA{235, 240, 248, 190}
	btnLabelDim := color.RGBA{200, 210, 225, 140}

	tc.drawStick(r, stickIdle, stickActive, stickStroke)

	for _, z := range tc.zones {
		pressed := tc.curDown[z.action]
		if z.r > 0 {
			tc.drawCircleButton(r, z, pressed, btnIdle, btnPressed, btnStroke, btnLabel, btnLabelDim)
		} else {
			tc.drawRectButton(r, z, pressed, btnIdle, btnPressed, btnStroke, btnLabel)
		}
	}
}

func (tc *TouchControls) drawStick(r *Renderer, idle, active, stroke color.RGBA) {
	base := idle
	knob := color.RGBA{180, 195, 215, 55}
	if tc.stickHeld {
		base = active
		knob = color.RGBA{210, 225, 240, 100}
	}
	r.fillCircle(stickCX, stickCY, stickRadius, base)
	r.strokeCircle(stickCX, stickCY, stickRadius, 1, stroke)
	r.fillCircle(stickCX+tc.stickKnobDX, stickCY+tc.stickKnobDY, stickKnobRadius, knob)
	r.strokeCircle(stickCX+tc.stickKnobDX, stickCY+tc.stickKnobDY, stickKnobRadius, 0.75, stroke)
}

func (tc *TouchControls) drawCircleButton(r *Renderer, z touchZone, pressed bool, idle, active, stroke, labelClr, labelDim color.RGBA) {
	fill := idle
	lbl := labelDim
	if pressed {
		fill = active
		lbl = labelClr
	}
	r.fillCircle(z.cx, z.cy, z.r, fill)
	r.strokeCircle(z.cx, z.cy, z.r, 0.75, stroke)
	if z.label == "" {
		return
	}
	scale := float64(0.62)
	if len(z.label) > 3 {
		scale = 0.52
	}
	tw := float64(len(z.label)) * 4.5 * scale
	tx := int(z.cx - float32(tw*0.5))
	ty := int(z.cy - 3)
	r.DrawTextOpt(tx, ty, z.label, services.TextOptions{Scale: scale, Color: lbl})
}

func (tc *TouchControls) drawRectButton(r *Renderer, z touchZone, pressed bool, idle, active, stroke, labelClr color.RGBA) {
	fill := idle
	if pressed {
		fill = active
	}
	r.FillRect(z.x, z.y, z.w, z.h, fill)
	r.StrokeRect(z.x, z.y, z.w, z.h, 0.75, stroke)
	if z.label == "" {
		return
	}
	scale := 0.58
	tw := float64(len(z.label)) * 4.5 * scale
	tx := int(z.x + z.w*0.5 - float32(tw*0.5))
	ty := int(z.y + z.h*0.5 - 4)
	r.DrawTextOpt(tx, ty, z.label, services.TextOptions{Scale: scale, Color: labelClr})
}
