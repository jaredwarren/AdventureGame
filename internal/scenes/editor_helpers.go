package scenes

import (
	"fmt"
	"path/filepath"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

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
	s.ensureTileLayers()
	s.rebuild(ctx)
}

func (s *EditorScene) ensureTileLayers() {
	if s.tm == nil {
		return
	}
	var tileLayers []*tiled.Layer
	for i := range s.tm.Layers {
		if s.tm.Layers[i].Type == "tilelayer" {
			tileLayers = append(tileLayers, &s.tm.Layers[i])
		}
	}
	if len(tileLayers) < 2 {
		baseData := make([]int, s.tm.Width*s.tm.Height)
		for i := range baseData {
			baseData[i] = tile.GIDGrass
		}
		id := s.tm.NextLayerID
		if id == 0 {
			id = len(s.tm.Layers) + 1
		}
		s.tm.NextLayerID = id + 1

		baseLayer := tiled.Layer{
			ID:      id,
			Type:    "tilelayer",
			Name:    "base",
			Visible: true,
			Opacity: 1,
			Data:    baseData,
			Width:   s.tm.Width,
			Height:  s.tm.Height,
			X:       0,
			Y:       0,
		}
		s.tm.Layers = append([]tiled.Layer{baseLayer}, s.tm.Layers...)
		s.activeLayerIndex = 1
	}
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
		r := world.MarkerObjectHitRect(lg.Objects[i])
		if pointInRect(wx, wy, r) {
			return i
		}
	}
	return -1
}

func (s *EditorScene) worldXY(ctx GameContext) (float64, float64) {
	mx, my := ctx.Input().CursorPosition()
	cam := ctx.Renderer().Camera()
	return float64(mx) + cam.X, float64(my) + cam.Y
}

func (s *EditorScene) newMarkerObject(wx, wy float64) tiled.Object {
	id := s.tm.NextObjectID
	s.tm.NextObjectID++
	types := world.MarkerTypeNames()
	t := types[s.markerTypeIndex%len(types)]
	name := fmt.Sprintf("%s_%d", t, id)
	o := tiled.Object{ID: id, Name: name, Type: t, X: wx, Y: wy}
	world.InitMarkerObject(&o, wx, wy, world.MarkerEditorContext{
		TileWidth:             float64(s.tm.TileWidth),
		TileHeight:            float64(s.tm.TileHeight),
		ActivePickupTiledName: s.activePickupKind,
	})
	return o
}

func (s *EditorScene) filteredTileGIDs() []int {
	allGIDs := world.RegisteredTileGIDs()
	if !s.showOnlyActiveLayer {
		return allGIDs
	}
	var filtered []int
	for _, gid := range allGIDs {
		def := world.TileDefOf(gid)
		if s.activeLayerIndex == 0 {
			if def.IsFloor() || gid == tile.GIDEmpty {
				filtered = append(filtered, gid)
			}
		} else {
			if !def.IsFloor() || gid == tile.GIDEmpty {
				filtered = append(filtered, gid)
			}
		}
	}
	if len(filtered) == 0 {
		return allGIDs
	}
	return filtered
}

type editorMenuItem struct {
	isCategory bool
	category   string
	isBack     bool
	gid        int
	name       string
}

func (s *EditorScene) currentMenuItems() []editorMenuItem {
	// If we are in a sub-category:
	if s.tileMenuCategory != "" {
		var items []editorMenuItem
		items = append(items, editorMenuItem{isBack: true, name: ".. [Back]"})
		if s.tileMenuCategory == "water" {
			items = append(items, editorMenuItem{gid: tile.GIDWater, name: "water (base)"})
			for _, g := range []int{
				tile.GIDWaterShoreTop, tile.GIDWaterShoreBottom, tile.GIDWaterShoreLeft, tile.GIDWaterShoreRight,
				tile.GIDWaterShoreNE, tile.GIDWaterShoreNW, tile.GIDWaterShoreSW, tile.GIDWaterShoreSE,
				tile.GIDWaterShoreNEInner, tile.GIDWaterShoreNWInner, tile.GIDWaterShoreSWInner, tile.GIDWaterShoreSEInner,
			} {
				items = append(items, editorMenuItem{gid: g, name: world.TileDefOf(g).Name})
			}
		} else if s.tileMenuCategory == "wall" {
			items = append(items, editorMenuItem{gid: tile.GIDWall, name: "wall (base)"})
			for _, g := range []int{
				tile.GIDWallTop, tile.GIDWallBottom, tile.GIDWallLeft, tile.GIDWallRight,
				tile.GIDWallNE, tile.GIDWallNW, tile.GIDWallSW, tile.GIDWallSE,
				tile.GIDWallNEInner, tile.GIDWallNWInner, tile.GIDWallSWInner, tile.GIDWallSEInner,
			} {
				items = append(items, editorMenuItem{gid: g, name: world.TileDefOf(g).Name})
			}
		} else if s.tileMenuCategory == "rock" {
			items = append(items, editorMenuItem{gid: tile.GIDRock, name: "rock (base)"})
			for _, g := range []int{
				tile.GIDRockTop, tile.GIDRockBottom, tile.GIDRockLeft, tile.GIDRockRight,
				tile.GIDRockNE, tile.GIDRockNW, tile.GIDRockSW, tile.GIDRockSE,
				tile.GIDRockNEInner, tile.GIDRockNWInner, tile.GIDRockSWInner, tile.GIDRockSEInner,
			} {
				items = append(items, editorMenuItem{gid: g, name: world.TileDefOf(g).Name})
			}
		}
		return items
	}

	// Root menu items:
	var items []editorMenuItem
	allGIDs := s.filteredTileGIDs()

	hasWater := false
	hasWall := false
	hasRock := false

	for _, g := range allGIDs {
		isWaterGID := g == tile.GIDWater || (g >= tile.GIDWaterShoreTop && g <= tile.GIDWaterShoreSEInner)
		isWallGID := g == tile.GIDWall || (g >= tile.GIDWallTop && g <= tile.GIDWallSEInner)
		isRockGID := g == tile.GIDRock || (g >= tile.GIDRockTop && g <= tile.GIDRockSEInner)

		if isWaterGID {
			hasWater = true
		} else if isWallGID {
			hasWall = true
		} else if isRockGID {
			hasRock = true
		}
	}

	addedWallCategory := false
	addedWaterCategory := false
	addedRockCategory := false

	for _, g := range allGIDs {
		isWaterGID := g == tile.GIDWater || (g >= tile.GIDWaterShoreTop && g <= tile.GIDWaterShoreSEInner)
		isWallGID := g == tile.GIDWall || (g >= tile.GIDWallTop && g <= tile.GIDWallSEInner)
		isRockGID := g == tile.GIDRock || (g >= tile.GIDRockTop && g <= tile.GIDRockSEInner)

		if isWallGID {
			if hasWall && !addedWallCategory {
				items = append(items, editorMenuItem{isCategory: true, category: "wall", name: "wall >"})
				addedWallCategory = true
			}
		} else if isWaterGID {
			if hasWater && !addedWaterCategory {
				items = append(items, editorMenuItem{isCategory: true, category: "water", name: "water >"})
				addedWaterCategory = true
			}
		} else if isRockGID {
			if hasRock && !addedRockCategory {
				items = append(items, editorMenuItem{isCategory: true, category: "rock", name: "rock >"})
				addedRockCategory = true
			}
		} else {
			items = append(items, editorMenuItem{gid: g, name: world.TileDefOf(g).Name})
		}
	}

	return items
}

func (s *EditorScene) currentTileLayer() *tiled.Layer {
	if s.tm == nil {
		return nil
	}
	var tileLayers []*tiled.Layer
	for i := range s.tm.Layers {
		if s.tm.Layers[i].Type == "tilelayer" {
			tileLayers = append(tileLayers, &s.tm.Layers[i])
		}
	}
	if len(tileLayers) == 0 {
		return s.tm.TileLayer("ground")
	}
	idx := s.activeLayerIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= len(tileLayers) {
		idx = len(tileLayers) - 1
	}
	return tileLayers[idx]
}

func (s *EditorScene) paintTile(tx, ty int) {
	layer := s.currentTileLayer()
	if layer == nil || s.tm.Width <= 0 || s.tm.Height <= 0 {
		return
	}
	if tx < 0 || ty < 0 || tx >= s.tm.Width || ty >= s.tm.Height {
		return
	}
	idx := ty*s.tm.Width + tx
	if idx < 0 || idx >= len(layer.Data) {
		return
	}
	layer.Data[idx] = s.brushGID
}

func (s *EditorScene) selectItem(ctx GameContext, name string) {
	s.activePickupKind = name
	for idx, t := range world.MarkerTypeNames() {
		if t == "pickup" {
			s.markerTypeIndex = idx
			break
		}
	}
	lg := s.markersLayer()
	if lg != nil && s.selObj >= 0 && s.selObj < len(lg.Objects) {
		o := &lg.Objects[s.selObj]
		if o.Type == "pickup" {
			found := false
			for i, p := range o.Properties {
				if p.Name == "kind" {
					o.Properties[i].Value = name
					found = true
					break
				}
			}
			if !found {
				o.Properties = append(o.Properties, tiled.Property{
					Name:  "kind",
					Type:  "string",
					Value: name,
				})
			}
			s.rebuild(ctx)
		}
	}
}
