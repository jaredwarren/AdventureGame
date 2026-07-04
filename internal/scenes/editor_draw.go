package scenes

import (
	"fmt"
	"image/color"

	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func (s *EditorScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()
	cam := r.Camera()
	cam.X, cam.Y = 0, 0
	cam.ShakeTime = 0

	if s.errMsg != "" && s.tm == nil {
		r.DrawText(0, 0, s.errMsg)
		return
	}

	if sess.World != nil {
		r.DrawWorld(sess.World)
	}

	if s.modeTile {
		s.drawTileGrid(ctx)
		if s.showTileMenu {
			s.drawTileMenu(ctx)
		}
	} else {
		s.drawMarkerOverlay(ctx)
		if s.showItemMenu {
			s.drawItemMenu(ctx)
		}
	}

	mode := "MARKER"
	if s.modeTile {
		mode = fmt.Sprintf("TILE (brush: %s)", world.TileDefOf(s.brushGID).Name)
	}
	line := fmt.Sprintf("%s map=%s", mode, s.mapID)
	if s.errMsg != "" {
		line += " ERR:" + s.errMsg
	}
	r.DrawText(4, 4, line)
	r.DrawText(4, 16, "E mode  Tab menu  [ ] type  A add  Del  Ctrl+S save  Esc title")
	if !s.modeTile {
		types := world.MarkerTypeNames()
		r.DrawText(4, 28, fmt.Sprintf("new: %s", types[s.markerTypeIndex%len(types)]))
		s.drawSelectedEnemyProps(r)
	}
	if s.savedFlash > 0 {
		r.DrawText(280, 4, "saved")
	}
}

func (s *EditorScene) drawTileGrid(ctx GameContext) {
	w := ctx.Session().World
	if w == nil {
		return
	}
	r := ctx.Renderer()
	gc := color.RGBA{0xff, 0xff, 0xff, 0x18}
	for tx := 0; tx <= w.MapW; tx++ {
		x := float32(tx * tile.Size)
		r.StrokeLine(x, 0, x, float32(w.MapH*tile.Size), 1, gc)
	}
	for ty := 0; ty <= w.MapH; ty++ {
		y := float32(ty * tile.Size)
		r.StrokeLine(0, y, float32(w.MapW*tile.Size), y, 1, gc)
	}
}

func (s *EditorScene) drawSelectedEnemyProps(r services.Renderer) {
	lg := s.markersLayer()
	if lg == nil || s.selObj < 0 || s.selObj >= len(lg.Objects) {
		return
	}
	o := &lg.Objects[s.selObj]
	if o.Type != "enemy" {
		return
	}
	cfg := world.EnemyConfigFromTiled(o)
	r.DrawText(4, 40, fmt.Sprintf(
		"enemy hp=%d spd=%.2f aggro=%.0f dmg=%d boss=%v",
		cfg.HP, cfg.Speed, cfg.AggroRadius, cfg.ContactDamage, cfg.IsBoss,
	))
}

func (s *EditorScene) drawMarkerOverlay(ctx GameContext) {
	r := ctx.Renderer()
	lg := s.markersLayer()
	if lg == nil {
		return
	}
	wx, wy := s.worldXY(ctx)
	hover := s.hitMarker(wx, wy)

	for i := range lg.Objects {
		rc := world.MarkerObjectHitRect(lg.Objects[i])
		col := color.RGBA{0x00, 0xff, 0xff, 0x55}
		if i == hover {
			r.StrokeRect(float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H), 2, color.RGBA{0x00, 0xff, 0xff, 0xcc})
		}
		if i == s.selObj {
			r.StrokeRect(float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H), 2, color.RGBA{0xff, 0xff, 0x00, 0xff})
		} else if i != hover {
			r.StrokeRect(float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H), 1, col)
		}
	}
}
