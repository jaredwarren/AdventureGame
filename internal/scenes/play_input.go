package scenes

import (
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

func (s *PlayScene) handleActions(ctx GameContext, w *world.World) {
	in := ctx.Input()
	pr := w.PlayerRect()
	if in.JustPressed(services.ActionInteract) {
		if !w.TryOpenChest() {
			for _, sh := range w.Shrines {
				if pr.Overlaps(sh.Rect) {
					ctx.Manager().PushOverlay(SceneShop, nil)
					break
				}
			}
		}
	}

	if in.JustPressed(services.ActionAttack) {
		if w.TrySwingSword() {
			ctx.Audio().Play("swing.wav", 0.2)
		}
	}
	if in.JustPressed(services.ActionBomb) {
		switch w.SelectedItem {
		case world.ItemSlotBomb:
			s.tryDropBomb(ctx, w)
		case world.ItemSlotTorch:
			s.trySwingTorch(ctx, w)
		}
	}

	cost := w.Player.EffectiveDodgeStaminaCost()
	if in.JustPressed(services.ActionDodge) && w.Player.Stamina >= cost {
		w.Player.Stamina -= cost
		w.Player.DodgeTimer = w.Player.EffectiveDodgeDuration()
		s.dodgeImpulse = w.Player.EffectiveDodgeMaxImpulse()
	}
}

func (s *PlayScene) tryDropBomb(ctx GameContext, w *world.World) {
	if w.TryPlaceBomb() {
		ctx.Audio().Play("swing.wav", 0.15)
	}
}

func (s *PlayScene) trySwingTorch(ctx GameContext, w *world.World) {
	if !w.HasTorch {
		return
	}
	if w.TrySwingTorch() {
		ctx.Audio().Play("swing.wav", 0.22)
		pc := w.PlayerRect()
		cx := pc.X + pc.W*0.5
		cy := pc.Y + pc.H*0.5
		for k := 0; k < 5; k++ {
			s.particles = append(s.particles, NewEmberParticle(cx, cy))
		}
	}
}

func (s *PlayScene) handleItemMenu(ctx GameContext, w *world.World) {
	in := ctx.Input()

	// Build ordered slot list based on what player owns.
	slots := []world.ItemSlot{world.ItemSlotBomb}
	if w.HasTorch {
		slots = append(slots, world.ItemSlotTorch)
	}

	// Clamp cursor to valid range.
	if s.itemMenuCursor >= len(slots) {
		s.itemMenuCursor = len(slots) - 1
	}

	if in.JustPressed(services.ActionMoveUp) && s.itemMenuCursor > 0 {
		s.itemMenuCursor--
	}
	if in.JustPressed(services.ActionMoveDown) && s.itemMenuCursor < len(slots)-1 {
		s.itemMenuCursor++
	}
	if in.JustPressed(services.ActionConfirm) {
		w.SelectedItem = slots[s.itemMenuCursor]
		s.showItemMenu = false
	}
	if in.JustPressed(services.ActionCancel) || in.JustPressed(services.ActionItemMenu) {
		s.showItemMenu = false
	}
}

// handleMovement reads directional input, applies sprint/stamina drain,
// dodge impulse, and moves the player. Stamina regen runs here too because
// it's the natural inverse of drain; a future StaminaSystem could migrate
// both once Player.Sprinting is modeled explicitly.
func (s *PlayScene) handleMovement(ctx GameContext, w *world.World) {
	in := ctx.Input()
	ax, ay := in.Axis2D()
	isMoving := ax != 0 || ay != 0
	isSprintHeld := in.IsDown(services.ActionSprint)

	w.Player.SprintHeld = isSprintHeld
	w.Player.IsMoving = isMoving

	spd := w.Player.EffectiveBaseSpeed()
	if w.Player.IsSprinting {
		spd = w.Player.EffectiveSprintSpeed()
	}

	dx := float64(ax) * spd
	dy := float64(ay) * spd
	if dx != 0 || dy != 0 {
		w.SetFacingFromMotion(dx, dy)
	}

	if s.dodgeImpulse > 0 {
		s.dodgeImpulse--
		dodgeSpd := w.Player.EffectiveDodgeSpeed()
		switch w.Player.Dir {
		case world.DirDown:
			dy += dodgeSpd
		case world.DirUp:
			dy -= dodgeSpd
		case world.DirLeft:
			dx -= dodgeSpd
		case world.DirRight:
			dx += dodgeSpd
		}
	}

	w.TryMove(dx, dy)
}
