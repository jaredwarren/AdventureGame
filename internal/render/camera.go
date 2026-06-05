// Package render holds presentation helpers that are still “math first” (no Ebiten images here).
package render

import "math/rand"

// Camera stores the top-left of the view rectangle in world pixels plus simple per-frame shake.
// NOTE: There is no clamping to map bounds yet—out-of-bounds tiles draw as void unless guarded in drawWorld.
type Camera struct {
	X, Y             float64 // top-left of view in world px
	ShakeTime        int     // frames remaining
	ShakeMag         float64
	ReduceShake      bool
	screenW, screenH int
}

func NewCamera(screenW, screenH int) *Camera {
	return &Camera{screenW: screenW, screenH: screenH}
}

func (c *Camera) SetScreenSize(w, h int) {
	c.screenW, c.screenH = w, h
}

// Follow centers the camera on the given world point (typically player center).
func (c *Camera) Follow(worldCX, worldCY float64) {
	c.X = worldCX - float64(c.screenW)*0.5
	c.Y = worldCY - float64(c.screenH)*0.5
}

func (c *Camera) AddShake(frames int, magnitude float64) {
	if c.ReduceShake {
		frames /= 2
		magnitude *= 0.35
		if frames < 1 {
			frames = 1
		}
	}
	if frames > c.ShakeTime {
		c.ShakeTime = frames
		c.ShakeMag = magnitude
	}
}

func (c *Camera) TickShake() {
	if c.ShakeTime > 0 {
		c.ShakeTime--
	}
}

// Offset returns a random shake offset for this frame (uses global math/rand; fine for VFX).
// TODO: seed from game RNG if you need deterministic replay.
func (c *Camera) Offset() (dx, dy float64) {
	if c.ShakeTime <= 0 {
		return 0, 0
	}
	m := c.ShakeMag * (float64(c.ShakeTime) / 8.0)
	if m < 0.5 {
		m = 0.5
	}
	dx = (rand.Float64()*2 - 1) * m
	dy = (rand.Float64()*2 - 1) * m
	return dx, dy
}
