package world

import (
	"math"

	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/geom"
)

// Player holds locomotion + combat timers (frame counts, not seconds).
// Player-only state (Swing, Dodge, Stamina) stays inline rather than as
// one-off components; those concerns aren't shared with other entities.
// SprintHeld reserves a future "toggle sprint" UX; today stamina drain
// is keyed off Shift in scenes.PlayScene.
type Player struct {
	ID EntityID
	Transform
	Hitbox
	Facing

	Swing                  int // >0 while sword animation runs; only mid-window counts as hit frames (see SwordHitbox)
	SwingCD                int // frames until another Z press is accepted
	TorchSwing             int // >0 while torch swing runs; same active window as sword (see TorchHitbox)
	TorchSwingCD           int // frames until another torch press is accepted
	Invuln                 int // i-frames after taking damage
	DodgeTimer             int // if >0, enemy contact check skipped (paired with scene dodgeImpulse nudge)
	Stamina                int // drained by sprint; refilled when not sprinting
	SprintHeld             bool
	SprintExhausted        bool
	IsMoving               bool
	IsSprinting            bool
	WasOnQuicksand         bool
	RunningAcrossQuicksand bool
	VX                     float64
	VY                     float64

	balance.PlayerTuning
}

// Rect returns the player's AABB.
func (p Player) Rect() geom.Rect { return geom.Rect{X: p.X, Y: p.Y, W: p.W, H: p.H} }

// Center returns the player's AABB center point.
func (p Player) Center() (float64, float64) { return p.X + p.W*0.5, p.Y + p.H*0.5 }

func (w *World) TryMove(dx, dy float64) {
	nx, ny := w.SlidePlayerAABB(w.Player.X, w.Player.Y, w.Player.W, w.Player.H, dx, dy)
	w.Player.X, w.Player.Y = nx, ny
}

func (w *World) SetFacingFromMotion(dx, dy float64) {
	if math.Abs(dx) > math.Abs(dy) {
		if dx > 0 {
			w.Player.Dir = DirRight
		} else if dx < 0 {
			w.Player.Dir = DirLeft
		}
	} else {
		if dy > 0 {
			w.Player.Dir = DirDown
		} else if dy < 0 {
			w.Player.Dir = DirUp
		}
	}
}

func swordSwingHitbox(swing int, activeStart, activeEnd int, reach, thick float64, dir Dir, px, py float64) (geom.Rect, bool) {
	if swing <= 0 {
		return geom.Rect{}, false
	}
	active := swing >= activeStart && swing <= activeEnd
	if !active {
		return geom.Rect{}, false
	}
	switch dir {
	case DirDown:
		return geom.Rect{X: px - thick*0.5, Y: py, W: thick, H: reach}, true
	case DirUp:
		return geom.Rect{X: px - thick*0.5, Y: py - reach, W: thick, H: reach}, true
	case DirLeft:
		return geom.Rect{X: px - reach, Y: py - thick*0.5, W: reach, H: thick}, true
	default: // DirRight
		return geom.Rect{X: px, Y: py - thick*0.5, W: reach, H: thick}, true
	}
}

// SwordHitbox returns the sword arc AABB during the active swing frames.
func (w *World) SwordHitbox() (geom.Rect, bool) {
	px := w.Player.X + w.Player.W*0.5
	py := w.Player.Y + w.Player.H*0.5
	return swordSwingHitbox(
		w.Player.Swing,
		w.Player.EffectiveSwingActiveStart(),
		w.Player.EffectiveSwingActiveEnd(),
		w.Player.EffectiveSwordReach(),
		w.Player.EffectiveSwordThickness(),
		w.Player.Dir, px, py,
	)
}

// TorchHitbox matches SwordHitbox timing/geometry but keys off TorchSwing.
func (w *World) TorchHitbox() (geom.Rect, bool) {
	px := w.Player.X + w.Player.W*0.5
	py := w.Player.Y + w.Player.H*0.5
	return swordSwingHitbox(
		w.Player.TorchSwing,
		w.Player.EffectiveTorchSwingActiveStart(),
		w.Player.EffectiveTorchSwingActiveEnd(),
		w.Player.EffectiveTorchReach(),
		w.Player.EffectiveTorchThickness(),
		w.Player.Dir, px, py,
	)
}

// TrySwingSword starts a sword swing if not already swinging / on cooldown.
func (w *World) TrySwingSword() bool {
	if w.Player.Swing > 0 || w.Player.SwingCD > 0 {
		return false
	}
	if w.Player.TorchSwing > 0 || w.Player.TorchSwingCD > 0 {
		return false
	}
	w.Player.Swing = w.Player.EffectiveSwingDuration()
	w.Player.SwingCD = w.Player.EffectiveMaxSwingCD()
	return true
}

// TrySwingTorch starts a torch swing if the player has a torch and no
// sword or torch animation is already running.
func (w *World) TrySwingTorch() bool {
	if !w.HasItem("torch") {
		return false
	}
	if w.Player.TorchSwing > 0 || w.Player.TorchSwingCD > 0 {
		return false
	}
	if w.Player.Swing > 0 || w.Player.SwingCD > 0 {
		return false
	}
	w.Player.TorchSwing = w.Player.EffectiveTorchSwingDuration()
	w.Player.TorchSwingCD = w.Player.EffectiveMaxTorchSwingCD()
	return true
}
