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

	"github.com/jaredwarren/game-test/internal/world"
)

// enemyHurtCD is how long an enemy must wait before landing another
// contact hit on the player. Shared across all enemies today; could
// migrate onto ContactHurt as a per-enemy value if we ever need it.
const enemyHurtCD = 60

// contactMargin expands the enemy hurtbox slightly so flush AABBs still register
// as contact now that movement keeps bodies from overlapping.
const contactMargin = 1.0

// EnemyAISystem homes live enemies on the player and applies contact damage.
type EnemyAISystem struct{}

func (EnemyAISystem) Update(w *world.World, bus *EventBus, _ float64) error {
	pcx, pcy := w.Player.Center()

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
		if w.IsNight() {
			aggroR *= 1.5
			speed *= 1.5
		}
		dx := pcx - ecx
		dy := pcy - ecy
		if dx*dx+dy*dy <= aggroR*aggroR {
			tgx, tgy := w.EnemyChaseTarget(ecx, ecy, pcx, pcy)
			stepEnemyTowards(w, e, tgx, tgy, speed)
		}
		if tryEnemyContactDamage(w, e) {
			tryPush(bus, PlayerHurtEvent{EnemyID: e.ID, Damage: e.Damage})
		}
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
// Returns true iff a hit landed (for event emission).
func tryEnemyContactDamage(w *world.World, e *world.Enemy) bool {
	if w.Player.Invuln > 0 || w.Player.DodgeTimer > 0 {
		return false
	}
	if e.HurtCD > 0 {
		return false
	}
	if !w.PlayerRect().OverlapsExpanded(e.Rect(), contactMargin) {
		return false
	}
	w.HP -= e.Damage
	if w.HP < 0 {
		w.HP = 0
	}
	ecx, ecy := e.Center()
	nx, ny := knockbackSlide(w, w.Player.X, w.Player.Y, w.Player.W, w.Player.H, ecx, ecy, w.Player.EffectivePlayerKnockbackForce(), true)
	w.Player.X, w.Player.Y = nx, ny
	w.Player.Invuln = w.Player.EffectiveInvulnFrames()
	e.HurtCD = enemyHurtCD
	return true
}
