package scenes

import (
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
)

func (s *EditorScene) Update(ctx GameContext) error {
	in := ctx.Input()
	s.ensureLoaded(ctx)
	if s.savedFlash > 0 {
		s.savedFlash--
	}

	if in.JustPressed(services.ActionCancel) {
		ctx.Manager().Replace(SceneTitle, nil)
		return nil
	}

	if s.tm == nil {
		return nil
	}

	// Ctrl+S: save to disk (same chord the rest of the game uses).
	if in.IsModifierDown(services.ModCtrl) && in.JustPressed(services.ActionQuickSave) {
		if err := s.tm.WriteFile(s.path); err != nil {
			s.errMsg = err.Error()
		} else {
			s.errMsg = ""
			s.savedFlash = 90
		}
	}

	gids := world.RegisteredTileGIDs()

	// Tile selection menu input handling
	if s.showTileMenu {
		if in.JustPressed(services.ActionEditorTileMenu) || in.JustPressed(services.ActionCancel) {
			s.showTileMenu = false
			return nil
		}

		if in.JustPressed(services.ActionMoveUp) {
			s.tileMenuSelect--
			if s.tileMenuSelect < 0 {
				s.tileMenuSelect = len(gids) - 1
			}
			if s.tileMenuSelect < s.tileMenuScroll {
				s.tileMenuScroll = s.tileMenuSelect
			} else if s.tileMenuSelect >= s.tileMenuScroll+7 {
				s.tileMenuScroll = s.tileMenuSelect - 6
			}
		}
		if in.JustPressed(services.ActionMoveDown) {
			s.tileMenuSelect++
			if s.tileMenuSelect >= len(gids) {
				s.tileMenuSelect = 0
			}
			if s.tileMenuSelect < s.tileMenuScroll {
				s.tileMenuScroll = s.tileMenuSelect
			} else if s.tileMenuSelect >= s.tileMenuScroll+7 {
				s.tileMenuScroll = s.tileMenuSelect - 6
			}
		}

		if in.JustPressed(services.ActionConfirm) {
			s.brushGID = gids[s.tileMenuSelect]
			s.showTileMenu = false
			return nil
		}

		if in.MouseJustPressed(services.MouseLeft) {
			mx, my := in.CursorPosition()
			const panelX, panelY = 70, 30
			const panelW, panelH = 180, 180
			const headerH = 22
			const itemH = 20

			if mx >= panelX && mx < panelX+panelW && my >= panelY && my < panelY+panelH {
				listY := panelY + headerH
				if my >= listY && my < listY+140 {
					clickedRow := (my - listY) / itemH
					clickedIndex := s.tileMenuScroll + clickedRow
					if clickedIndex >= 0 && clickedIndex < len(gids) {
						s.tileMenuSelect = clickedIndex
						s.brushGID = gids[s.tileMenuSelect]
						s.showTileMenu = false
					}
				}
			} else {
				s.showTileMenu = false
			}
		}
		return nil
	}

	// Item selection menu input handling
	if s.showItemMenu {
		if in.JustPressed(services.ActionEditorTileMenu) || in.JustPressed(services.ActionCancel) {
			s.showItemMenu = false
			return nil
		}

		if in.JustPressed(services.ActionMoveUp) {
			s.itemMenuSelect--
			if s.itemMenuSelect < 0 {
				s.itemMenuSelect = len(world.AllPickups) - 1
			}
		}
		if in.JustPressed(services.ActionMoveDown) {
			s.itemMenuSelect++
			if s.itemMenuSelect >= len(world.AllPickups) {
				s.itemMenuSelect = 0
			}
		}

		if in.JustPressed(services.ActionConfirm) {
			s.selectItem(ctx, world.AllPickups[s.itemMenuSelect].TiledName())
			s.showItemMenu = false
			return nil
		}

		if in.JustPressed(services.ActionEditorAdd) {
			s.selectItem(ctx, world.AllPickups[s.itemMenuSelect].TiledName())
			s.showItemMenu = false
			wx, wy := s.worldXY(ctx)
			lg := s.markersLayer()
			if lg != nil {
				o := s.newMarkerObject(wx, wy)
				lg.Objects = append(lg.Objects, o)
				s.selObj = len(lg.Objects) - 1
				s.rebuild(ctx)
			}
			return nil
		}

		if in.MouseJustPressed(services.MouseLeft) {
			mx, my := in.CursorPosition()
			const panelX, panelY = 70, 30
			const panelW, panelH = 180, 180
			const headerH = 22
			const itemH = 20

			if mx >= panelX && mx < panelX+panelW && my >= panelY && my < panelY+panelH {
				listY := panelY + headerH
				pickups := world.AllPickups
				if my >= listY && my < listY+len(pickups)*itemH {
					clickedRow := (my - listY) / itemH
					if clickedRow >= 0 && clickedRow < len(pickups) {
						s.itemMenuSelect = clickedRow
						s.selectItem(ctx, world.AllPickups[s.itemMenuSelect].TiledName())
						s.showItemMenu = false
					}
				}
			} else {
				s.showItemMenu = false
			}
		}
		return nil
	}

	if s.modeTile && in.JustPressed(services.ActionEditorTileMenu) {
		s.showTileMenu = true
		s.tileMenuSelect = 0
		for idx, gid := range gids {
			if gid == s.brushGID {
				s.tileMenuSelect = idx
				break
			}
		}
		if s.tileMenuSelect < s.tileMenuScroll || s.tileMenuSelect >= s.tileMenuScroll+7 {
			s.tileMenuScroll = s.tileMenuSelect
			if s.tileMenuScroll > len(gids)-7 {
				s.tileMenuScroll = len(gids) - 7
			}
			if s.tileMenuScroll < 0 {
				s.tileMenuScroll = 0
			}
		}
		return nil
	}

	if !s.modeTile && in.JustPressed(services.ActionEditorTileMenu) {
		s.showItemMenu = true
		currentKind := s.activePickupKind
		lg := s.markersLayer()
		if lg != nil && s.selObj >= 0 && s.selObj < len(lg.Objects) {
			o := &lg.Objects[s.selObj]
			if o.Type == "pickup" {
				if k, ok := tiled.ObjProp(o, "kind"); ok {
					currentKind = k
				}
			}
		}
		for idx, item := range world.AllPickups {
			if item.TiledName() == currentKind {
				s.itemMenuSelect = idx
				break
			}
		}
		return nil
	}

	if in.JustPressed(services.ActionEditorToggleMode) {
		s.modeTile = !s.modeTile
		s.selObj = -1
		s.dragging = false
		s.showTileMenu = false
		s.showItemMenu = false
	}

	for i, gid := range editorBrushPalette {
		if i < len(editorBrushActions) && in.JustPressed(editorBrushActions[i]) {
			s.brushGID = gid
		}
	}
	if in.JustPressed(services.ActionEditorBrushClear) {
		s.brushGID = world.GIDEmpty
	}

	if in.JustPressed(services.ActionEditorPrevType) {
		types := world.MarkerTypeNames()
		s.markerTypeIndex = (s.markerTypeIndex + len(types) - 1) % len(types)
	}
	if in.JustPressed(services.ActionEditorNextType) {
		types := world.MarkerTypeNames()
		s.markerTypeIndex = (s.markerTypeIndex + 1) % len(types)
	}

	wx, wy := s.worldXY(ctx)

	if s.modeTile {
		if in.MouseJustPressed(services.MouseRight) {
			ground := s.tm.TileLayer("ground")
			if ground != nil && s.tm.Width > 0 {
				tx := int(wx) / world.TileSize
				ty := int(wy) / world.TileSize
				if tx >= 0 && ty >= 0 && tx < s.tm.Width && ty < s.tm.Height {
					idx := ty*s.tm.Width + tx
					if idx >= 0 && idx < len(ground.Data) {
						s.brushGID = ground.Data[idx]
					}
				}
			}
		}
		tx := int(wx) / world.TileSize
		ty := int(wy) / world.TileSize
		if in.MousePressed(services.MouseLeft) {
			if !s.painting || tx != s.lastPaintTX || ty != s.lastPaintTY {
				s.paintTile(tx, ty)
				s.lastPaintTX, s.lastPaintTY = tx, ty
				s.painting = true
				s.rebuild(ctx)
			}
		} else {
			s.painting = false
		}
	} else {
		s.painting = false
		lg := s.markersLayer()
		if lg == nil {
			return nil
		}

		if in.JustPressed(services.ActionEditorAdd) {
			o := s.newMarkerObject(wx, wy)
			lg.Objects = append(lg.Objects, o)
			s.selObj = len(lg.Objects) - 1
			s.rebuild(ctx)
		}

		if in.JustPressed(services.ActionEditorDelete) && s.selObj >= 0 && s.selObj < len(lg.Objects) {
			lg.Objects = append(lg.Objects[:s.selObj], lg.Objects[s.selObj+1:]...)
			s.selObj = -1
			s.dragging = false
			s.rebuild(ctx)
		}

		if in.MouseJustPressed(services.MouseLeft) {
			hit := s.hitMarker(wx, wy)
			if hit >= 0 {
				s.selObj = hit
				o := &lg.Objects[hit]
				s.dragging = true
				s.dragGrabDX = o.X - wx
				s.dragGrabDY = o.Y - wy
			} else {
				s.selObj = -1
				s.dragging = false
			}
		}
		if s.dragging && in.MousePressed(services.MouseLeft) && s.selObj >= 0 && s.selObj < len(lg.Objects) {
			o := &lg.Objects[s.selObj]
			o.X = wx + s.dragGrabDX
			o.Y = wy + s.dragGrabDY
			s.rebuild(ctx)
		}
		if in.MouseJustReleased(services.MouseLeft) {
			s.dragging = false
		}
	}

	return nil
}
