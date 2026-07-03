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
	"math"
	"math/rand"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
)

type ActiveBomb struct {
	X, Y   float64
	TX, TY int
	Timer  int
}

// PlayScene is the main gameplay scene.
type PlayScene struct {
	hitStop      int
	dodgeImpulse int

	pipeline  *systems.Pipeline
	particles []*Particle
	bombs     []*ActiveBomb

	showItemMenu   bool
	itemMenuCursor int

	toastItem    *world.PickupKind
	toastTimer   int
	toastMessage string
}

func newPlayScene() Scene {
	return &PlayScene{
		pipeline:  systems.Default(),
		particles: make([]*Particle, 0),
		bombs:     make([]*ActiveBomb, 0),
	}
}

func (s *PlayScene) ID() SceneID                                        { return ScenePlay }
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

	if s.toastTimer > 0 {
		s.toastTimer--
	}

	// Item menu intercepts all gameplay input while open.
	if in.JustPressed(services.ActionItemMenu) && !s.showItemMenu {
		s.showItemMenu = true
		s.itemMenuCursor = int(w.SelectedItem)
		return nil
	}
	if s.showItemMenu {
		s.handleItemMenu(ctx, w)
		return nil
	}

	// Update existing particles
	s.particles = UpdateParticles(s.particles)

	// Update active bombs
	var activeBombs []*ActiveBomb
	for _, b := range s.bombs {
		b.Timer--
		if b.Timer <= 0 {
			// Explode!
			if broke, saveKey := w.BreakTileAt(b.TX, b.TY, world.DamageBomb); broke {
				ctx.Session().MarkDestroyedTile(w.MapID, saveKey)
			}

			// Damage nearby enemies within 32 pixels
			for i := range w.Enemies {
				enemy := &w.Enemies[i]
				if enemy.HP <= 0 {
					continue
				}
				ecx, ecy := enemy.Center()
				dist := math.Hypot(ecx-b.X, ecy-b.Y)
				bombRadius := w.Player.EffectiveBombRadius()
				bombDamage := w.Player.EffectiveBombDamage()
				if dist <= bombRadius {
					w.DamageEnemy(i, bombDamage) // high damage
				}
			}

			// Audio & Camera Shake
			ctx.Audio().Play("hit.wav", 0.3)
			ctx.Renderer().Camera().AddShake(12, 4.5)

			// Spawn explosion/debris particles
			for k := 0; k < 16; k++ {
				// Debris
				s.particles = append(s.particles, NewDebrisParticle(b.X, b.Y, color.RGBA{100, 100, 100, 255}))
				// Embers
				angle := rand.Float64() * 2 * math.Pi
				speed := rand.Float64()*2.0 + 1.0
				s.particles = append(s.particles, &Particle{
					X:      b.X,
					Y:      b.Y,
					VX:     math.Cos(angle) * speed,
					VY:     math.Sin(angle) * speed,
					Color:  color.RGBA{255, uint8(100 + rand.Intn(150)), 0, 255},
					Type:   ParticleEmber,
					Size:   rand.Float32()*3.0 + 1.5,
					Life:   1.0,
					Decay:  rand.Float64()*0.05 + 0.03,
					Wobble: rand.Float64() * 10,
				})
			}
		} else {
			// Spawn fuse sparks at top-right of the bomb at the end of the fuse line (x + 3, y - 9)
			if rand.Float64() < 0.6 {
				s.particles = append(s.particles, &Particle{
					X:      b.X + 3,
					Y:      b.Y - 9,
					VX:     (rand.Float64() - 0.5) * 0.5,
					VY:     -rand.Float64()*0.5 - 0.2,
					Color:  color.RGBA{255, uint8(180 + rand.Intn(75)), 0, 255},
					Type:   ParticleEmber,
					Size:   rand.Float32()*1.2 + 0.6,
					Life:   1.0,
					Decay:  rand.Float64()*0.05 + 0.05,
					Wobble: rand.Float64() * 5,
				})
			}
			activeBombs = append(activeBombs, b)
		}
	}
	s.bombs = activeBombs

	// Spawn flame particles for active flames
	for _, f := range w.Flames {
		if rand.Float64() < 0.4 {
			px := f.X + (rand.Float64()-0.5)*12
			py := f.Y + (rand.Float64()-0.5)*12
			s.particles = append(s.particles, &Particle{
				X:      px,
				Y:      py,
				VX:     (rand.Float64() - 0.5) * 0.3,
				VY:     -rand.Float64()*0.4 - 0.4,
				Color:  color.RGBA{255, uint8(80 + rand.Intn(120)), 0, 255},
				Type:   ParticleEmber,
				Size:   rand.Float32()*2.5 + 1.0,
				Life:   1.0,
				Decay:  rand.Float64()*0.04 + 0.02,
				Wobble: rand.Float64() * 5,
			})
		}
	}

	// Ambient dust particles around player
	if len(s.particles) < 40 && rand.Float64() < 0.15 {
		pr := w.PlayerRect()
		// Spawn within a 160px radius of the player
		rx := pr.X + pr.W*0.5 + (rand.Float64()-0.5)*320
		ry := pr.Y + pr.H*0.5 + (rand.Float64()-0.5)*240
		s.particles = append(s.particles, NewDustParticle(rx, ry))
	}

	// Ambient ember particles only when torch is the active item.
	if w.HasTorch && w.SelectedItem == world.ItemSlotTorch && rand.Float64() < 0.25 {
		pr := w.PlayerRect()
		px := pr.X + pr.W*0.5
		py := pr.Y + pr.H*0.5
		s.particles = append(s.particles, NewEmberParticle(px, py))
	}

	s.handleActions(ctx, w)
	s.handleMovement(ctx, w)

	// Check if player touched any shrines
	pr := w.PlayerRect()
	for i := range w.Shrines {
		sh := &w.Shrines[i]
		if !sh.Active && pr.Overlaps(sh.Rect) {
			sh.Active = true
			sess.MarkShrineActivated(w.MapID, world.PersistentShrineSaveKey(w.MapID, sh.TiledID))
			ctx.Audio().Play("pickup.wav", 0.25)
			scx, scy := sh.Rect.X+sh.Rect.W*0.5, sh.Rect.Y+sh.Rect.H*0.5
			for k := 0; k < 12; k++ {
				s.particles = append(s.particles, NewSparkleParticle(scx, scy))
			}
		}
	}

	events, err := s.pipeline.Tick(w, services.TickSeconds)
	if err != nil {
		return err
	}
	s.reactToEvents(ctx, w, events)
	s.handleDeath(ctx)

	pr = w.PlayerRect()
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
		chestOpened := false
		for i := range w.Pickups {
			p := &w.Pickups[i]
			if p.PersistentSaveKey != "" && !p.Opened && !p.Gone && pr.OverlapsExpanded(p.Rect(), 2.0) {
				p.Opened = true
				chestOpened = true

				w.ApplyPickupReward(p.Kind)

				// Persist save state
				ctx.Session().MarkPersistentPickupCollected(w.MapID, p.PersistentSaveKey)

				// Play pickup audio
				ctx.Audio().Play("pickup.wav", 0.25)

				// Spawn sparkle particles (same as systems.PickupEvent reaction)
				px, py := p.X+p.W*0.5, p.Y+p.H*0.5
				for k := 0; k < 8; k++ {
					s.particles = append(s.particles, NewSparkleParticle(px, py))
				}

				// Set toast notification
				s.toastItem = p.Kind
				s.toastTimer = 150
				s.toastMessage = p.Kind.ToastMessage()
				break
			}
		}

		if !chestOpened {
			for _, sh := range w.Shrines {
				if pr.Overlaps(sh.Rect) {
					ctx.Manager().Replace(SceneShop, nil)
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
	if w.Bombs <= 0 {
		return
	}
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
	for _, b := range s.bombs {
		if b.TX == tx && b.TY == ty {
			return
		}
	}
	w.Bombs--
	s.bombs = append(s.bombs, &ActiveBomb{
		X: bx, Y: by, TX: tx, TY: ty,
		Timer: w.Player.EffectiveBombFuseDuration(),
	})
	ctx.Audio().Play("swing.wav", 0.15)
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

	if !isSprintHeld {
		w.Player.SprintExhausted = false
	}

	spd := w.Player.EffectiveBaseSpeed()

	isSprinting := isSprintHeld && isMoving && !w.Player.SprintExhausted && w.Player.Stamina > 0

	if isSprinting {
		spd = w.Player.EffectiveSprintSpeed()
		w.Player.Stamina--
		if w.Player.Stamina <= 0 {
			w.Player.Stamina = 0
			w.Player.SprintExhausted = true
		}
	} else if w.Player.Stamina < w.MaxStamina() {
		if w.Tick%w.Player.EffectiveStaminaRegenInterval() == 0 {
			w.Player.Stamina++
		}
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
				w.Currency += progression.DefaultEconomy().BossKillCoinBonus
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
			ctx.Session().MarkDestroyedTile(w.MapID, e.SaveKey)
			if _, tx, ty, ok := world.ParseMapTilePersistKey(e.SaveKey); ok {
				bx := float64(tx*world.TileSize) + world.TileSize*0.5
				by := float64(ty*world.TileSize) + world.TileSize*0.5
				for k := 0; k < 12; k++ {
					s.particles = append(s.particles, NewDebrisParticle(bx, by, color.RGBA{60, 60, 60, 255}))
					s.particles = append(s.particles, NewEmberParticle(bx, by))
				}
				ctx.Audio().Play("hit.wav", 0.2)
			}
		case systems.PickupEvent:
			ctx.Session().MarkPersistentPickupCollected(w.MapID, e.PersistentSaveKey)
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
			ctx.Session().MarkOpenedLockTile(w.MapID, world.MapTilePersistKey(w.MapID, e.Tile[0], e.Tile[1]))
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

		// Draw active bombs
		cam := r.Camera()
		ox, oy := cam.X, cam.Y
		for _, b := range s.bombs {
			sx := float32(b.X - ox)
			sy := float32(b.Y - oy)

			var cycleLen int
			if b.Timer > 60 {
				cycleLen = 20
			} else if b.Timer > 30 {
				cycleLen = 10
			} else if b.Timer > 10 {
				cycleLen = 4
			} else {
				cycleLen = 2
			}
			isFlash := (b.Timer/(cycleLen/2))%2 == 0

			bodyColor := color.RGBA{20, 20, 20, 255}
			if isFlash {
				bodyColor = color.RGBA{240, 50, 50, 255}
			}

			// Draw bomb body (rounded 10px diameter shape using overlapping rects)
			r.FillRect(sx-5, sy-4, 10, 8, bodyColor)
			r.FillRect(sx-4, sy-5, 8, 10, bodyColor)

			// Draw fuse cap (dark grey)
			r.FillRect(sx-2, sy-7, 4, 2, color.RGBA{60, 60, 60, 255})

			// Draw fuse line (brown/grey curve)
			r.StrokeLine(sx, sy-7, sx+3, sy-9, 1, color.RGBA{130, 110, 90, 255})
		}

		// Draw active flames
		for _, f := range sess.World.Flames {
			sx := float32(f.X - ox)
			sy := float32(f.Y - oy)
			tick := sess.World.Tick
			flicker := (tick / 4) % 3

			// Outer red flame
			r.FillRect(sx-6, sy-3+float32(flicker), 12, 8-float32(flicker), color.RGBA{230, 50, 20, 220})
			r.FillRect(sx-4, sy-7+float32(flicker), 8, 12-float32(flicker), color.RGBA{230, 50, 20, 220})

			// Inner orange flame
			r.FillRect(sx-4, sy-1+float32(flicker), 8, 6-float32(flicker), color.RGBA{255, 120, 20, 240})
			r.FillRect(sx-2, sy-5+float32(flicker), 4, 10-float32(flicker), color.RGBA{255, 120, 20, 240})

			// Core yellow flame
			r.FillRect(sx-2, sy+1, 4, 3, color.RGBA{255, 220, 40, 255})
			r.FillRect(sx-1, sy-2+float32(flicker), 2, 6-float32(flicker), color.RGBA{255, 220, 40, 255})
		}

		DrawHUD(r, sess.World, sess)
		if s.toastTimer > 0 {
			s.drawToast(r)
		}
		if s.showItemMenu {
			DrawItemMenu(r, sess.World, s.itemMenuCursor)
		}
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

func (s *PlayScene) drawToast(r services.Renderer) {
	const bannerW float32 = 180
	const bannerH float32 = 26
	const screenW float32 = 320

	bx := (screenW - bannerW) / 2
	var by float32 = 16
	var alpha float64 = 1.0

	// micro-animations: slide in and fade out
	if s.toastTimer > 135 { // first 15 frames (0.25s) slide down
		t := float32(150-s.toastTimer) / 15.0 // 0 to 1
		by = -bannerH + t*(16+bannerH)
	} else if s.toastTimer < 15 { // last 15 frames (0.25s) fade out
		alpha = float64(s.toastTimer) / 15.0
	}

	// Draw dark premium background with gold border
	bgCol := color.RGBA{0x0b, 0x0c, 0x14, uint8(230 * alpha)}
	borderCol := color.RGBA{0xd8, 0xa0, 0x20, uint8(255 * alpha)}
	r.FillRect(bx, by, bannerW, bannerH, bgCol)
	r.StrokeRect(bx, by, bannerW, bannerH, 1, borderCol)

	// Draw item image on the left of the banner
	// Size: 16x16, positioned at bx + 8, by + 5
	r.DrawPickupScreen(s.toastItem, bx+8, by+5, 16, 16)

	// Draw text to the right of the image
	// Text x: bx + 32, y: by + 9
	r.DrawText(int(bx)+32, int(by)+9, s.toastMessage)
}
