package scenes

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/jaredwarren/game-test/internal/services"
)

type ParticleType int

const (
	ParticleDust ParticleType = iota
	ParticleDebris
	ParticleEmber
	ParticleSparkle
)

type Particle struct {
	X, Y     float64
	VX, VY   float64
	Color    color.RGBA
	Type     ParticleType
	Size     float32
	Life     float64 // 1.0 down to 0.0
	Decay    float64
	Wobble   float64
}

func NewDustParticle(x, y float64) *Particle {
	return &Particle{
		X:      x,
		Y:      y,
		VX:     (rand.Float64() - 0.5) * 0.1,
		VY:     rand.Float64()*0.1 + 0.05,
		Color:  color.RGBA{180, 200, 220, 100},
		Type:   ParticleDust,
		Size:   rand.Float32()*1.5 + 0.5,
		Life:   1.0,
		Decay:  rand.Float64()*0.002 + 0.001,
		Wobble: rand.Float64() * 100,
	}
}

func NewDebrisParticle(x, y float64, c color.RGBA) *Particle {
	angle := rand.Float64() * 2 * math.Pi
	speed := rand.Float64()*1.5 + 0.5
	return &Particle{
		X:     x,
		Y:     y,
		VX:    math.Cos(angle) * speed,
		VY:    math.Sin(angle)*speed - 0.5,
		Color: c,
		Type:  ParticleDebris,
		Size:  rand.Float32()*2.5 + 1.5,
		Life:  1.0,
		Decay: rand.Float64()*0.04 + 0.02,
	}
}

func NewEmberParticle(x, y float64) *Particle {
	return &Particle{
		X:      x,
		Y:      y,
		VX:     (rand.Float64() - 0.5) * 0.2,
		VY:     -rand.Float64()*0.5 - 0.3,
		Color:  color.RGBA{255, uint8(100 + rand.Intn(100)), 30, 200},
		Type:   ParticleEmber,
		Size:   rand.Float32()*1.8 + 0.8,
		Life:   1.0,
		Decay:  rand.Float64()*0.025 + 0.015,
		Wobble: rand.Float64() * 10,
	}
}

func NewSparkleParticle(x, y float64) *Particle {
	return &Particle{
		X:     x,
		Y:     y,
		VX:    (rand.Float64() - 0.5) * 0.4,
		VY:    (rand.Float64() - 0.5) * 0.4,
		Color: cGolden(),
		Type:  ParticleSparkle,
		Size:  rand.Float32()*2.0 + 1.0,
		Life:  1.0,
		Decay: rand.Float64()*0.03 + 0.02,
	}
}

func cGolden() color.RGBA {
	return color.RGBA{255, 215, 0, 220}
}

func UpdateParticles(particles []*Particle) []*Particle {
	var active []*Particle
	for _, p := range particles {
		p.Life -= p.Decay
		if p.Life <= 0 {
			continue
		}

		switch p.Type {
		case ParticleDust:
			p.Wobble += 0.02
			p.X += math.Sin(p.Wobble) * 0.08
			p.Y += p.VY
		case ParticleDebris:
			p.VY += 0.12 // Gravity
			p.VX *= 0.95
			p.VY *= 0.95
			p.X += p.VX
			p.Y += p.VY
		case ParticleEmber:
			p.Wobble += 0.05
			p.X += math.Sin(p.Wobble) * 0.15
			p.Y += p.VY
		case ParticleSparkle:
			p.VX *= 0.92
			p.VY *= 0.92
			p.X += p.VX
			p.Y += p.VY
		}
		active = append(active, p)
	}
	return active
}

func DrawParticles(r services.Renderer, particles []*Particle) {
	cam := r.Camera()
	ox, oy := cam.X, cam.Y
	for _, p := range particles {
		sx := float32(p.X - ox)
		sy := float32(p.Y - oy)

		c := p.Color
		c.A = uint8(float64(c.A) * p.Life)

		r.FillRect(sx-p.Size*0.5, sy-p.Size*0.5, p.Size, p.Size, c)
	}
}
