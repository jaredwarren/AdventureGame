package world

import (
	"math"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world/enemy"
)

const (
	defaultEnemyW = enemy.HitboxW
	defaultEnemyH = enemy.HitboxH
)

// Enemy is one combat unit with simple homing AI. Components:
//
//   - Transform / Hitbox — position + AABB
//   - Health            — HP + MaxHP
//   - AI                — seek speed
//   - ContactHurt       — on-touch damage + cooldown
//
// IsBoss flags the run's boss (used for the kill bonus); the field could
// migrate to its own marker component once we have more than one role.
type Enemy struct {
	ID EntityID
	Transform
	Hitbox
	Health
	AI AIHomer
	ContactHurt

	// BurnTimer counts down while the enemy is ignited (torch DoT).
	// BurnCD counts down to the next burn tick while BurnTimer > 0.
	BurnTimer int
	BurnCD    int

	Dir    Dir // facing direction for rendering
	IsBoss bool

	Armor       *Armor
	MeleeAttack *MeleeAttack
}

// Armor allows an entity to deflect damage until the armor is broken.
type Armor struct {
	Health    int
	MaxHealth int
}

// MeleeAttack allows an entity to perform telegraphed swing attacks.
type MeleeAttack struct {
	Damage         int
	Reach          float64
	Thickness      float64
	WindupFrames   int
	ActiveFrames   int
	CooldownFrames int

	Windup   int
	Timer    int
	Cooldown int
}

// Rect returns the enemy's AABB using its Transform + Hitbox components.
func (e Enemy) Rect() geom.Rect { return geom.Rect{X: e.X, Y: e.Y, W: e.W, H: e.H} }

// Center returns the enemy's AABB center point.
func (e Enemy) Center() (float64, float64) { return e.X + e.W*0.5, e.Y + e.H*0.5 }

// DamageArmor decrements armor health and returns true if the armor broke this frame.
func (e *Enemy) DamageArmor(amount int) bool {
	if e.Armor == nil || e.Armor.Health <= 0 {
		return false
	}
	e.Armor.Health -= amount
	if e.Armor.Health < 0 {
		e.Armor.Health = 0
	}
	return e.Armor.Health == 0
}

// DamageEnemy applies dmg to the enemy at slice index i. Zero-clamped.
// No-op when the index is out of range or the enemy is already dead.
func (w *World) DamageEnemy(i int, dmg int) {
	if i < 0 || i >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[i]
	if e.HP <= 0 {
		return
	}
	e.HP -= dmg
	if e.HP < 0 {
		e.HP = 0
	}
}

// EnemyRect returns the AABB of the enemy at slice index i.
func (w *World) EnemyRect(i int) geom.Rect { return w.Enemies[i].Rect() }

// IgniteEnemy extends or starts torch burn on the enemy at index i.
func (w *World) IgniteEnemy(i int) {
	if i < 0 || i >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[i]
	if e.HP <= 0 {
		return
	}
	burnDuration := w.Player.EffectiveTorchBurnDuration()
	burnInterval := w.Player.EffectiveTorchBurnInterval()
	if e.BurnTimer < burnDuration {
		e.BurnTimer = burnDuration
	}
	if e.BurnCD <= 0 {
		e.BurnCD = burnInterval
	}
}

// LegalizeEnemyOutOfPlayer nudges a live enemy that overlaps the player until
// separated or attempts are exhausted (spawn overlap, knockback quirks).
func (w *World) LegalizeEnemyOutOfPlayer(e *Enemy) {
	if e.HP <= 0 {
		return
	}
	pr := w.PlayerRect()
	pcx, pcy := w.Player.Center()
	for iter := 0; iter < 32; iter++ {
		if !e.Rect().Overlaps(pr) {
			return
		}
		tcx := e.X + e.W*0.5
		tcy := e.Y + e.H*0.5
		dx := tcx - pcx
		dy := tcy - pcy
		d := math.Hypot(dx, dy)
		const step = 2.5
		var nx, ny float64
		if d < 1e-6 {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, step, 0)
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, step)
			}
		} else {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, dx/d*step, dy/d*step)
		}
		if nx == e.X && ny == e.Y {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, step, 0)
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, -step, 0)
			}
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, step)
			}
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, -step)
			}
		}
		e.X, e.Y = nx, ny
	}
}

// NewEnemy is the pure factory for Enemy values. hp <= 0 uses defaultEnemyHP.
func NewEnemy(id EntityID, x, y float64, hp int, isBoss bool) Enemy {
	cfg := DefaultEnemyConfig()
	if hp > 0 {
		cfg.HP = hp
	}
	cfg.IsBoss = isBoss
	return NewEnemyWithConfig(id, x, y, cfg)
}

// NewEnemyWithConfig builds an enemy from a fully resolved EnemyConfig.
func NewEnemyWithConfig(id EntityID, x, y float64, cfg EnemyConfig) Enemy {
	e := Enemy{
		ID:        id,
		Transform: Transform{X: x, Y: y},
		Hitbox:    Hitbox{W: defaultEnemyW, H: defaultEnemyH},
		Health:    Health{HP: cfg.HP, MaxHP: cfg.HP},
		AI: AIHomer{
			Speed:       cfg.Speed,
			AggroRadius: cfg.AggroRadius,
		},
		ContactHurt: ContactHurt{
			Damage: cfg.ContactDamage,
		},
		IsBoss: cfg.IsBoss,
	}
	if cfg.IsArmoredKnight {
		e.Armor = &Armor{
			Health:    cfg.ArmorHealth,
			MaxHealth: cfg.ArmorHealth,
		}
		e.MeleeAttack = &MeleeAttack{
			Damage:         4,
			Reach:          24,
			Thickness:      24,
			WindupFrames:   30,
			ActiveFrames:   30,
			CooldownFrames: 90,
		}
	}
	return e
}

// SpawnEnemy allocates an ID, builds the Enemy, appends it, and returns
// the freshly-assigned ID for callers that want to remember it.
func (w *World) SpawnEnemy(x, y float64, hp int, isBoss bool) EntityID {
	cfg := DefaultEnemyConfig()
	if hp > 0 {
		cfg.HP = hp
	}
	cfg.IsBoss = isBoss
	return w.SpawnEnemyConfig(x, y, cfg)
}

// SpawnEnemyConfig spawns an enemy using per-instance tuning from Tiled or tests.
func (w *World) SpawnEnemyConfig(x, y float64, cfg EnemyConfig) EntityID {
	e := NewEnemyWithConfig(w.allocID(), x, y, cfg)
	w.Enemies = append(w.Enemies, e)
	return e.ID
}
