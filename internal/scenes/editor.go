// editor.go — on-disk .tmj tile/marker editor.
//
// EditorScene reads from disk (not embedded assets) so changes are round-
// trippable. It holds a bit of mutable state (dragging, brush, selection)
// that is strictly scene-local; nothing else in the engine looks at it.
package scenes

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strconv"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
)

// EditorScene loads a .tmj from disk, edits ground + markers, previews via
// BuildFromTiled + Renderer.DrawWorld.
type EditorScene struct {
	// mapID is the logical map being edited; resolved to a .tmj path in
	// Enter. Provided by the CLI flag via Manager params.
	mapID string

	path   string
	tm     *tiled.Map
	errMsg string

	modeTile bool // false = marker mode

	brushGID        int
	markerTypeIndex int
	selObj          int // index in markers layer; -1 = none

	dragging   bool
	dragGrabDX float64
	dragGrabDY float64

	painting    bool
	lastPaintTX int
	lastPaintTY int

	savedFlash int

	showTileMenu   bool
	tileMenuSelect int
	tileMenuScroll int
}

func newEditorScene() Scene { return &EditorScene{} }

var (
	editorBrushPalette = []int{
		world.GIDGrass, world.GIDWall, world.GIDCracked, world.GIDDoor,
		world.GIDWater, world.GIDLock, world.GIDFloor2, world.GIDTree, world.GIDEmpty,
		world.GIDWaterShoreTop, world.GIDWaterShoreBottom, world.GIDWaterShoreLeft, world.GIDWaterShoreRight,
		world.GIDWaterShoreNE, world.GIDWaterShoreNW, world.GIDWaterShoreSW, world.GIDWaterShoreSE,
		world.GIDWaterShoreNEInner, world.GIDWaterShoreNWInner, world.GIDWaterShoreSWInner, world.GIDWaterShoreSEInner,
	}
	editorMarkerTypes = []string{"spawn", "enemy", "pickup", "door", "shrine"}
	// editorBrushActions aligns 1:1 with editorBrushPalette; index N selects palette[N].
	editorBrushActions = []services.Action{
		services.ActionEditorBrush1, services.ActionEditorBrush2,
		services.ActionEditorBrush3, services.ActionEditorBrush4,
		services.ActionEditorBrush5, services.ActionEditorBrush6,
		services.ActionEditorBrush7, services.ActionEditorBrush8,
	}
)

func (s *EditorScene) ID() SceneID { return SceneEditor }

func (s *EditorScene) Enter(ctx GameContext, params map[string]any) error {
	if params != nil {
		if v, ok := params["mapID"].(string); ok {
			s.mapID = v
		}
	}
	s.selObj = -1
	s.brushGID = world.GIDGrass
	s.showTileMenu = false
	s.tileMenuSelect = 0
	s.tileMenuScroll = 0
	return nil
}

func (s *EditorScene) Exit(ctx GameContext) error {
	ctx.Session().World = nil
	return nil
}

func (s *EditorScene) ensureLoaded(ctx GameContext) {
	if s.tm != nil {
		return
	}
	s.path = filepath.Join("assets", "maps", s.mapID+".tmj")
	tm, err := tiled.LoadMap(s.path)
	if err != nil {
		s.errMsg = err.Error()
		return
	}
	s.tm = tm
	s.rebuild(ctx)
}

func (s *EditorScene) markersLayer() *tiled.Layer {
	if s.tm == nil {
		return nil
	}
	lg := s.tm.ObjectGroupLayer("markers")
	if lg != nil {
		return lg
	}
	// Create markers group if missing (new maps).
	id := s.tm.NextLayerID
	if id == 0 {
		id = len(s.tm.Layers) + 1
	}
	s.tm.NextLayerID = id + 1
	s.tm.Layers = append(s.tm.Layers, tiled.Layer{
		ID:      id,
		Type:    "objectgroup",
		Name:    "markers",
		Visible: true,
		Opacity: 1,
		Objects: nil,
		X:       0,
		Y:       0,
	})
	return s.tm.ObjectGroupLayer("markers")
}

func (s *EditorScene) rebuild(ctx GameContext) {
	if s.tm == nil {
		return
	}
	w, err := world.BuildFromTiled(s.tm, s.mapID, progression.DefaultStats(), nil)
	if err != nil {
		s.errMsg = err.Error()
		ctx.Session().World = nil
		return
	}
	s.errMsg = ""
	w.IsEditor = true
	w.HasAmbientLightOverride = true
	w.AmbientLightOverride = 1.0
	w.TimeOfDay = 3000
	ctx.Session().World = w
	cam := ctx.Renderer().Camera()
	cam.X, cam.Y = 0, 0
	cam.ShakeTime = 0
}

func objectHitRect(o tiled.Object) geom.Rect {
	switch o.Type {
	case "door", "shrine":
		w, h := o.Width, o.Height
		if w <= 0 {
			w = 16
		}
		if h <= 0 {
			h = 16
		}
		return geom.Rect{X: o.X, Y: o.Y, W: w, H: h}
	case "spawn":
		return geom.Rect{X: o.X, Y: o.Y - 12, W: 12, H: 12}
	case "enemy":
		return geom.Rect{X: o.X, Y: o.Y - 12, W: 14, H: 12}
	case "pickup":
		return geom.Rect{X: o.X, Y: o.Y - 12, W: 12, H: 12}
	default:
		return geom.Rect{X: o.X, Y: o.Y, W: 16, H: 16}
	}
}

func pointInRect(px, py float64, r geom.Rect) bool {
	return px >= r.X && px < r.X+r.W && py >= r.Y && py < r.Y+r.H
}

func (s *EditorScene) hitMarker(wx, wy float64) int {
	lg := s.markersLayer()
	if lg == nil {
		return -1
	}
	for i := len(lg.Objects) - 1; i >= 0; i-- {
		r := objectHitRect(lg.Objects[i])
		if pointInRect(wx, wy, r) {
			return i
		}
	}
	return -1
}

func (s *EditorScene) worldXY(ctx GameContext) (float64, float64) {
	mx, my := ctx.Input().CursorPosition()
	return float64(mx), float64(my)
}



func (s *EditorScene) newMarkerObject(wx, wy float64) tiled.Object {
	id := s.tm.NextObjectID
	s.tm.NextObjectID++
	t := editorMarkerTypes[s.markerTypeIndex]
	name := fmt.Sprintf("%s_%d", t, id)
	o := tiled.Object{ID: id, Name: name, Type: t, X: wx, Y: wy}
	switch t {
	case "door":
		tw := float64(s.tm.TileWidth)
		if tw <= 0 {
			tw = world.TileSize
		}
		th := float64(s.tm.TileHeight)
		if th <= 0 {
			th = world.TileSize
		}
		o.X = float64(int(wx/tw) * int(tw))
		o.Y = float64(int(wy/th) * int(th))
		o.Width = 16
		o.Height = 32
		o.Properties = []tiled.Property{
			{Name: "target_map", Type: "string", Value: "field1"},
			{Name: "spawn_x", Type: "string", Value: strconv.Itoa(int(o.X))},
			{Name: "spawn_y", Type: "string", Value: strconv.Itoa(int(o.Y))},
		}
	case "shrine":
		o.Width = 16
		o.Height = 16
	case "pickup":
		o.Properties = []tiled.Property{
			{Name: "kind", Type: "string", Value: "coin"},
		}
	}
	return o
}

func (s *EditorScene) paintTile(tx, ty int) {
	ground := s.tm.TileLayer("ground")
	if ground == nil || s.tm.Width <= 0 || s.tm.Height <= 0 {
		return
	}
	if tx < 0 || ty < 0 || tx >= s.tm.Width || ty >= s.tm.Height {
		return
	}
	idx := ty*s.tm.Width + tx
	if idx < 0 || idx >= len(ground.Data) {
		return
	}
	ground.Data[idx] = s.brushGID
}

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

	if in.JustPressed(services.ActionEditorToggleMode) {
		s.modeTile = !s.modeTile
		s.selObj = -1
		s.dragging = false
		s.showTileMenu = false
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
		s.markerTypeIndex = (s.markerTypeIndex + len(editorMarkerTypes) - 1) % len(editorMarkerTypes)
	}
	if in.JustPressed(services.ActionEditorNextType) {
		s.markerTypeIndex = (s.markerTypeIndex + 1) % len(editorMarkerTypes)
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
		r.DrawText(4, 28, fmt.Sprintf("new: %s", editorMarkerTypes[s.markerTypeIndex]))
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
		x := float32(tx * world.TileSize)
		r.StrokeLine(x, 0, x, float32(w.MapH*world.TileSize), 1, gc)
	}
	for ty := 0; ty <= w.MapH; ty++ {
		y := float32(ty * world.TileSize)
		r.StrokeLine(0, y, float32(w.MapW*world.TileSize), y, 1, gc)
	}
}

func (s *EditorScene) drawTileMenu(ctx GameContext) {
	r := ctx.Renderer()
	gids := world.RegisteredTileGIDs()

	const panelX, panelY float32 = 70, 30
	const panelW, panelH float32 = 180, 180
	const headerH float32 = 22
	const footerH float32 = 18
	const rowH float32 = 20

	// Draw panel background (dark semi-transparent window)
	r.FillRect(panelX, panelY, panelW, panelH, color.RGBA{0x08, 0x0a, 0x14, 0xf0})
	r.StrokeRect(panelX, panelY, panelW, panelH, 1, color.RGBA{0x80, 0x80, 0xa0, 0xff})

	// Header Text
	r.DrawText(int(panelX)+50, int(panelY)+6, "SELECT TILE")

	// Divider below header
	r.StrokeLine(panelX+4, panelY+headerH-2, panelX+panelW-4, panelY+headerH-2, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Scrollable items
	visibleCount := 7
	for i := 0; i < visibleCount; i++ {
		idx := s.tileMenuScroll + i
		if idx >= len(gids) {
			break
		}
		gid := gids[idx]
		def := world.TileDefOf(gid)

		rowY := panelY + headerH + float32(i)*rowH

		// Highlight currently selected item in menu
		if idx == s.tileMenuSelect {
			r.FillRect(panelX+4, rowY+1, panelW-8, rowH-2, color.RGBA{0x2b, 0x4a, 0x8a, 0x80})
			r.StrokeRect(panelX+4, rowY+1, panelW-8, rowH-2, 1, color.RGBA{0x50, 0x80, 0xff, 0xff})
		}

		// Draw tile image fallback/sprite
		r.DrawTileScreen(gid, panelX+8, rowY+2, 16, 16)

		// Draw tile name text
		r.DrawText(int(panelX)+30, int(rowY)+4, def.Name)
	}

	// Draw scrollbar
	if len(gids) > visibleCount {
		trackX := panelX + panelW - 8
		trackY := panelY + headerH + 2
		trackH := float32(140) - 4
		trackW := float32(4)

		// Track
		r.FillRect(trackX, trackY, trackW, trackH, color.RGBA{0x20, 0x20, 0x30, 0xff})

		// Thumb
		thumbH := trackH * float32(visibleCount) / float32(len(gids))
		thumbY := trackY + trackH*float32(s.tileMenuScroll)/float32(len(gids))

		r.FillRect(trackX, thumbY, trackW, thumbH, color.RGBA{0x80, 0x80, 0xa0, 0xff})
	}

	// Divider above footer
	r.StrokeLine(panelX+4, panelY+panelH-footerH, panelX+panelW-4, panelY+panelH-footerH, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Footer hints
	r.DrawText(int(panelX)+10, int(panelY+panelH)-14, "Up/Dn scroll  Enter confirm")
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
		rc := objectHitRect(lg.Objects[i])
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
