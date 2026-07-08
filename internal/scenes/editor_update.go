package scenes

import (
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
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

	items := s.currentMenuItems()

	// Tile selection menu input handling
	if s.showTileMenu {
		if in.JustPressed(services.ActionEditorTileMenu) || in.JustPressed(services.ActionCancel) {
			s.showTileMenu = false
			return nil
		}

		if in.JustPressed(services.ActionMoveUp) {
			s.tileMenuSelect--
			if s.tileMenuSelect < 0 {
				s.tileMenuSelect = len(items) - 1
			}
			if s.tileMenuSelect < s.tileMenuScroll {
				s.tileMenuScroll = s.tileMenuSelect
			} else if s.tileMenuSelect >= s.tileMenuScroll+7 {
				s.tileMenuScroll = s.tileMenuSelect - 6
			}
		}
		if in.JustPressed(services.ActionMoveDown) {
			s.tileMenuSelect++
			if s.tileMenuSelect >= len(items) {
				s.tileMenuSelect = 0
			}
			if s.tileMenuSelect < s.tileMenuScroll {
				s.tileMenuScroll = s.tileMenuSelect
			} else if s.tileMenuSelect >= s.tileMenuScroll+7 {
				s.tileMenuScroll = s.tileMenuSelect - 6
			}
		}

		if in.JustPressed(services.ActionConfirm) {
			item := items[s.tileMenuSelect]
			if item.isCategory {
				s.tileMenuCategory = item.category
				s.tileMenuSelect = 0
				s.tileMenuScroll = 0
			} else if item.isBack {
				s.tileMenuCategory = ""
				s.tileMenuSelect = 0
				s.tileMenuScroll = 0
			} else {
				s.brushGID = item.gid
				s.showTileMenu = false
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
				if my >= listY && my < listY+140 {
					clickedRow := (my - listY) / itemH
					clickedIndex := s.tileMenuScroll + clickedRow
					if clickedIndex >= 0 && clickedIndex < len(items) {
						s.tileMenuSelect = clickedIndex
						item := items[s.tileMenuSelect]
						if item.isCategory {
							s.tileMenuCategory = item.category
							s.tileMenuSelect = 0
							s.tileMenuScroll = 0
						} else if item.isBack {
							s.tileMenuCategory = ""
							s.tileMenuSelect = 0
							s.tileMenuScroll = 0
						} else {
							s.brushGID = item.gid
							s.showTileMenu = false
						}
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

		// Determine category based on current brush GID
		isWaterGID := s.brushGID == tile.GIDWater || (s.brushGID >= tile.GIDWaterShoreTop && s.brushGID <= tile.GIDWaterShoreSEInner)
		isWallGID := s.brushGID == tile.GIDWall || (s.brushGID >= tile.GIDWallTop && s.brushGID <= tile.GIDWallSEInner)
		isRockGID := s.brushGID == tile.GIDRock || (s.brushGID >= tile.GIDRockTop && s.brushGID <= tile.GIDRockSEInner)
		if isWaterGID {
			s.tileMenuCategory = "water"
		} else if isWallGID {
			s.tileMenuCategory = "wall"
		} else if isRockGID {
			s.tileMenuCategory = "rock"
		} else {
			s.tileMenuCategory = ""
		}

		items := s.currentMenuItems()
		for idx, item := range items {
			if !item.isCategory && !item.isBack && item.gid == s.brushGID {
				s.tileMenuSelect = idx
				break
			}
		}
		if s.tileMenuSelect < s.tileMenuScroll || s.tileMenuSelect >= s.tileMenuScroll+7 {
			s.tileMenuScroll = s.tileMenuSelect
			if s.tileMenuScroll > len(items)-7 {
				s.tileMenuScroll = len(items) - 7
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

	if in.JustPressed(services.ActionEditorToggleLayer) {
		s.ensureTileLayers()
		var tileLayers []*tiled.Layer
		if s.tm != nil {
			for i := range s.tm.Layers {
				if s.tm.Layers[i].Type == "tilelayer" {
					tileLayers = append(tileLayers, &s.tm.Layers[i])
				}
			}
		}
		if len(tileLayers) > 0 {
			s.activeLayerIndex = (s.activeLayerIndex + 1) % len(tileLayers)
		}
	}

	if in.JustPressed(services.ActionEditorToggleVisibility) {
		s.showOnlyActiveLayer = !s.showOnlyActiveLayer
	}

	for i, gid := range editorBrushPalette {
		if i < len(editorBrushActions) && in.JustPressed(editorBrushActions[i]) {
			s.brushGID = gid
		}
	}
	if in.JustPressed(services.ActionEditorBrushClear) {
		s.brushGID = tile.GIDEmpty
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
			layer := s.currentTileLayer()
			if layer != nil && s.tm.Width > 0 {
				tx := int(wx) / tile.Size
				ty := int(wy) / tile.Size
				if tx >= 0 && ty >= 0 && tx < s.tm.Width && ty < s.tm.Height {
					idx := ty*s.tm.Width + tx
					if idx >= 0 && idx < len(layer.Data) {
						s.brushGID = layer.Data[idx]
					}
				}
			}
		}
		tx := int(wx) / tile.Size
		ty := int(wy) / tile.Size
		if in.MouseJustPressed(services.MouseLeft) {
			s.painting = true
			s.lastPaintTX, s.lastPaintTY = -1, -1
		}
		if in.MousePressed(services.MouseLeft) && s.painting {
			if tx != s.lastPaintTX || ty != s.lastPaintTY {
				s.paintTile(tx, ty)
				s.lastPaintTX, s.lastPaintTY = tx, ty
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

	// Camera panning when menus are closed
	if !s.showTileMenu && !s.showItemMenu {
		cam := ctx.Renderer().Camera()
		const scrollSpeed = 4.0
		if in.IsDown(services.ActionMoveUp) {
			cam.Y -= scrollSpeed
		}
		if in.IsDown(services.ActionMoveDown) {
			cam.Y += scrollSpeed
		}
		if in.IsDown(services.ActionMoveLeft) {
			cam.X -= scrollSpeed
		}
		if in.IsDown(services.ActionMoveRight) {
			cam.X += scrollSpeed
		}

		// Clamp camera to map boundaries to avoid scrolling past edges
		if s.tm != nil {
			maxCamX := float64(s.tm.Width*tile.Size) - 320.0
			maxCamY := float64(s.tm.Height*tile.Size) - 240.0
			if cam.X < 0 {
				cam.X = 0
			}
			if cam.Y < 0 {
				cam.Y = 0
			}
			if maxCamX > 0 && cam.X > maxCamX {
				cam.X = maxCamX
			}
			if maxCamY > 0 && cam.Y > maxCamY {
				cam.Y = maxCamY
			}
		}
	}

	return nil
}
