// play.go — main gameplay scene.
//
// PlayScene owns all per-frame state that exists only while gameplay is
// running:
//
//   - hitStop: frozen-frame counter after landing a combat hit (juice).
//   - dodgeImpulse: residual velocity kick after ActionDodge.
//   - pipeline: ordered systems that evolve World each tick.
//
// Phase 3 notes:
//
//   - UpdateSim is gone. The tick now runs a systems.Pipeline and drains
//     events into local translation to SFX / camera shake / hit-stop.
//   - The enemy-HP / currency / key snapshot-diff hack is gone; events are
//     authoritative.
package scenes

import (
	"image/color"
	"math/rand"

	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
)

// PlayScene is the main gameplay scene.
type PlayScene struct {
	hitStop      int
	dodgeImpulse int

	pipeline  *systems.Pipeline
	particles []*Particle
}

func newPlayScene() Scene {
	return &PlayScene{
		pipeline:  systems.Default(),
		particles: make([]*Particle, 0),
	}
}

func (s *PlayScene) ID() SceneID                                     { return ScenePlay }
func (s *PlayScene) Enter(ctx GameContext, params map[string]any) error { return nil }
func (s *PlayScene) Exit(ctx GameContext) error                         { return nil }

func (s *PlayScene) Update(ctx GameContext) error {
	in := ctx.Input()
	sess := ctx.Session()

	if in.JustPressed(services.ActionDebugToggle) {
		sess.ShowDebugOverlay = !sess.ShowDebugOverlay
	}
	if in.JustPressed(services.ActionPause) {
		ctx.Manager().Replace(ScenePause, nil)
		return nil
	}
	// Play-mode quicksave is chorded (Ctrl+S); plain S is bound to
	// ActionMoveDown via WASD fallback, so the modifier check is required
	// to disambiguate.
	if in.IsModifierDown(services.ModCtrl) && in.JustPressed(services.ActionQuickSave) {
		_ = save.Save("", BuildSave(sess, ctx.Renderer().Camera()))
	}

	// Hit-stop skips movement/combat for a few frames but still ticks
	// camera shake decay.
	if s.hitStop > 0 {
		s.hitStop--
		ctx.Renderer().Camera().TickShake()
		return nil
	}

	w := sess.World
	if w == nil {
		return nil
	}

	// Update existing particles
	s.particles = UpdateParticles(s.particles)

	// Ambient dust particles around player
	if len(s.particles) < 40 && rand.Float64() < 0.15 {
		pr := w.PlayerRect()
		// Spawn within a 160px radius of the player
		rx := pr.X + pr.W*0.5 + (rand.Float64()-0.5)*320
		ry := pr.Y + pr.H*0.5 + (rand.Float64()-0.5)*240
		s.particles = append(s.particles, NewDustParticle(rx, ry))
	}

	// Ambient ember particles if player has torch (or is swinging torch)
	if w.HasTorch && rand.Float64() < 0.25 {
		pr := w.PlayerRect()
		px := pr.X + pr.W*0.5
		py := pr.Y + pr.H*0.5
		s.particles = append(s.particles, NewEmberParticle(px, py))
	}

	s.handleActions(ctx, w)
	s.handleMovement(ctx, w)

	events, err := s.pipeline.Tick(w, services.TickSeconds)
	if err != nil {
		return err
	}
	s.reactToEvents(ctx, w, events)
	s.handleDeath(ctx)

	pr := w.PlayerRect()
	cam := ctx.Renderer().Camera()
	cam.Follow(pr.X+pr.W*0.5, pr.Y+pr.H*0.5)
	cam.TickShake()

	TryDoors(ctx.Assets(), sess, cam)

	return nil
}

// handleActions processes shrine interaction, sword, bomb, torch, and dodge input.
func (s *PlayScene) handleActions(ctx GameContext, w *world.World) {
	in := ctx.Input()
	pr := w.PlayerRect()
	if in.JustPressed(services.ActionInteract) {
		for _, sh := range w.Shrines {
			if pr.Overlaps(sh.Rect) {
				ctx.Manager().Replace(SceneShop, nil)
				break
			}
		}
	}

	if in.JustPressed(services.ActionAttack) {
		if w.TrySwingSword() {
			ctx.Audio().Play("swing.wav", 0.2)
		}
	}
	if in.JustPressed(services.ActionBomb) {
		if broke, key := w.TryDamageFaceTile(world.DamageBomb); broke {
			ctx.Session().MarkDestroyedTile(key)
			ctx.Audio().Play("hit.wav", 0.25)
			ctx.Renderer().Camera().AddShake(8, 3)

			// Spawn grey debris particles for the broken tile
			pc := w.PlayerRect()
			cx := pc.X + pc.W*0.5
			cy := pc.Y + pc.H*0.5
			tx := int(cx / world.TileSize)
			ty := int(cy / world.TileSize)
			switch w.Player.Dir {
			case world.DirDown:
				ty++
			case world.DirUp:
				ty--
			case world.DirLeft:
				tx--
			case world.DirRight:
				tx++
			}
			bx := float64(tx*world.TileSize) + world.TileSize*0.5
			by := float64(ty*world.TileSize) + world.TileSize*0.5
			for k := 0; k < 18; k++ {
				s.particles = append(s.particles, NewDebrisParticle(bx, by, color.RGBA{100, 100, 100, 255}))
			}
		}
	}
	if in.JustPressed(services.ActionTorch) {
		if w.TrySwingTorch() {
			ctx.Audio().Play("swing.wav", 0.22)
			// Spawn a few embers in front of player
			pc := w.PlayerRect()
			cx := pc.X + pc.W*0.5
			cy := pc.Y + pc.H*0.5
			for k := 0; k < 5; k++ {
				s.particles = append(s.particles, NewEmberParticle(cx, cy))
			}
		}
	}

	if in.JustPressed(services.ActionDodge) && w.Player.Stamina >= 20 {
		w.Player.Stamina -= 20
		w.Player.DodgeTimer = 20
		s.dodgeImpulse = 12
	}
}

// handleMovement reads directional input, applies sprint/stamina drain,
// dodge impulse, and moves the player. Stamina regen runs here too because
// it's the natural inverse of drain; a future StaminaSystem could migrate
// both once Player.Sprinting is modeled explicitly.
func (s *PlayScene) handleMovement(ctx GameContext, w *world.World) {
	in := ctx.Input()

	const baseSpd = 1.35
	spd := baseSpd
	if in.IsDown(services.ActionSprint) && w.Player.Stamina > 0 {
		spd = 2.15
		w.Player.Stamina--
		if w.Player.Stamina < 0 {
			w.Player.Stamina = 0
		}
	} else if w.Player.Stamina < w.MaxStamina() {
		if w.Tick%2 == 0 {
			w.Player.Stamina++
		}
	}

	ax, ay := in.Axis2D()
	dx := float64(ax) * spd
	dy := float64(ay) * spd
	if dx != 0 || dy != 0 {
		w.SetFacingFromMotion(dx, dy)
	}

	if s.dodgeImpulse > 0 {
		s.dodgeImpulse--
		switch w.Player.Dir {
		case world.DirDown:
			dy += 2.8
		case world.DirUp:
			dy -= 2.8
		case world.DirLeft:
			dx -= 2.8
		case world.DirRight:
			dx += 2.8
		}
	}

	w.TryMove(dx, dy)
}

// reactToEvents translates sim events into scene-level juice (SFX, camera
// shake, hit-stop) and side-effects that are properly scene-layer (boss
// kill coin bonus). Runs once per tick after the pipeline.
func (s *PlayScene) reactToEvents(ctx GameContext, w *world.World, events []systems.Event) {
	cam := ctx.Renderer().Camera()
	lockOpened := false
	for _, ev := range events {
		switch e := ev.(type) {
		case systems.HitEvent:
			if !e.FromBurnDoT {
				s.hitStop = 5
				cam.AddShake(6, 2.5)
				ctx.Audio().Play("hit.wav", 0.3)
			}
			if e.Killed && e.IsBoss {
				w.Currency += 25
			}
			// Find enemy position
			var ex, ey float64
			for _, enemy := range w.Enemies {
				if enemy.ID == e.EnemyID {
					ex, ey = enemy.X+enemy.W*0.5, enemy.Y+enemy.H*0.5
					break
				}
			}
			if ex != 0 || ey != 0 {
				count := 5
				clr := color.RGBA{180, 40, 40, 255}
				if e.Killed {
					count = 15
				}
				if e.IsBoss {
					clr = color.RGBA{110, 10, 10, 255}
				}
				for k := 0; k < count; k++ {
					s.particles = append(s.particles, NewDebrisParticle(ex, ey, clr))
				}
			}
		case systems.TileDestroyedEvent:
			ctx.Session().MarkDestroyedTile(e.SaveKey)
		case systems.PickupEvent:
			ctx.Session().MarkPersistentPickupCollected(e.PersistentSaveKey)
			ctx.Audio().Play("pickup.wav", 0.25)
			var px, py float64
			for _, p := range w.Pickups {
				if p.ID == e.PickupID {
					px, py = p.X+p.W*0.5, p.Y+p.H*0.5
					break
				}
			}
			if px != 0 || py != 0 {
				for k := 0; k < 8; k++ {
					s.particles = append(s.particles, NewSparkleParticle(px, py))
				}
			}
		case systems.PlayerHurtEvent:
			cam.AddShake(4, 2)
			pr := w.PlayerRect()
			pxx := pr.X + pr.W*0.5
			pyy := pr.Y + pr.H*0.5
			for k := 0; k < 8; k++ {
				s.particles = append(s.particles, NewDebrisParticle(pxx, pyy, color.RGBA{220, 30, 30, 255}))
			}
		case systems.LockOpenEvent:
			ctx.Session().MarkOpenedLockTile(world.MapTilePersistKey(w.MapID, e.Tile[0], e.Tile[1]))
			// Coalesce multi-tile lock openings into a single SFX/shake.
			lockOpened = true
			lx := float64(e.Tile[0]*world.TileSize) + world.TileSize*0.5
			ly := float64(e.Tile[1]*world.TileSize) + world.TileSize*0.5
			for k := 0; k < 10; k++ {
				s.particles = append(s.particles, NewSparkleParticle(lx, ly))
			}
		}
	}
	if lockOpened {
		ctx.Audio().Play("pickup.wav", 0.32)
		cam.AddShake(5, 2)
	}
}

// handleDeath checks for player death and respawns.
func (s *PlayScene) handleDeath(ctx GameContext) {
	if ctx.Session().World != nil && ctx.Session().World.HP <= 0 {
		Respawn(ctx.Assets(), ctx.Session())
		ctx.Audio().Play("hit.wav", 0.4)
	}
}

func (s *PlayScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()
	if sess.World != nil {
		r.DrawWorld(sess.World)
		DrawParticles(r, s.particles)
		DrawHUD(r, sess.World, sess)
		if sess.ShowDebugOverlay {
			DrawDebugOverlay(r, sess, s.ID(), DebugOverlayExtras{
				HitStop:      s.hitStop,
				DodgeImpulse: s.dodgeImpulse,
			})
		}
	} else {
		r.DrawText(0, 0, "no world")
	}
}
