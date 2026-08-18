// play.go — main gameplay scene.
//
// PlayScene owns all per-frame state that exists only while gameplay is
// running:
//
//   - hitStop: frozen-frame counter after landing a combat hit (juice).
//   - dodgeImpulse: residual velocity kick after ActionDodge.
//   - pipeline: ordered systems that evolve World each tick.
package scenes

import (
	"github.com/jaredwarren/game-test/internal/run"
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
	}
}

func (s *PlayScene) ID() SceneID                                        { return ScenePlay }
func (s *PlayScene) Enter(ctx GameContext, params map[string]any) error { return nil }
func (s *PlayScene) Exit(ctx GameContext) error                         { return nil }

func (s *PlayScene) Update(ctx GameContext) error {
	in := ctx.Input()
	sess := ctx.Session()

	HandleSessionDebugInput(sess, in)
	if in.JustPressed(services.ActionPause) {
		ctx.Manager().PushOverlay(ScenePause, nil)
		return nil
	}
	if in.JustPressed(services.ActionMap) {
		ctx.Manager().PushOverlay(SceneMap, nil)
		return nil
	}
	if in.IsModifierDown(services.ModCtrl) && in.JustPressed(services.ActionQuickSave) {
		_ = save.Save("", run.BuildSave(sess, ctx.Renderer().Camera()))
	}

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

	if in.JustPressed(services.ActionItemMenu) && !s.showItemMenu {
		s.showItemMenu = true
		s.itemMenuCursor = int(w.SelectedItem)
		return nil
	}
	if s.showItemMenu {
		s.handleItemMenu(ctx, w)
		return nil
	}

	s.particles = UpdateParticles(s.particles)
	s.tickAmbientParticles(w)

	s.handleActions(ctx, w)
	s.handleMovement(ctx, w)

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

	run.TryDoors(ctx.Assets(), sess, cam)
	return nil
}

func (s *PlayScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()
	if sess.World != nil {
		r.DrawWorld(sess.World)
		if sess.ShowDoorHitboxes {
			r.DrawDoorHitboxes(sess.World)
		}
		DrawParticles(r, s.particles)

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
