// overlays.go — shared debug/UI overlays composed from services.Renderer
// primitives.
//
// These are intentionally free functions rather than methods on some
// overlay struct: they're stateless views over (Session, World) that
// multiple scenes reuse (HUD shows in play+pause+editor, debug overlay
// shows everywhere).
//
// Pure read: no helper mutates Session or World.
package scenes

import (
	"fmt"
	"image/color"

	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

// DebugOverlayExtras pulls scene-local state into the overlay without
// coupling overlays.go to the concrete scene types. PlayScene fills it in;
// other scenes pass a zero value.
type DebugOverlayExtras struct {
	HitStop      int
	DodgeImpulse int
}

// DrawHUD renders hearts, coins, keys, player pixel position (x,y),
// bomb count / torch flag, stamina bar, and the weekly epoch stamp. No-op when w is nil.
func DrawHUD(r services.Renderer, w *world.World, sess *Session) {
	if w == nil {
		return
	}
	y := 4
	maxH := w.MaxHP()
	for i := 0; i < maxH; i++ {
		c := color.RGBA{0x40, 0x40, 0x40, 0xff}
		if i < w.HP {
			c = color.RGBA{0xe0, 0x30, 0x40, 0xff}
		}
		r.FillRect(float32(4+i*8), float32(y), 7, 8, c)
	}
	r.DrawText(200, y, fmt.Sprintf("coin %d", w.Currency))
	r.DrawText(200, y+16, fmt.Sprintf("key %d", w.SmallKey))
	r.DrawText(200, y+32, fmt.Sprintf("%d,%d", int(w.Player.X), int(w.Player.Y)))
	invX := 260
	if w.Bombs > 0 {
		r.DrawText(invX, y, fmt.Sprintf("bomb x%d/%d", w.Bombs, world.MaxBombsCarry))
		invX += 88
	}
	if w.HasTorch {
		r.DrawText(invX, y, "torch")
	}

	barW := 60
	st := w.Player.Stamina
	maxS := w.MaxStamina()
	fw := float32(st) / float32(maxS) * float32(barW)
	r.FillRect(4, float32(y+12), float32(barW), 4, color.RGBA{0x30, 0x30, 0x40, 0xff})
	r.FillRect(4, float32(y+12), fw, 4, color.RGBA{0x50, 0xa0, 0xff, 0xff})

	r.DrawText(4, 220, fmt.Sprintf("weekly %d", sess.WeeklySeed))
}

// DrawDebugOverlay renders the F3 playtest HUD. Pure read.
func DrawDebugOverlay(r services.Renderer, sess *Session, sceneID SceneID, extras DebugOverlayExtras) {
	const lineH = 12
	px, py := 4, 44
	panelW, panelH := float32(214), float32(176)
	r.FillRect(float32(px-2), float32(py-6), panelW, panelH, color.RGBA{0x08, 0x0a, 0x10, 0x92})

	line := func(format string, args ...any) {
		r.DrawText(px, py, fmt.Sprintf(format, args...))
		py += lineH
	}

	w := sess.World
	tick := 0
	doorCD := 0
	if w != nil {
		tick = w.Tick
		doorCD = w.DoorCooldown
	}
	cam := r.Camera()
	line("scene %s tick %d", sceneID, tick)
	line("cam %.0f %.0f shk %d", cam.X, cam.Y, cam.ShakeTime)
	line("doorCD %d hitStop %d dodgeImp %d", doorCD, extras.HitStop, extras.DodgeImpulse)
	line("dbgF3 on")

	if w == nil {
		line("world nil")
		line("weekly %d", sess.WeeklySeed)
		return
	}

	pcX := w.Player.X + w.Player.W*0.5
	pcY := w.Player.Y + w.Player.H*0.5
	tx := int(pcX / world.TileSize)
	ty := int(pcY / world.TileSize)
	gid := w.GIDAt(tx, ty)

	line("map %s %dx%d", w.MapID, w.MapW, w.MapH)
	line("player %.1f %.1f  tile %d,%d", w.Player.X, w.Player.Y, tx, ty)
	line("gid@feet %d  dir %s", gid, world.DirLabel(w.Player.Dir))
	line("hp %d/%d stam %d/%d", w.HP, w.MaxHP(), w.Player.Stamina, w.MaxStamina())
	line("swing %d cd %d torch %d tcd %d i-%d dodge %d", w.Player.Swing, w.Player.SwingCD, w.Player.TorchSwing, w.Player.TorchSwingCD, w.Player.Invuln, w.Player.DodgeTimer)
	line("coin %d key %d bomb %d/%d torch %v", w.Currency, w.SmallKey, w.Bombs, world.MaxBombsCarry, w.HasTorch)
	wst := w.Stats
	line("stats V%d R%d M%d W%d F%d", wst.Vitality, wst.Resolve, wst.Might, wst.Wits, wst.Fortune)

	nEn, nEnLive := 0, 0
	for i := range w.Enemies {
		nEn++
		if w.Enemies[i].HP > 0 {
			nEnLive++
		}
	}
	nPick := 0
	for i := range w.Pickups {
		if !w.Pickups[i].Gone {
			nPick++
		}
	}
	line("foes %d/%d pkup %d doors %d shr %d", nEnLive, nEn, nPick, len(w.Doors), len(w.Shrines))
	line("weekly %d", sess.WeeklySeed)
}
