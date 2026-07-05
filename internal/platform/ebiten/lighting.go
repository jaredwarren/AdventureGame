package ebitenplat

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jaredwarren/game-test/internal/world"
)

const nightShaderCode = `
//kage:unit pixels
package main

var LightSource vec2
var LightRadius float
var PersonalRadius float
var AmbientColor vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	pixelPos := position.xy - imageDstOrigin()
	toPixel := pixelPos - LightSource
	dist := length(toPixel)

	// Personal ambient circle surrounding the player
	personalIntensity := 0.0
	if dist < PersonalRadius {
		personalIntensity = 1.0 - (dist / max(PersonalRadius, 0.001))
		personalIntensity = personalIntensity * personalIntensity * (3.0 - 2.0*personalIntensity)
	}

	// Flashlight/torch circle
	torchIntensity := 0.0
	if dist < LightRadius {
		torchIntensity = 1.0 - (dist / max(LightRadius, 0.001))
		torchIntensity = torchIntensity * torchIntensity * (3.0 - 2.0*torchIntensity)
	}

	totalLight := max(personalIntensity, torchIntensity)
	totalLight = clamp(totalLight, 0.0, 1.0)

	// Return ambient color tinted by (1.0 - totalLight)
	return AmbientColor * (1.0 - totalLight)
}
`

var nightShader *ebiten.Shader

func init() {
	var err error
	nightShader, err = ebiten.NewShader([]byte(nightShaderCode))
	if err != nil {
		panic("failed to compile night shader: " + err.Error())
	}
}

func (r *Renderer) drawNightOverlay(w *world.World) {
	mult := w.LightMultiplier()
	if mult >= 1.0 {
		return
	}
	alpha := float32((1.0 - mult) * 1.175)
	if alpha > 0.94 {
		alpha = 0.94
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha <= 0.01 {
		return
	}

	ox, oy := r.camOffX, r.camOffY
	vp := r.screen.Bounds()
	pr := w.PlayerRect()
	px := float32(pr.X - ox + pr.W*0.5)
	py := float32(pr.Y - oy + pr.H*0.5)

	lightR := float32(0.0)
	if w.HasCapability(world.CapLightSource) {
		lightR = float32(w.EffectiveStat(world.StatTorchLightRadius, w.Player.EffectiveTorchLightRadius()))
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"LightSource":    []float32{px, py},
		"PersonalRadius": float32(w.Player.EffectivePersonalLightRadius()),
		"LightRadius":    lightR,
		"AmbientColor":   []float32{0.03 * alpha, 0.03 * alpha, 0.15 * alpha, alpha},
	}
	r.screen.DrawRectShader(vp.Dx(), vp.Dy(), nightShader, op)
}
