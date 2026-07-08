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

	if s.errMsg != "" && s.tm == nil {
		r.DrawText(0, 0, s.errMsg)
		return
	}

	if sess.World != nil {
		if s.modeTile && s.showOnlyActiveLayer {
			sess.World.ActiveLayerFilter = s.activeLayerIndex
		} else {
			sess.World.ActiveLayerFilter = -1
		}
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
		layerName := "Layer 0"
		if cur := s.currentTileLayer(); cur != nil && cur.Name != "" {
			layerName = cur.Name
		}
		viewState := "ONLY"
		if !s.showOnlyActiveLayer {
			viewState = "ALL"
		}
		mode = fmt.Sprintf("TILE L%d[%s] (%s) (brush: %s)", s.activeLayerIndex, layerName, viewState, world.TileDefOf(s.brushGID).Name)
	}
	line := fmt.Sprintf("%s map=%s", mode, s.mapID)
	if s.errMsg != "" {
		line += " ERR:" + s.errMsg
	}
	r.DrawText(4, 4, line)
	r.DrawText(4, 16, "E mode  L layer  V view  Tab menu  A add  Del")
	if !s.modeTile {
		types := world.MarkerTypeNames()
		r.DrawText(4, 28, fmt.Sprintf("new: %s  [ ] type  Ctrl+S save", types[s.markerTypeIndex%len(types)]))
		s.drawSelectedEnemyProps(r)
	} else {
		r.DrawText(4, 28, "Ctrl+S save  Esc title")
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
	cam := r.Camera()
	cx := float32(cam.X)
	cy := float32(cam.Y)
	gc := color.RGBA{0xff, 0xff, 0xff, 0x18}
	for tx := 0; tx <= w.MapW; tx++ {
		x := float32(tx*tile.Size) - cx
		r.StrokeLine(x, -cy, x, float32(w.MapH*tile.Size)-cy, 1, gc)
	}
	for ty := 0; ty <= w.MapH; ty++ {
		y := float32(ty*tile.Size) - cy
		r.StrokeLine(-cx, y, float32(w.MapW*tile.Size)-cx, y, 1, gc)
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
	cam := r.Camera()
	lg := s.markersLayer()
	if lg == nil {
		return
	}
	wx, wy := s.worldXY(ctx)
	hover := s.hitMarker(wx, wy)

	cx := float32(cam.X)
	cy := float32(cam.Y)

	for i := range lg.Objects {
		rc := world.MarkerObjectHitRect(lg.Objects[i])
		col := color.RGBA{0x00, 0xff, 0xff, 0x55}
		rx := float32(rc.X) - cx
		ry := float32(rc.Y) - cy
		rw := float32(rc.W)
		rh := float32(rc.H)

		if i == hover {
			r.StrokeRect(rx, ry, rw, rh, 2, color.RGBA{0x00, 0xff, 0xff, 0xcc})
		}
		if i == s.selObj {
			r.StrokeRect(rx, ry, rw, rh, 2, color.RGBA{0xff, 0xff, 0x00, 0xff})
		} else if i != hover {
			r.StrokeRect(rx, ry, rw, rh, 1, col)
		}
	}
}
