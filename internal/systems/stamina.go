package systems

import "github.com/jaredwarren/game-test/internal/world"

// StaminaSystem updates player stamina drain while sprinting and stamina recovery.
type StaminaSystem struct{}

func (StaminaSystem) Update(w *world.World, bus *EventBus, _ float64) error {
	p := &w.Player
	if !p.SprintHeld {
		p.SprintExhausted = false
	}

	canSprint := p.SprintHeld && p.IsMoving && !p.SprintExhausted && p.Stamina > 0
	if canSprint {
		p.IsSprinting = true
		p.Stamina--
		if p.Stamina <= 0 {
			p.Stamina = 0
			p.SprintExhausted = true
			p.IsSprinting = false
		}
	} else {
		p.IsSprinting = false
		if p.Stamina < w.MaxStamina() {
			if w.Tick%p.EffectiveStaminaRegenInterval() == 0 {
				p.Stamina++
			}
		}
	}
	return nil
}
