package scenes

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/systems"
	"github.com/jaredwarren/game-test/internal/world"
)

func (s *PlayScene) tickAmbientParticles(w *world.World) {
	for _, b := range w.ActiveBombs {
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
	}

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

	if len(s.particles) < 40 && rand.Float64() < 0.15 {
		pr := w.PlayerRect()
		rx := pr.X + pr.W*0.5 + (rand.Float64()-0.5)*320
		ry := pr.Y + pr.H*0.5 + (rand.Float64()-0.5)*240
		s.particles = append(s.particles, NewDustParticle(rx, ry))
	}

	if w.HasTorch && w.SelectedItem == world.ItemSlotTorch && rand.Float64() < 0.25 {
		pr := w.PlayerRect()
		px := pr.X + pr.W*0.5
		py := pr.Y + pr.H*0.5
		s.particles = append(s.particles, NewEmberParticle(px, py))
	}
}

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
		case systems.ExplosionEvent:
			ctx.Audio().Play("hit.wav", 0.3)
			cam.AddShake(12, 4.5)
			for k := 0; k < 16; k++ {
				s.particles = append(s.particles, NewDebrisParticle(e.X, e.Y, color.RGBA{100, 100, 100, 255}))
				angle := rand.Float64() * 2 * math.Pi
				speed := rand.Float64()*2.0 + 1.0
				s.particles = append(s.particles, &Particle{
					X:      e.X,
					Y:      e.Y,
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
		case systems.PickupEvent:
			if e.PersistentSaveKey != "" {
				ctx.Session().MarkPersistentPickupCollected(w.MapID, e.PersistentSaveKey)
				s.toastItem = e.Kind
				s.toastTimer = 150
				s.toastMessage = e.Kind.ToastMessage()
			}
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

func (s *PlayScene) handleDeath(ctx GameContext) {
	if ctx.Session().World != nil && ctx.Session().World.HP <= 0 {
		run.Respawn(ctx.Assets(), ctx.Session())
		ctx.Audio().Play("hit.wav", 0.4)
	}
}

func (s *PlayScene) drawToast(r services.Renderer) {
	const bannerW float32 = 180
	const bannerH float32 = 26
	const screenW float32 = 320

	bx := (screenW - bannerW) / 2
	var by float32 = 16
	var alpha float64 = 1.0

	if s.toastTimer > 135 {
		t := float32(150-s.toastTimer) / 15.0
		by = -bannerH + t*(16+bannerH)
	} else if s.toastTimer < 15 {
		alpha = float64(s.toastTimer) / 15.0
	}

	bgCol := color.RGBA{0x0b, 0x0c, 0x14, uint8(230 * alpha)}
	borderCol := color.RGBA{0xd8, 0xa0, 0x20, uint8(255 * alpha)}
	r.FillRect(bx, by, bannerW, bannerH, bgCol)
	r.StrokeRect(bx, by, bannerW, bannerH, 1, borderCol)
	r.DrawPickupScreen(s.toastItem, bx+8, by+5, 16, 16)
	r.DrawText(int(bx)+32, int(by)+9, s.toastMessage)
}
