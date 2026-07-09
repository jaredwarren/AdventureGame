// enemy_ai.go — naive homing AI + contact damage.
//
// Behavior:
//
//   - Each live enemy moves toward a chase target only while the player is
//     within AIHomer.AggroRadius (center-to-center pixels). Outside that range
//     the enemy does not path or slide toward the player.
//     Inside aggro: move at AIHomer.Speed, axis-separated against solids and the
//     player hitbox. Target is the next tile on an A* path (world.EnemyChaseTarget),
//     falling back to direct homing when pathing fails.
//   - Spawns that start inside the player are nudged apart (LegalizeEnemyOutOfPlayer).
//   - Contact at touch distance (flush edges count) deals Enemy.ContactHurt.Damage
//     when the player is neither invuln nor dodging AND the enemy's own HurtCD has
//     elapsed; both timers are refreshed on a successful hit.
//
// Enemy–enemy avoidance is still unimplemented.
package systems

import (
	"math"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world"
)

// EnemyAISystem homes live enemies on the player and applies contact damage.
type EnemyAISystem struct{}

func bossSwordHitbox(e *world.Enemy) (geom.Rect, bool) {
	if e.MeleeAttack == nil || e.MeleeAttack.Timer < 10 || e.MeleeAttack.Timer > 20 {
		return geom.Rect{}, false
	}
	// Boss center
	cx := e.X + e.W*0.5
	cy := e.Y + e.H*0.5
	reach := e.MeleeAttack.Reach
	thick := e.MeleeAttack.Thickness
	switch e.Dir {
	case world.DirDown:
		return geom.Rect{X: cx - thick*0.5, Y: cy, W: thick, H: reach}, true
	case world.DirUp:
		return geom.Rect{X: cx - thick*0.5, Y: cy - reach, W: thick, H: reach}, true
	case world.DirLeft:
		return geom.Rect{X: cx - reach, Y: cy - thick*0.5, W: reach, H: thick}, true
	default: // DirRight
		return geom.Rect{X: cx, Y: cy - thick*0.5, W: reach, H: thick}, true
	}
}

func (EnemyAISystem) Update(w *world.World, bus *EventBus, _ float64) error {
	pcx, pcy := w.Player.Center()
	bal := w.EffectiveBalance()

	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.HP <= 0 {
			continue
		}
		w.LegalizeEnemyOutOfPlayer(e)
		ecx, ecy := e.Center()
		aggroR := e.AI.AggroRadius
		if aggroR <= 0 {
			aggroR = world.DefaultEnemyAggroRadiusPx
		}
		speed := e.AI.Speed
		if e.MeleeAttack != nil && speed >= 0.55 {
			speed = 0.25
		}
		if w.IsNight() {
			aggroR *= bal.NightBuffs.AggroMultiplier
			speed *= bal.NightBuffs.SpeedMultiplier
		}
		surface := w.SurfaceAtFeet(e.X, e.Y)
		speed *= surface.SpeedMultiplier
		dx := pcx - ecx
		dy := pcy - ecy

		if e.MeleeAttack != nil {
			// Attack cooldown ticking
			if e.MeleeAttack.Cooldown > 0 {
				e.MeleeAttack.Cooldown--
			}

			// Windup state
			if e.MeleeAttack.Windup > 0 {
				e.MeleeAttack.Windup--
				if e.MeleeAttack.Windup == 0 {
					e.MeleeAttack.Timer = e.MeleeAttack.ActiveFrames
				}
				continue
			}

			// Active swing state
			if e.MeleeAttack.Timer > 0 {
				e.MeleeAttack.Timer--
				if hb, ok := bossSwordHitbox(e); ok {
					if w.Player.Invuln <= 0 && w.Player.DodgeTimer <= 0 {
						if w.PlayerRect().Overlaps(hb) {
							damage := e.MeleeAttack.Damage
							blocked := false
							if w.ShieldLevel > 0 {
								pcx := w.Player.X + w.Player.W*0.5
								pcy := w.Player.Y + w.Player.H*0.5
								ecx, ecy := e.Center()
								dx := ecx - pcx
								dy := ecy - pcy
								dist := math.Hypot(dx, dy)
								if dist > 0 {
									ndx := dx / dist
									ndy := dy / dist

									var fdx, fdy float64
									switch w.Player.Dir {
									case world.DirRight:
										fdx, fdy = 1, 0
									case world.DirLeft:
										fdx, fdy = -1, 0
									case world.DirDown:
										fdx, fdy = 0, 1
									case world.DirUp:
										fdx, fdy = 0, -1
									}

									dot := ndx*fdx + ndy*fdy
									if dot >= 0.5 {
										blocked = true
										percent := w.Player.PlayerTuning.ShieldL1BlockPercent
										if w.ShieldLevel >= 2 {
											percent = w.Player.PlayerTuning.ShieldL2BlockPercent
										}
										reduced := int(float64(damage) * percent)
										damage = damage - reduced
										if damage < 1 {
											damage = 1
										}
									}
								}
							}
							w.HP -= damage
							if w.HP < 0 {
								w.HP = 0
							}
							ecx, ecy := e.Center()
							nx, ny := knockbackSlide(w, w.Player.X, w.Player.Y, w.Player.W, w.Player.H, ecx, ecy, w.Player.EffectivePlayerKnockbackForce()*2.0, true)
							w.Player.X, w.Player.Y = nx, ny
							w.Player.Invuln = w.Player.EffectiveInvulnFrames()
							tryPush(bus, PlayerHurtEvent{EnemyID: e.ID, Damage: damage, Blocked: blocked})
						}
					}
				}
				continue
			}

			// Check trigger sword swing range
			dist := math.Hypot(dx, dy)
			if dist <= e.MeleeAttack.Reach+4.0 && e.MeleeAttack.Cooldown == 0 {
				if math.Abs(dx) > math.Abs(dy) {
					if dx > 0 {
						e.Dir = world.DirRight
					} else {
						e.Dir = world.DirLeft
					}
				} else {
					if dy > 0 {
						e.Dir = world.DirDown
					} else {
						e.Dir = world.DirUp
					}
				}
				e.MeleeAttack.Windup = e.MeleeAttack.WindupFrames
				e.MeleeAttack.Cooldown = e.MeleeAttack.CooldownFrames
				continue
			}

			// Normal movement for boss
			if dx*dx+dy*dy <= aggroR*aggroR {
				tgx, tgy := w.EnemyChaseTarget(ecx, ecy, pcx, pcy)
				stepEnemyTowards(w, e, tgx, tgy, speed)
			}
			tryEnemyContactDamage(w, bus, e)
			continue
		}

		if dx*dx+dy*dy <= aggroR*aggroR {
			tgx, tgy := w.EnemyChaseTarget(ecx, ecy, pcx, pcy)
			stepEnemyTowards(w, e, tgx, tgy, speed)
		}
		tryEnemyContactDamage(w, bus, e)
	}
	return nil
}

// stepEnemyTowards moves e one tick toward (tx, ty), axis-sliding on walls.
// Hitbox size is read from the enemy's Hitbox component; no magic numbers.
func stepEnemyTowards(w *world.World, e *world.Enemy, tx, ty float64, speed float64) {
	ecx, ecy := e.Center()
	dx := tx - ecx
	dy := ty - ecy
	dlen := math.Hypot(dx, dy)
	if dlen > 0.5 {
		dx = dx / dlen * speed
		dy = dy / dlen * speed

		// Update facing direction based on movement vector
		if math.Abs(dx) > math.Abs(dy) {
			if dx > 0 {
				e.Dir = world.DirRight
			} else {
				e.Dir = world.DirLeft
			}
		} else {
			if dy > 0 {
				e.Dir = world.DirDown
			} else {
				e.Dir = world.DirUp
			}
		}
	} else {
		dx, dy = 0, 0
	}
	nx, ny := w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, dx, dy)
	e.X, e.Y = nx, ny
}

// tryEnemyContactDamage applies contact damage if eligibility checks pass.
// Returns true iff a hit landed.
func tryEnemyContactDamage(w *world.World, bus *EventBus, e *world.Enemy) bool {
	if w.Player.Invuln > 0 || w.Player.DodgeTimer > 0 {
		return false
	}
	if e.HurtCD > 0 {
		return false
	}
	contactCfg := w.EffectiveBalance().Contact
	if !w.PlayerRect().OverlapsExpanded(e.Rect(), contactCfg.ContactMargin) {
		return false
	}

	damage := e.Damage
	blocked := false
	if w.ShieldLevel > 0 {
		pcx := w.Player.X + w.Player.W*0.5
		pcy := w.Player.Y + w.Player.H*0.5
		ecx, ecy := e.Center()
		dx := ecx - pcx
		dy := ecy - pcy
		dist := math.Hypot(dx, dy)
		if dist > 0 {
			ndx := dx / dist
			ndy := dy / dist

			var fdx, fdy float64
			switch w.Player.Dir {
			case world.DirRight:
				fdx, fdy = 1, 0
			case world.DirLeft:
				fdx, fdy = -1, 0
			case world.DirDown:
				fdx, fdy = 0, 1
			case world.DirUp:
				fdx, fdy = 0, -1
			}

			dot := ndx*fdx + ndy*fdy
			if dot >= 0.5 {
				blocked = true
				percent := w.Player.PlayerTuning.ShieldL1BlockPercent
				if w.ShieldLevel >= 2 {
					percent = w.Player.PlayerTuning.ShieldL2BlockPercent
				}
				reduced := int(float64(damage) * percent)
				damage = damage - reduced
				if damage < 1 {
					damage = 1
				}
			}
		}
	}

	w.HP -= damage
	if w.HP < 0 {
		w.HP = 0
	}
	ecx, ecy := e.Center()
	nx, ny := knockbackSlide(w, w.Player.X, w.Player.Y, w.Player.W, w.Player.H, ecx, ecy, w.Player.EffectivePlayerKnockbackForce(), true)
	w.Player.X, w.Player.Y = nx, ny
	w.Player.Invuln = w.Player.EffectiveInvulnFrames()
	e.HurtCD = contactCfg.EnemyHurtCD

	tryPush(bus, PlayerHurtEvent{EnemyID: e.ID, Damage: damage, Blocked: blocked})
	return true
}
