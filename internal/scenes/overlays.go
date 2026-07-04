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

	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
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
func DrawHUD(r services.Renderer, w *world.World, sess *run.Session) {
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
	// Item slot pill boxes — selected slot gets bright border, unselected gets dim.
	const pillH float32 = 18
	const pillGap float32 = 4
	px := float32(260)
	drawSlot := func(kind *world.PickupKind, selected bool, countText string) {
		var border color.RGBA
		if selected {
			border = color.RGBA{0xff, 0xff, 0xff, 0xff}
		} else {
			border = color.RGBA{0x60, 0x60, 0x60, 0xff}
		}
		var pillW float32 = 18
		if countText != "" {
			pillW = 34
		}
		r.FillRect(px, float32(y)-1, pillW, pillH, color.RGBA{0x10, 0x10, 0x18, 0xcc})
		r.StrokeRect(px, float32(y)-1, pillW, pillH, 1, border)
		r.DrawPickupScreen(kind, px+1, float32(y), 16, 16)
		if countText != "" {
			r.DrawText(int(px)+19, y+4, countText)
		}
		px += pillW + pillGap
	}
	drawSlot(world.PickupBomb, w.SelectedItem == world.ItemSlotBomb, fmt.Sprintf("%d", w.Bombs))
	if w.HasTorch {
		drawSlot(world.PickupTorch, w.SelectedItem == world.ItemSlotTorch, "")
	}

	barW := 60
	st := w.Player.Stamina
	maxS := w.MaxStamina()
	fw := float32(st) / float32(maxS) * float32(barW)
	r.FillRect(4, float32(y+12), float32(barW), 4, color.RGBA{0x30, 0x30, 0x40, 0xff})
	r.FillRect(4, float32(y+12), fw, 4, color.RGBA{0x50, 0xa0, 0xff, 0xff})

	r.DrawText(4, 208, fmt.Sprintf("map %s", w.MapID))
	r.DrawText(4, 220, fmt.Sprintf("weekly %d", sess.WeeklySeed))
}

// DrawDebugOverlay renders the F3 playtest HUD. Pure read.
func DrawDebugOverlay(r services.Renderer, sess *run.Session, sceneID SceneID, extras DebugOverlayExtras) {
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
	tx := int(pcX / tile.Size)
	ty := int(pcY / tile.Size)
	gid := w.GIDAt(tx, ty)

	line("map %s %dx%d", w.MapID, w.MapW, w.MapH)
	line("player %.1f %.1f  tile %d,%d", w.Player.X, w.Player.Y, tx, ty)
	line("gid@feet %d  dir %s", gid, world.DirLabel(w.Player.Dir))
	line("hp %d/%d stam %d/%d", w.HP, w.MaxHP(), w.Player.Stamina, w.MaxStamina())
	line("swing %d cd %d torch %d tcd %d i-%d dodge %d", w.Player.Swing, w.Player.SwingCD, w.Player.TorchSwing, w.Player.TorchSwingCD, w.Player.Invuln, w.Player.DodgeTimer)
	line("coin %d key %d bomb %d/%d torch %v", w.Currency, w.SmallKey, w.Bombs, w.MaxBombsCarry(), w.HasTorch)
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

// DrawItemMenu renders the item selection overlay. cursor is the index into
// the visible slot list (bomb always first, torch second if owned).
func DrawItemMenu(r services.Renderer, w *world.World, cursor int) {
	if w == nil {
		return
	}

	const panelX, panelY float32 = 72, 52
	const panelW, panelH float32 = 176, 120
	r.FillRect(panelX, panelY, panelW, panelH, color.RGBA{0x08, 0x0a, 0x14, 0xe0})
	r.StrokeRect(panelX, panelY, panelW, panelH, 1, color.RGBA{0x80, 0x80, 0xa0, 0xff})

	tx := int(panelX) + 8
	r.DrawText(tx+44, int(panelY)+6, "ITEMS")

	// Build slot list.
	type slot struct {
		label string
		kind  world.ItemSlot
	}
	slots := []slot{
		{fmt.Sprintf("BOMB  x%d/%d", w.Bombs, w.MaxBombsCarry()), world.ItemSlotBomb},
	}
	if w.HasTorch {
		slots = append(slots, slot{"TORCH", world.ItemSlotTorch})
	}

	lineY := int(panelY) + 22
	for i, sl := range slots {
		prefix := "  "
		if i == cursor {
			prefix = "> "
			r.FillRect(panelX+4, float32(lineY)-1, panelW-8, 12, color.RGBA{0x20, 0x28, 0x40, 0xff})
		}
		r.DrawText(tx, lineY, prefix+sl.label)
		lineY += 14
	}

	// Divider.
	divY := float32(lineY) + 2
	r.StrokeLine(panelX+4, divY, panelX+panelW-4, divY, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Stats block.
	infoY := int(divY) + 6
	r.DrawText(tx, infoY, fmt.Sprintf("HP %d/%d   Stam %d/%d", w.HP, w.MaxHP(), w.Player.Stamina, w.MaxStamina()))
	infoY += 12
	r.DrawText(tx, infoY, fmt.Sprintf("Coins %d   Keys %d", w.Currency, w.SmallKey))
	infoY += 12
	st := w.Stats
	r.DrawText(tx, infoY, fmt.Sprintf("V%d R%d M%d W%d F%d", st.Vitality, st.Resolve, st.Might, st.Wits, st.Fortune))

	// Footer hint.
	r.DrawText(tx, int(panelY)+int(panelH)-14, "up/dn select  Enter confirm  Esc close")
}

// DrawOverlayDim darkens the framebuffer under a pause/shop overlay.
func DrawOverlayDim(r services.Renderer) {
	r.FillRect(0, 0, 320, 240, color.RGBA{0, 0, 0, 100})
}
