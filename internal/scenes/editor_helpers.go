package scenes

import (
	"fmt"
	"path/filepath"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
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
	return float64(mx), float64(my)
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
